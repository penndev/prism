import Foundation
import Combine

private let logLimit = 1000

@MainActor
final class Store: ObservableObject {
    @Published var settings = AppSettings()
    @Published var servers: [ServerItem] = []
    @Published var selectedId: String?
    @Published var running = false
    @Published var vpnBusy = false
    @Published var vpnDesired = false
    @Published var pingingAll = false
    @Published var importing = false
    @Published var importPreview: [ServerItem] = []
    @Published var importSource: String?
    @Published var statusLogs: [String] = []
    @Published var connectionLogs: [String] = []
    @Published var traffic = TrafficUi()
    @Published var rules = RuleDraft()
    @Published var geoAreas: [AreaUi] = []
    @Published var dbStatus = ""
    @Published var dbReady = false
    @Published var dbBusy = false
    @Published var snackbar: String?

    var selectedServer: ServerItem? {
        servers.first { $0.id == selectedId }
    }

    var language: String { settings.system.language }

    private let prefs = Prefs()
    private var trafficTimer: Timer?

    init() {
        settings = prefs.loadSettings()
        servers = prefs.loadServers()
        selectedId = prefs.loadSelectedId()
        rules = prefs.loadRules()
        TunnelController.shared.rules = rules
        appendStatus(str("status_ready", servers.count))
        observeVpn()
        DispatchQueue.global(qos: .utility).async { [weak self] in
            openIpregionDb()
            DispatchQueue.main.async { self?.refreshEngineUi() }
        }
    }

    func t(_ key: String, _ args: CVarArg...) -> String {
        L10n.t(key, language: language, args: args)
    }

    func snack(_ key: String, _ args: CVarArg...) {
        snackbar = L10n.t(key, language: language, args: args)
    }

    func snackMessage(_ message: String) {
        snackbar = message
    }

    func consumeSnackbar() {
        snackbar = nil
    }

    func selectServer(_ server: ServerItem?) {
        selectedId = server?.id
        prefs.saveSelectedId(server?.id)
        if server == nil {
            if running { stop() }
            appendStatus(str("status_node_cleared"))
        } else {
            appendStatus(str("status_node_selected", args: [server!.displayName]))
        }
    }

    func start() {
        if vpnBusy { return }
        guard let server = selectedServer else {
            snack("proxy_need_node")
            return
        }
        vpnBusy = true
        vpnDesired = true
        TunnelController.shared.start(server: server)
        appendStatus(str("status_vpn_starting", args: [server.displayName]))
    }

    func stop() {
        vpnBusy = true
        vpnDesired = false
        TunnelController.shared.stop()
    }

    func pingAll() {
        let host = settings.latencyTest.host.trimmingCharacters(in: .whitespacesAndNewlines)
        if host.isEmpty {
            snack("settings_latency_test_host_required")
            return
        }
        if servers.isEmpty || pingingAll { return }
        pingingAll = true
        let items = servers
        Task.detached { [items, host] in
            await withTaskGroup(of: (String, Int).self) { group in
                for server in items {
                    group.addTask {
                        let ms = Engine.ping(server.toProxyURL(), host)
                        return (server.id, Int(ms))
                    }
                }
                for await (id, latency) in group {
                    await MainActor.run {
                        if let i = self.servers.firstIndex(where: { $0.id == id }) {
                            self.servers[i].latencyMs = latency
                        }
                    }
                }
            }
            await MainActor.run {
                if self.settings.latencyTest.sortAfterPing {
                    self.servers.sort(by: latencyLess)
                }
                self.prefs.saveServers(self.servers)
                self.pingingAll = false
                self.snack("server_list_ping_all_done")
            }
        }
    }

    func parseSubscription(type: String, source: String, fromFile: Bool = false) {
        let trimmed = source.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty {
            snack("subscribe_error_url")
            return
        }
        importing = true
        Task.detached {
            do {
                let isURL = !fromFile && (
                    trimmed.lowercased().hasPrefix("http://") || trimmed.lowercased().hasPrefix("https://")
                )
                let servers: [ServerItem]
                if isURL {
                    servers = try fetchSubscription(type: type, url: trimmed)
                } else {
                    servers = try SubscriptionParser.parseContent(type: type, raw: trimmed)
                }
                await MainActor.run {
                    self.importing = false
                    self.importPreview = servers
                    self.importSource = self.str(fromFile ? "subscribe_source_file" : "subscribe_source_url")
                    self.snack("subscribe_parsed", servers.count)
                }
            } catch {
                await MainActor.run {
                    self.importing = false
                    self.importPreview = []
                    self.importSource = nil
                    self.snackbar = self.errorMessage(error)
                }
            }
        }
    }

