import NetworkExtension

// 预留：主 App 暂不嵌入、不申请 VPN 权限。接 TUN 时再编进 Prism。
class PacketTunnelProvider: NEPacketTunnelProvider {

    override func startTunnel(options: [String: NSObject]?, completionHandler: @escaping (Error?) -> Void) {
        let proto = protocolConfiguration as? NETunnelProviderProtocol
        let proxy = proto?.providerConfiguration?["proxy"] as? String ?? ""
        let upstream = proto?.providerConfiguration?["upstream"] as? String ?? ""

        let settings = NEPacketTunnelNetworkSettings(tunnelRemoteAddress: "127.0.0.1")
        settings.mtu = 1500
        let ipv4 = NEIPv4Settings(addresses: ["172.19.0.1"], subnetMasks: ["255.255.255.255"])
        // 模拟阶段不劫持默认路由，避免流量进隧道后被丢掉。
        ipv4.includedRoutes = []
        settings.ipv4Settings = ipv4
        settings.dnsSettings = NEDNSSettings(servers: ["114.114.114.114"])

        setTunnelNetworkSettings(settings) { error in
            if let error {
                completionHandler(error)
                return
            }
            let opt = EngineOptions()
            opt.mtu = 1500
            opt.proxy = proxy
            opt.upstream = upstream
            opt.handler = TunnelHandler()
            do {
                try Engine.start(opt)
                completionHandler(nil)
            } catch {
                completionHandler(error)
            }
        }
    }

    override func stopTunnel(with reason: NEProviderStopReason, completionHandler: @escaping () -> Void) {
        Engine.stop()
        completionHandler()
    }
}

private final class TunnelHandler: NSObject, EngineHandlerProtocol {
    func onLog(_ line: String?) {
        if let line { NSLog("%@", line) }
    }

    func needFake(_ name: String?) -> Bool {
        var host = (name ?? "").trimmingCharacters(in: CharacterSet(charactersIn: ".")).lowercased()
        if host.isEmpty { return false }
        let domains = Prefs().loadRules().domains.filter { isDomainName($0) }
        if domains.isEmpty { return false }
        let set = Set(domains)
        while !host.isEmpty {
            if set.contains(host) { return true }
            guard let i = host.firstIndex(of: ".") else { return false }
            host = String(host[host.index(after: i)...])
        }
        return false
    }

    func useProxy(_ network: String?, address: String?) -> Bool {
        _ = network
        let rules = Prefs().loadRules()
        let address = address ?? ""
        switch rules.geoMode {
        case .global: return true
        case .none: return false
        case .proxy: return inSelectedAreas(address, rules.selectedAreaIds)
        case .bypass: return !inSelectedAreas(address, rules.selectedAreaIds)
        }
    }

    func onProxyRead(_ n: Int64) {}
    func onProxyWrite(_ n: Int64) {}

    private func inSelectedAreas(_ address: String, _ ids: Set<Int64>) -> Bool {
        if ids.isEmpty { return false }
        return Engine.lookup(address).contains { ids.contains($0.id_) }
    }
}
