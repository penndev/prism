import Darwin
import NetworkExtension

class PacketTunnelProvider: NEPacketTunnelProvider {
    private var reading = false
    private var handler: TunnelHandler?

    override func startTunnel(options: [String: NSObject]?, completionHandler: @escaping (Error?) -> Void) {
        let proto = protocolConfiguration as? NETunnelProviderProtocol
        let proxy = proto?.providerConfiguration?["proxy"] as? String ?? ""
        let upstream = proto?.providerConfiguration?["upstream"] as? String ?? ""
        if proxy.isEmpty {
            TunnelBridge.setError("Could not create the VPN interface")
            completionHandler(tunnelError("Could not create the VPN interface"))
            return
        }

        let settings = NEPacketTunnelNetworkSettings(tunnelRemoteAddress: "127.0.0.1")
        settings.mtu = 1500
        let ipv4 = NEIPv4Settings(addresses: ["172.19.0.1"], subnetMasks: ["255.255.255.255"])
        ipv4.includedRoutes = [NEIPv4Route.default()]
        settings.ipv4Settings = ipv4
        // 把 IPv6 默认路由也收进隧道，否则系统会继续用局域网的 IPv6 DNS，查询进不了引擎。
        let ipv6 = NEIPv6Settings(addresses: ["fdfe:dcba:9876::1"], networkPrefixLengths: [NSNumber(value: 128)])
        ipv6.includedRoutes = [NEIPv6Route.default()]
        settings.ipv6Settings = ipv6
        // 8.8.8.8 会被系统升级成加密 DNS，不走 UDP/53。这里用只在隧道里可达的地址，并强制明文。
        let dns = NEDNSSettings(servers: ["172.19.0.2", "fdfe:dcba:9876::2"])
        dns.matchDomains = [""]
        dns.matchDomainsNoSearch = true
        settings.dnsSettings = dns

        setTunnelNetworkSettings(settings) { error in
            if error != nil {
                TunnelBridge.setError("Could not create the VPN interface")
                completionHandler(error)
                return
            }
            let handler = TunnelHandler(flow: self.packetFlow)
            self.handler = handler
            let opt = EngineOptions()
            opt.mtu = 1500
            opt.proxy = proxy
            opt.upstream = upstream
            opt.handler = handler
            do {
                try Engine.start(opt)
            } catch {
                TunnelBridge.setError("engine start: \(error.localizedDescription)")
                completionHandler(error)
                return
            }
            handler.ensureDb()
            let shown = upstream.trimmingCharacters(in: .whitespacesAndNewlines)
            TunnelBridge.appendDns("Upstream DNS \(shown.isEmpty ? "8.8.8.8" : shown)")
            self.reading = true
            self.readLoop()
            completionHandler(nil)
        }
    }

    override func stopTunnel(with reason: NEProviderStopReason, completionHandler: @escaping () -> Void) {
        reading = false
        TunnelBridge.setStopReason("\(stopReasonText(reason))(\(reason.rawValue))")
        Engine.stop()
        TunnelBridge.flushTraffic()
        TunnelBridge.flushConnections()
        TunnelBridge.flushDns()
        TunnelBridge.flushStatus()
        handler = nil
        completionHandler()
    }

    private func readLoop() {
        packetFlow.readPackets { [weak self] packets, _ in
            guard let self, self.reading else { return }
            if !packets.isEmpty {
                Engine.writePacket(packetBlob(packets))
            }
            self.readLoop()
        }
    }

    private func tunnelError(_ message: String) -> NSError {
        NSError(domain: "PrismTunnel", code: 1, userInfo: [NSLocalizedDescriptionKey: message])
    }
}

private func stopReasonText(_ reason: NEProviderStopReason) -> String {
    switch reason {
    case .none: return "none"
    case .userInitiated: return "userInitiated"
    case .providerFailed: return "providerFailed"
    case .noNetworkAvailable: return "noNetworkAvailable"
    case .unrecoverableNetworkChange: return "unrecoverableNetworkChange"
    case .providerDisabled: return "providerDisabled"
    case .authenticationCanceled: return "authenticationCanceled"
    case .configurationFailed: return "configurationFailed"
    case .idleTimeout: return "idleTimeout"
    case .configurationDisabled: return "configurationDisabled"
    case .configurationRemoved: return "configurationRemoved"
    case .superceded: return "superceded"
    case .userLogout: return "userLogout"
    case .userSwitch: return "userSwitch"
    case .connectionFailed: return "connectionFailed"
    case .sleep: return "sleep"
    case .appUpdate: return "appUpdate"
    @unknown default: return "unknown"
    }
}