    func confirmImport() {
        if importPreview.isEmpty {
            snack("subscribe_error_no_preview")
            return
        }
        let preview = importPreview
        let selectedStill = preview.first { $0.id == selectedId }
        let shouldStop = selectedStill == nil && running
        prefs.saveServers(preview)
        prefs.saveSelectedId(selectedStill?.id)
        if shouldStop { stop() }
        servers = preview
        selectedId = selectedStill?.id
        importPreview = []
        importSource = nil
        snack("subscribe_imported", preview.count)
        appendStatus(str("status_imported", preview.count))
    }

    func exportServers() -> String {
        SubscriptionParser.exportJson(servers)
    }

    func addOrUpdateServer(
        editingId: String?,
        host: String,
        remark: String,
        protocolName: String,
        username: String,
        password: String,
    ) -> Bool {
        let trimmedHost = host.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmedHost.isEmpty {
            snack("server_list_validate_host")
            return false
        }
        if !hostMatches(trimmedHost) {
            snack("server_list_validate_host_format")
            return false
        }
        if protocolName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            snack("server_list_validate_protocol")
            return false
        }
        let payload = ServerItem(
            id: editingId ?? UUID().uuidString,
            host: trimmedHost,
            remark: remark.trimmingCharacters(in: .whitespacesAndNewlines),
            username: username.trimmingCharacters(in: .whitespacesAndNewlines),
            password: password,
            protocolName: protocolName,
        )
        var current = servers
        let nextId = SubscriptionParser.identity(
            host: payload.host,
            protocolName: payload.protocolName,
            username: payload.username,
            password: payload.password,
        )
        if current.contains(where: {
            $0.id != payload.id &&
                SubscriptionParser.identity(
                    host: $0.host,
                    protocolName: $0.protocolName,
                    username: $0.username,
                    password: $0.password,
                ) == nextId
        }) {
            snack("server_list_duplicate")
            return false
        }
        if let idx = current.firstIndex(where: { $0.id == payload.id }) {
            var updated = payload
            updated.latencyMs = current[idx].latencyMs
            current[idx] = updated
            snack("server_list_update_success")
        } else {
            current.append(payload)
            snack("server_list_add_success")
        }
        prefs.saveServers(current)
        servers = current
        if let editingId, selectedId == editingId {
            selectServer(current.first { $0.id == payload.id })
        }
        return true
    }

    func deleteServer(id: String) {
        let remain = servers.filter { $0.id != id }
        prefs.saveServers(remain)
        if selectedId == id { selectServer(nil) }
        servers = remain
        snack("server_list_delete_success")
    }

    func deleteServers(ids: Set<String>) {
        let remain = servers.filter { !ids.contains($0.id) }
        prefs.saveServers(remain)
        if let selectedId, ids.contains(selectedId) { selectServer(nil) }
        servers = remain
        snack("server_list_delete_success")
    }

    func updateLatencySettings(_ transform: (LatencyTestSettings) -> LatencyTestSettings) {
        updateSettings { s in
            var next = s
            next.latencyTest = transform(s.latencyTest)
            return next
        }
    }

    func updateSystemSettings(_ transform: (SystemSettings) -> SystemSettings) {
        updateSettings { s in
            var next = s
            next.system = transform(s.system)
            return next
        }
    }

    func clearStatusLogs() { statusLogs = [] }
    func clearConnectionLogs() { connectionLogs = [] }

    func updateRules(_ transform: (RuleDraft) -> RuleDraft) {
        rules = transform(rules)
        prefs.saveRules(rules)
        TunnelController.shared.rules = rules
    }

    func downloadDb() {
        let url = rules.dbUrl.trimmingCharacters(in: .whitespacesAndNewlines)
        if url.isEmpty {
            snack("rules_db_url_required")
            return
        }
        runDbJob(okKey: "rules_download_ok") {
            let dest = ipregionFile()
            let tmp = FileManager.default.temporaryDirectory.appendingPathComponent("ipregion-download.tmp")
            try downloadIpregionDb(url: url, dest: dest, tmp: tmp)
        }
    }

    func importDb(url: URL) {
        runDbJob(okKey: "rules_upload_ok") {
            let tmp = FileManager.default.temporaryDirectory.appendingPathComponent("ipregion-upload.tmp")
            let accessed = url.startAccessingSecurityScopedResource()
            defer { if accessed { url.stopAccessingSecurityScopedResource() } }
            try? FileManager.default.removeItem(at: tmp)
            do {
                try FileManager.default.copyItem(at: url, to: tmp)
            } catch {
                throw EngineError(L10n.raw("rules_upload_empty"))
            }
            try installIpregionDb(src: tmp, dest: ipregionFile())
            try? FileManager.default.removeItem(at: tmp)
        }
    }

    private func observeVpn() {
        let tunnel = TunnelController.shared
        tunnel.onRunning { [weak self] running in
            Task { @MainActor in
                self?.running = running
                self?.vpnBusy = false
                self?.vpnDesired = running
            }
        }
        tunnel.onStatus { [weak self] line in
            Task { @MainActor in
                self?.appendStatus(line)
            }
        }
        tunnel.onConnection { [weak self] line in
            Task { @MainActor in
                guard let self, self.settings.system.enableLogRecording else { return }
                self.connectionLogs = Array((self.connectionLogs + [line]).suffix(logLimit))
            }
        }
        trafficTimer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { [weak self] _ in
            Task { @MainActor in
                guard let self, self.running else { return }
                self.traffic = TunnelController.shared.trafficUi()
            }
        }
    }

    private func runDbJob(okKey: String, block: @escaping () throws -> Void) {
        if dbBusy { return }
        dbBusy = true
        Task.detached {
            let error: Error? = {
                do {
                    try block()
                    return nil
                } catch {
                    return error
                }
            }()
            await MainActor.run {
                if error == nil {
                    self.updateRules {
                        var next = $0
                        next.geoMode = .global
                        next.selectedAreaIds = []
                        return next
                    }
                }
            }
            await MainActor.run { self.refreshEngineUi() }
            await MainActor.run {
                self.dbBusy = false
                if let error {
                    let msg = error.localizedDescription
                    self.snackbar = msg.isEmpty ? self.str(okKey) : msg
                } else {
                    self.snack(okKey)
                }
            }
        }
    }

    private func updateSettings(_ transform: (AppSettings) -> AppSettings) {
        let next = transform(settings)
        prefs.saveSettings(next)
        settings = next
    }

    private func appendStatus(_ line: String) {
        statusLogs = Array((statusLogs + [line]).suffix(logLimit))
    }

    private func refreshEngineUi() {
        let st = Engine.dbStatus()
        dbReady = st.exists
        if !dbReady {
            dbStatus = str("rules_db_missing")
        } else {
            let ver = st.version.isEmpty ? "-" : st.version
            dbStatus = str("rules_db_status", ver, formatBytes(st.size), Int(st.areas))
        }
        geoAreas = dbReady ? loadAreaTree() : []
    }

    private func str(_ key: String, _ args: CVarArg...) -> String {
        L10n.t(key, language: language, args: args)
    }

    private func str(_ key: String, args: [CVarArg]) -> String {
        L10n.t(key, language: language, args: args)
    }

    private func errorMessage(_ error: Error) -> String {
        if let e = error as? SubscriptionError {
            if let n = e.intArg {
                return L10n.t(e.key, language: language, args: [n])
            }
            return L10n.t(e.key, language: language)
        }
        let msg = error.localizedDescription
        return msg.isEmpty ? str("subscribe_error_unknown") : msg
    }
}

