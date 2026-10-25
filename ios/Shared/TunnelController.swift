import Foundation
import NetworkExtension

final class TunnelController {
    static let shared = TunnelController()
    static let providerBundleId = "com.penndev.prism.PacketTunnel"

    private let lock = NSLock()
    private var runningListeners: [(Bool) -> Void] = []
    private var statusListeners: [(String) -> Void] = []

    private(set) var running = false
    var session: ServerItem?
    var fakeDomains: Set<String> = []
    var rules = RuleDraft() {
        didSet {
            fakeDomains = Set(rules.domains.filter { isDomainName($0) })
        }
    }

    private var manager: NETunnelProviderManager?
    private var wantRunning = false
    private var sawActive = false
    private var suppressStopStatus = false
    private var lastUp: Int64 = 0
    private var lastDown: Int64 = 0

    private init() {
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(vpnStatusChanged(_:)),
            name: .NEVPNStatusDidChange,
            object: nil,
        )
        Task { await reload() }
    }

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

    func start(server: ServerItem) {
        session = server
        wantRunning = true
        #if targetEnvironment(simulator)
        wantRunning = false
        markRunning(false)
        emitStatus("vpn_unavailable")
        #else
        let proxy = server.toProxyURL()
        let upstream = systemDnsIpv4()
        let address = server.host.isEmpty ? "Prism" : server.host
        Task { @MainActor in
            do {
                let manager = try await existingManager() ?? NETunnelProviderManager()
                let proto = (manager.protocolConfiguration as? NETunnelProviderProtocol) ?? NETunnelProviderProtocol()
                proto.providerBundleIdentifier = Self.providerBundleId
                proto.serverAddress = address
                proto.providerConfiguration = [
                    "proxy": proxy,
                    "upstream": upstream,
                ]
                proto.disconnectOnSleep = false
                manager.protocolConfiguration = proto
                manager.localizedDescription = "Prism"
                manager.isEnabled = true
                try await manager.saveToPreferences()
                try await manager.loadFromPreferences()
                self.manager = manager
                let status = manager.connection.status
                if status == .connected || status == .connecting || status == .reasserting {
                    manager.connection.stopVPNTunnel()
                    await waitUntilStopped(manager)
                }
                TunnelBridge.resetTraffic()
                try manager.connection.startVPNTunnel()
            } catch {
                wantRunning = false
                suppressStopStatus = true
                sawActive = false
                markRunning(false)
                emitStatus(message(for: error))
            }
        }
        #endif
    }

    func stop() {
        wantRunning = false
        Task {
            guard let manager = try? await existingManager() else {
                markRunning(false)
                emitStatus("status_stopped")
                return
            }
            self.manager = manager
            if manager.connection.status == .disconnected || manager.connection.status == .invalid {
                markRunning(false)
                emitStatus("status_stopped")
                return
            }
            manager.connection.stopVPNTunnel()
        }
    }

    func markRunning(_ value: Bool) {
        lock.lock()
        running = value
        if !value {
            lastUp = 0
            lastDown = 0
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

    func trafficUi() -> TrafficUi {
        let counts = TunnelBridge.counters()
        lock.lock()
        let upSpeed = max(counts.up - lastUp, 0)
        let downSpeed = max(counts.down - lastDown, 0)
        lastUp = counts.up
        lastDown = counts.down
        lock.unlock()
        return TrafficUi(
            downSpeed: "\(formatBytes(downSpeed))/s",
            upSpeed: "\(formatBytes(upSpeed))/s",
            downTotal: formatBytes(counts.down),
            upTotal: formatBytes(counts.up),
        )
    }

    @objc private func vpnStatusChanged(_ note: Notification) {
        guard let conn = note.object as? NEVPNConnection else { return }
        if let manager, conn !== manager.connection { return }
        if manager == nil { return }
        apply(conn.status, announce: true)
    }

    private func reload() async {
        manager = try? await existingManager()
        let status = manager?.connection.status ?? .disconnected
        if status == .connected || status == .connecting || status == .reasserting {
            sawActive = true
            wantRunning = status == .connected
            if status == .connected { markRunning(true) }
        }
    }

    private func existingManager() async throws -> NETunnelProviderManager? {
        let managers = try await NETunnelProviderManager.loadAllFromPreferences()
        return managers.first(where: isOurs)
    }

    private func isOurs(_ manager: NETunnelProviderManager) -> Bool {
        (manager.protocolConfiguration as? NETunnelProviderProtocol)?.providerBundleIdentifier == Self.providerBundleId
    }

    private func waitUntilStopped(_ manager: NETunnelProviderManager) async {
        for _ in 0..<50 {
            let status = manager.connection.status
            if status == .disconnected || status == .invalid { return }
            try? await Task.sleep(nanoseconds: 100_000_000)
        }
    }

    private func apply(_ status: NEVPNStatus, announce: Bool) {
        switch status {
        case .connected:
            sawActive = true
            wantRunning = true
            markRunning(true)
            if announce { emitStatus("vpn_started") }
        case .connecting, .reasserting, .disconnecting:
            sawActive = true
        case .disconnected, .invalid:
            if suppressStopStatus {
                suppressStopStatus = false
                sawActive = false
                markRunning(false)
                return
            }
            if let err = TunnelBridge.takeError() {
                wantRunning = false
                sawActive = false
                markRunning(false)
                if announce { emitStatus(err) }
                return
            }
            if wantRunning { return }
            let notify = announce && sawActive
            sawActive = false
            markRunning(false)
            if notify { emitStatus("status_stopped") }
        default:
            break
        }
    }

    private func message(for error: Error) -> String {
        let ns = error as NSError
        if ns.domain == NEVPNErrorDomain && ns.code == NEVPNError.Code.configurationDisabled.rawValue {
            return "vpn_permission_denied"
        }
        if ns.domain == "NEConfigurationErrorDomain" && ns.code == 11 {
            return "vpn_unavailable"
        }
        if ns.domain == "NEConfigurationErrorDomain" {
            return "vpn_permission_denied"
        }
        let text = ns.localizedDescription.trimmingCharacters(in: .whitespacesAndNewlines)
        if text.isEmpty { return "vpn_establish_failed" }
        return text
    }
}

@_silgen_name("prism_system_dns_ipv4")
private func prismSystemDnsIpv4() -> UnsafePointer<CChar>

private func systemDnsIpv4() -> String {
    String(cString: prismSystemDnsIpv4())
}
