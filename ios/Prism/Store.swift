import Foundation
import Combine

private let logLimit = 1000
private let logClock: DateFormatter = {
    let f = DateFormatter()
    f.locale = Locale(identifier: "en_US_POSIX")
    f.dateFormat = "HH:mm:ss.SSS"
    return f
}()

@MainActor
final class Store: ObservableObject {
    @Published var settings = AppSettings()
    @Published var servers: [ServerItem] = []
    @Published var selectedId: String?
    @Published var running = false
    @Published var vpnBusy = false
    @Published var vpnDesired = false
    @Published var pingingGroup: String?
    @Published var subscription: Subscription?
    @Published var importing = false
    @Published var importPreview: [ServerItem] = []
    @Published var importSource: String?
    @Published var statusLogs: [String] = []
    @Published var connectionLogs: [String] = []
    @Published var dnsLogs: [String] = []
    @Published var traffic = TrafficUi()
    @Published var rules = RuleDraft()
    @Published var geoAreas: [AreaUi] = []
    @Published var dbStatus = ""
    @Published var dbReady = false
    @Published var dbBusy = false
    @Published var snackbar: String?
    private var pingSinks: [PingSink] = []
    private var engineStatus: [String] = []

    var selectedServer: ServerItem? {
        servers.first { $0.id == selectedId }
    }

    var language: String { settings.system.language }

    private let prefs = Prefs()
    private var trafficTimer: Timer?

