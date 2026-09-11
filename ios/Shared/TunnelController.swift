import Foundation

final class TunnelController {
    static let shared = TunnelController()

    // 接 Network Extension 时再用：com.penndev.prism.PacketTunnel
    static let providerBundleId = "com.penndev.prism.PacketTunnel"

    private let lock = NSLock()
    private var runningListeners: [(Bool) -> Void] = []
    private var statusListeners: [(String) -> Void] = []
    private var connectionListeners: [(String) -> Void] = []

    private(set) var running = false
    var session: ServerItem?
    var fakeDomains: Set<String> = []
    var rules = RuleDraft() {
        didSet {
            fakeDomains = Set(rules.domains.filter { isDomainName($0) })
        }
    }

    private var uploadBytes: Int64 = 0
    private var downloadBytes: Int64 = 0
    private var lastUp: Int64 = 0
    private var lastDown: Int64 = 0

    private init() {}

    func onRunning(_ block: @escaping (Bool) -> Void) {
        lock.lock()
        runningListeners.append(block)
        let value = running
        lock.unlock()
        block(value)
    }

    func onStatus(_ block: @escaping (String) -> Void) {
        lock.lock()
        statusListeners.append(block)
        lock.unlock()
    }

    func onConnection(_ block: @escaping (String) -> Void) {
        lock.lock()
        connectionListeners.append(block)
        lock.unlock()
    }

    func start(server: ServerItem) {
        session = server
        startInProcess(server: server)
    }

    func stop() {
        Engine.stop()
        markRunning(false)
        emitStatus(L10n.raw("status_stopped"))
    }

    func markRunning(_ value: Bool) {
        lock.lock()
        running = value
        if !value {
            lastUp = 0
            lastDown = 0
            uploadBytes = 0
            downloadBytes = 0
        }
        let listeners = runningListeners
        lock.unlock()
        DispatchQueue.main.async {
            listeners.forEach { $0(value) }
        }
    }

    func emitStatus(_ line: String) {
        lock.lock()
        let listeners = statusListeners
        lock.unlock()
        DispatchQueue.main.async {
            listeners.forEach { $0(line) }
        }
    }

    func emitConnection(_ line: String) {
        lock.lock()
        let listeners = connectionListeners
        lock.unlock()
        DispatchQueue.main.async {
            listeners.forEach { $0(line) }
        }
    }

    func trafficUi() -> TrafficUi {
        lock.lock()
        let up = uploadBytes
        let down = downloadBytes
        let upSpeed = max(up - lastUp, 0)
        let downSpeed = max(down - lastDown, 0)
        lastUp = up
        lastDown = down
        lock.unlock()
        return TrafficUi(
            downSpeed: "\(formatBytes(downSpeed))/s",
            upSpeed: "\(formatBytes(upSpeed))/s",
            downTotal: formatBytes(down),
            upTotal: formatBytes(up),
        )
    }

    func addUpload(_ n: Int64) {
        if n > 0 {
            lock.lock()
            uploadBytes += n
            lock.unlock()
        }
    }

    func addDownload(_ n: Int64) {
        if n > 0 {
            lock.lock()
            downloadBytes += n
            lock.unlock()
        }
    }

    /// 暂不申请系统 VPN 权限，只在进程内模拟 engine 会话。
    private func startInProcess(server: ServerItem) {
        let opt = EngineOptions()
        opt.mtu = 1500
        opt.proxy = server.toProxyURL()
        opt.upstream = ""
        opt.handler = InProcessHandler()
        lock.lock()
        uploadBytes = 0
        downloadBytes = 0
        lock.unlock()
        do {
            try Engine.start(opt)
            markRunning(true)
            emitStatus(L10n.raw("vpn_started"))
        } catch {
            markRunning(false)
            emitStatus("engine start: \(error.localizedDescription)")
        }
    }
}

private final class InProcessHandler: NSObject, EngineHandlerProtocol {
    func onLog(_ line: String?) {
        if let line, !line.isEmpty { TunnelController.shared.emitConnection(line) }
    }

    func needFake(_ name: String?) -> Bool {
        var host = (name ?? "").trimmingCharacters(in: CharacterSet(charactersIn: ".")).lowercased()
        if host.isEmpty { return false }
        let domains = TunnelController.shared.fakeDomains
        if domains.isEmpty { return false }
        while !host.isEmpty {
            if domains.contains(host) { return true }
            guard let i = host.firstIndex(of: ".") else { return false }
            host = String(host[host.index(after: i)...])
        }
        return false
    }

    func useProxy(_ network: String?, address: String?) -> Bool {
        let address = address ?? ""
        let network = network ?? ""
        let use = shouldProxy(address)
        let tag = use ? "proxy" : "direct"
        let line = network.isEmpty ? "\(tag) \(address)" : "\(tag) \(network) \(address)"
        if !address.isEmpty { TunnelController.shared.emitConnection(line) }
        return use
    }

    func onProxyRead(_ n: Int64) {
        TunnelController.shared.addUpload(n)
    }

    func onProxyWrite(_ n: Int64) {
        TunnelController.shared.addDownload(n)
    }

    private func shouldProxy(_ address: String) -> Bool {
        let rules = TunnelController.shared.rules
        switch rules.geoMode {
        case .global: return true
        case .none: return false
        case .proxy: return inSelectedAreas(address, rules.selectedAreaIds)
        case .bypass: return !inSelectedAreas(address, rules.selectedAreaIds)
        }
    }

    private func inSelectedAreas(_ address: String, _ ids: Set<Int64>) -> Bool {
        if ids.isEmpty { return false }
        let list = Engine.lookup(address)
        return list.contains { ids.contains($0.id_) }
    }
}