private func packetBlob(_ packets: [Data]) -> Data {
    var blob = Data()
    var hdr = [UInt8](repeating: 0, count: 4)
    for pkt in packets where !pkt.isEmpty {
        let n = pkt.count
        hdr[0] = UInt8((n >> 24) & 0xff)
        hdr[1] = UInt8((n >> 16) & 0xff)
        hdr[2] = UInt8((n >> 8) & 0xff)
        hdr[3] = UInt8(n & 0xff)
        blob.append(contentsOf: hdr)
        blob.append(pkt)
    }
    return blob
}

private final class TunnelHandler: NSObject, EngineHandlerProtocol {
    private let flow: NEPacketTunnelFlow
    private let dbLock = NSLock()
    private var dbStamp: Date? = .distantPast

    init(flow: NEPacketTunnelFlow) {
        self.flow = flow
    }

    /// 库文件修改时间变了就重新打开。useProxy 会在多个 Go 线程上并发调用，只让一个去打开。
    /// 打开失败把标记退回，下次连接再试。
    func ensureDb() {
        let attrs = try? FileManager.default.attributesOfItem(atPath: ipregionFile().path)
        let stamp = attrs?[.modificationDate] as? Date
        dbLock.lock()
        guard let stamp, stamp != dbStamp else {
            dbLock.unlock()
            return
        }
        let previous = dbStamp
        dbStamp = stamp
        dbLock.unlock()
        if openIpregionDb() { return }
        dbLock.lock()
        if dbStamp == stamp { dbStamp = previous }
        dbLock.unlock()
    }

    func onStatus(_ line: String?) {
        let line = line ?? ""
        if !line.isEmpty { TunnelBridge.appendStatus(line) }
    }

    func onConnection(_ network: String?, address: String?, err: String?) {
        let address = address ?? ""
        let network = network ?? ""
        let err = err ?? ""
        if address.isEmpty { return }
        if !err.isEmpty {
            let head = network.isEmpty ? address : "\(network) \(address)"
            TunnelBridge.appendConnection("\(head): \(err)")
            return
        }
        let line = network.isEmpty ? "proxy \(address)" : "proxy \(network) \(address)"
        TunnelBridge.appendConnection(line)
    }

    func onDNS(_ name: String?, qtype: String?, result: String?) {
        let name = name ?? ""
        let qtype = qtype ?? ""
        let result = result ?? ""
        if name.isEmpty && qtype.isEmpty { return }
        var line = "\(name) \(qtype)"
        if !result.isEmpty { line += " \(result)" }
        TunnelBridge.appendDns(line)
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
        let address = address ?? ""
        let network = network ?? ""
        let rules = Prefs().loadRules()
        // 本地模式和全局模式结论是固定的，查 IP 库只是拖慢每一条新连接。
        let matched: Bool
        switch rules.geoMode {
        case .global, .none:
            matched = false
        case .proxy, .bypass:
            ensureDb()
            matched = inSelectedAreas(ipregionLookupIds(address), rules.selectedAreaIds)
        }
        let use = shouldProxy(rules.geoMode, matched)
        if !address.isEmpty {
            let tag = use ? "proxy" : "direct"
            let line = network.isEmpty ? "\(tag) \(address)" : "\(tag) \(network) \(address)"
            TunnelBridge.appendConnection(line)
        }
        return use
    }

    func onProxyRead(_ n: Int64) {
        TunnelBridge.addUpload(n)
    }

    func onProxyWrite(_ n: Int64) {
        TunnelBridge.addDownload(n)
    }

    func writePacket(_ blob: Data?) {
        guard let blob, blob.count >= 4 else { return }
        var packets: [Data] = []
        var protocols: [NSNumber] = []
        blob.withUnsafeBytes { raw in
            let bytes = raw.bindMemory(to: UInt8.self)
            var i = 0
            while i + 4 <= bytes.count {
                let n = (Int(bytes[i]) << 24) | (Int(bytes[i + 1]) << 16) | (Int(bytes[i + 2]) << 8) | Int(bytes[i + 3])
                i += 4
                guard n > 0, i + n <= bytes.count else { return }
                let pkt = Data(bytes[i..<(i + n)])
                let version = bytes[i] >> 4
                packets.append(pkt)
                protocols.append(version == 6 ? NSNumber(value: AF_INET6) : NSNumber(value: AF_INET))
                i += n
            }
        }
        if !packets.isEmpty {
            flow.writePackets(packets, withProtocols: protocols)
        }
    }

    private func shouldProxy(_ mode: GeoMode, _ matched: Bool) -> Bool {
        switch mode {
        case .global: return true
        case .none: return false
        case .proxy: return matched
        case .bypass: return !matched
        }
    }

    private func inSelectedAreas(_ lookedUp: [Int64], _ ids: Set<Int64>) -> Bool {
        if ids.isEmpty { return false }
        return lookedUp.contains { ids.contains($0) }
    }
}