    init() {
        settings = prefs.loadSettings()
        servers = prefs.loadServers()
        subscription = prefs.loadSubscription()
        selectedId = prefs.loadSelectedId()
        rules = prefs.loadRules()
        TunnelController.shared.rules = rules
        // 日志文件只给扩展和主 App 传记录。划掉进程再打开时不带上一次的内容。
        TunnelBridge.clearConnections()
        TunnelBridge.clearDns()
        TunnelBridge.clearStatus()
        appendStatus("Loaded \(servers.count) nodes")
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
        let previous = selectedId
        selectedId = server?.id
        prefs.saveSelectedId(server?.id)
        if let server {
            appendStatus("Selected \(server.displayName)")
            if server.id != previous && running { start() }
        } else {
            if running { stop() }
            appendStatus("Current node cleared")
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
        appendStatus("Starting VPN (\(server.displayName))")
    }

    func stop() {
        vpnBusy = true
        vpnDesired = false
        TunnelController.shared.stop()
    }

    func pingGroup(_ subscriptionId: String?) {
        let targets = servers.filter { $0.subscriptionId == subscriptionId }
        pingTargets(
            key: subscriptionId ?? pingManual,
            targets: targets,
            subscriptionId: subscriptionId,
        )
    }

    func pingServer(id: String) {
        guard let server = servers.first(where: { $0.id == id }) else { return }
        pingTargets(key: "server:\(id)", targets: [server], subscriptionId: server.subscriptionId)
    }

    private func pingTargets(key: String, targets: [ServerItem], subscriptionId: String?) {
        let host = settings.latencyTest.host.trimmingCharacters(in: .whitespacesAndNewlines)
        if host.isEmpty {
            snack("settings_latency_test_host_required")
            return
        }
        if targets.isEmpty || pingingGroup != nil { return }
        pingingGroup = key
        let sort = settings.latencyTest.sortAfterPing
        let total = targets.count
        var finished = 0
        pingSinks = targets.map { server in
            let sink = PingSink { ms in
                if let i = self.servers.firstIndex(where: { $0.id == server.id }) {
                    self.servers[i].latencyMs = Int(ms)
                }
                finished += 1
                if finished == total {
                    self.finishPing(sort: sort, subscriptionId: subscriptionId)
                }
            }
            Engine.pingAsync(server.toProxyURL(), host, sink)
            return sink
        }
    }

    private func finishPing(sort: Bool, subscriptionId: String?) {
        var next = servers
        if sort {
            let manual = next.filter { $0.subscriptionId == nil }
            let subNodes = next.filter { $0.subscriptionId != nil }
            next = subscriptionId == nil
                ? manual.sorted(by: latencyLess) + subNodes
                : manual + subNodes.sorted(by: latencyLess)
        }
        prefs.saveServers(next)
        servers = next
        pingingGroup = nil
        pingSinks = []
    }

    func parseSubscription(type: String, source: String) {
        let trimmed = source.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty {
            snack("subscribe_error_url")
            return
        }
        importing = true
        Task.detached {
            do {
                let isURL = trimmed.lowercased().hasPrefix("http://") || trimmed.lowercased().hasPrefix("https://")
                let servers: [ServerItem]
                if isURL {
                    servers = try fetchSubscription(type: type, url: trimmed)
                } else {
                    servers = try SubscriptionParser.parseContent(type: type, raw: trimmed)
                }
                await MainActor.run {
                    self.importing = false
                    self.importPreview = servers
                    self.importSource = self.str("subscribe_source_url")
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

    func updateSubscriptionInfo(type: String, url: String, remark: String) {
        guard let current = subscription else { return }
        let next = Subscription(
            id: current.id,
            type: type.trimmingCharacters(in: .whitespacesAndNewlines),
            url: url.trimmingCharacters(in: .whitespacesAndNewlines),
            remark: remark.trimmingCharacters(in: .whitespacesAndNewlines),
        )
        prefs.saveSubscription(next)
        subscription = next
        snack("subscribe_saved")
    }

    func confirmImport(type: String, url: String, remark: String) {
        if importPreview.isEmpty {
            snack("subscribe_error_no_preview")
            return
        }
        saveSubscriptionNodes(
            Subscription(
                type: type.trimmingCharacters(in: .whitespacesAndNewlines),
                url: url.trimmingCharacters(in: .whitespacesAndNewlines),
                remark: remark.trimmingCharacters(in: .whitespacesAndNewlines),
            ),
            preview: importPreview,
        )
    }

    func refreshSubscription() {
        guard let sub = subscription else { return }
        let source = sub.url.trimmingCharacters(in: .whitespacesAndNewlines)
        if source.isEmpty || importing { return }
        importing = true
        let type = sub.type
        Task.detached {
            do {
                let isURL = source.lowercased().hasPrefix("http://") || source.lowercased().hasPrefix("https://")
                let servers = try isURL
                    ? fetchSubscription(type: type, url: source)
                    : SubscriptionParser.parseContent(type: type, raw: source)
                await MainActor.run {
                    self.saveSubscriptionNodes(sub, preview: servers)
                }
            } catch {
                await MainActor.run {
                    self.importing = false
                    self.snackbar = self.errorMessage(error)
                }
            }
        }
    }

    func deleteSubscription() {
        guard let id = subscription?.id else { return }
        let selectedInSub = selectedServer?.subscriptionId == id
        let remain = servers.filter { $0.subscriptionId != id }
        prefs.clearSubscription()
        prefs.saveServers(remain)
        servers = remain
        subscription = nil
        importPreview = []
        importSource = nil
        if selectedInSub { selectServer(nil) }
    }

    private func saveSubscriptionNodes(_ subscription: Subscription, preview: [ServerItem]) {
        let tagged = preview.map { node -> ServerItem in
            var next = node
            next.subscriptionId = subscription.id
            return next
        }
        let manual = servers.filter { $0.subscriptionId == nil }
        let merged = manual + tagged
        let selectedStill = merged.first { $0.id == selectedId }
        let shouldStop = selectedStill == nil && running
        prefs.saveSubscription(subscription)
        prefs.saveServers(merged)
        prefs.saveSelectedId(selectedStill?.id)
        if shouldStop { stop() }
        servers = merged
        self.subscription = subscription
        selectedId = selectedStill?.id
        importing = false
        importPreview = []
        importSource = nil
        snack("subscribe_imported", preview.count)
        appendStatus("Imported \(preview.count) nodes")
    }

    func importManualNodes(_ raw: String, onAdded: @escaping () -> Void) {
        let text = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        if text.isEmpty {
            snack("subscribe_error_empty")
            return
        }
        Task.detached {
            do {
                let incoming = try text.hasPrefix("{")
                    ? [SubscriptionParser.parseOne(text)]
                    : SubscriptionParser.parseContent(type: "prism", raw: text)
                await MainActor.run {
                    var current = self.servers
                    var added = 0
                    for node in incoming {
                        let id = SubscriptionParser.identity(
                            host: node.host,
                            protocolName: node.protocolName,
                            username: node.username,
                            password: node.password,
                        )
                        let exists = current.contains {
                            SubscriptionParser.identity(
                                host: $0.host,
                                protocolName: $0.protocolName,
                                username: $0.username,
                                password: $0.password,
                            ) == id
                        }
                        if exists { continue }
                        var manual = node
                        manual.id = id
                        manual.subscriptionId = nil
                        current.append(manual)
                        added += 1
                    }
                    if added == 0 {
                        self.snack("server_list_duplicate")
                        return
                    }
                    self.prefs.saveServers(current)
                    self.servers = current
                    self.snack("server_list_imported", added)
                    onAdded()
                }
            } catch {
                await MainActor.run {
                    self.snackbar = self.errorMessage(error)
                }
            }
        }
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
        var current = servers
        let payload = ServerItem(
            id: editingId ?? UUID().uuidString,
            host: trimmedHost,
            remark: remark.trimmingCharacters(in: .whitespacesAndNewlines),
            username: username.trimmingCharacters(in: .whitespacesAndNewlines),
            password: password,
            protocolName: protocolName,
            subscriptionId: editingId.flatMap { id in current.first { $0.id == id }?.subscriptionId },
        )
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

    func clearStatusLogs() {
        statusLogs = []
        engineStatus = []
        TunnelBridge.clearStatus()
    }
    func clearConnectionLogs() {
        connectionLogs = []
        TunnelBridge.clearConnections()
    }

    func clearDnsLogs() {
        dnsLogs = []
        TunnelBridge.clearDns()
    }

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
                guard let self else { return }
                self.appendStatus(line)
            }
        }
        trafficTimer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { [weak self] _ in
            Task { @MainActor in
                guard let self else { return }
                if self.running {
                    self.traffic = TunnelController.shared.trafficUi()
                }
                if self.settings.system.enableLogRecording {
                    let lines = TunnelBridge.connections()
                    if lines != self.connectionLogs { self.connectionLogs = lines }
                    let dns = TunnelBridge.dnsLogs()
                    if dns != self.dnsLogs { self.dnsLogs = dns }
                    let status = TunnelBridge.statusLogs()
                    if status != self.engineStatus {
                        var added = status
                        if let first = status.first, let k = self.engineStatus.firstIndex(of: first) {
                            let tail = self.engineStatus.count - k
                            if tail <= status.count && Array(self.engineStatus[k...]) == Array(status[..<tail]) {
                                added = Array(status[tail...])
                            }
                        }
                        self.engineStatus = status
                        if !added.isEmpty {
                            self.statusLogs = Array((self.statusLogs + added).suffix(logLimit))
                        }
                    }
                }
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
        statusLogs = Array((statusLogs + ["\(logClock.string(from: Date())) \(line)"]).suffix(logLimit))
    }

    private func refreshEngineUi() {
        let st = ipregionStatus()
        dbReady = st.exists
        if !dbReady {
            dbStatus = str("rules_db_missing")
        } else {
            let ver = st.version.isEmpty ? "-" : st.version
            dbStatus = str("rules_db_status", ver, formatBytes(st.size), st.areas)
        }
        geoAreas = dbReady ? loadAreaTree() : []
    }

    private func str(_ key: String, _ args: CVarArg...) -> String {
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
    let kind = type.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
    let userAgent: String?
    switch kind {
    case "shadowrocket":
        userAgent = "shadowrocket/*"
    case "prism":
        let version = (Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String)?
            .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        userAgent = version.isEmpty ? "prism/*" : "prism/\(version)"
    default:
        userAgent = nil
    }
    let body: String
    do {
        body = try httpGet(url, userAgent: userAgent)
    } catch let error as SubscriptionError {
        throw error
    } catch {
        throw SubscriptionError("subscribe_error_fetch")
    }
    return try SubscriptionParser.parseContent(type: type, raw: body)
}

private func httpGet(_ url: String, userAgent: String?) throws -> String {
    guard let uri = URL(string: url), let scheme = uri.scheme?.lowercased(),
          scheme == "http" || scheme == "https"
    else { throw SubscriptionError("subscribe_error_fetch") }
    var request = URLRequest(url: uri, timeoutInterval: 15)
    if let userAgent, !userAgent.isEmpty {
        request.setValue(userAgent, forHTTPHeaderField: "User-Agent")
    }
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

let pingManual = "manual"

private final class PingSink: NSObject, EnginePingCallbackProtocol {
    private let done: (Int64) -> Void

    init(done: @escaping (Int64) -> Void) {
        self.done = done
    }

    func onResult(_ ms: Int64) {
        let done = done
        DispatchQueue.main.async {
            done(ms)
        }
    }
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