private func fetchSubscription(type: String, url: String) throws -> [ServerItem] {
    let body: String
    do {
        body = try httpGet(url)
    } catch let error as SubscriptionError {
        throw error
    } catch {
        throw SubscriptionError("subscribe_error_fetch")
    }
    return try SubscriptionParser.parseContent(type: type, raw: body)
}

private func httpGet(_ url: String) throws -> String {
    guard let uri = URL(string: url), let scheme = uri.scheme?.lowercased(),
          scheme == "http" || scheme == "https"
    else { throw SubscriptionError("subscribe_error_fetch") }
    var request = URLRequest(url: uri, timeoutInterval: 15)
    request.setValue("Prism-iOS/0.1", forHTTPHeaderField: "User-Agent")
    let sem = DispatchSemaphore(value: 0)
    var result: Result<String, Error> = .failure(SubscriptionError("subscribe_error_fetch"))
    let task = URLSession.shared.dataTask(with: request) { data, response, error in
        defer { sem.signal() }
        if let error {
            result = .failure(error)
            return
        }
        let code = (response as? HTTPURLResponse)?.statusCode ?? 0
        let text = data.flatMap { SubscriptionParser.decodeText($0) } ?? ""
        if text.utf8.count > 2 * 1024 * 1024 {
            result = .failure(SubscriptionError("subscribe_error_fetch"))
            return
        }
        if !(200...299).contains(code) {
            result = .failure(SubscriptionError("subscribe_error_http", code))
            return
        }
        if text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            result = .failure(SubscriptionError("subscribe_error_empty"))
            return
        }
        result = .success(text)
    }
    task.resume()
    if sem.wait(timeout: .now() + 20) == .timedOut {
        task.cancel()
        throw SubscriptionError("subscribe_error_fetch")
    }
    return try result.get()
}

private func latencyLess(_ a: ServerItem, _ b: ServerItem) -> Bool {
    func rank(_ item: ServerItem) -> (Int, Int) {
        guard let latency = item.latencyMs else { return (2, Int.max) }
        if latency < 0 { return (1, Int.max) }
        return (0, latency)
    }
    let ra = rank(a)
    let rb = rank(b)
    if ra.0 != rb.0 { return ra.0 < rb.0 }
    return ra.1 < rb.1
}
