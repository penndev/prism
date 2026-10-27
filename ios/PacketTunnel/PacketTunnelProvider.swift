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
        let dns = NEDNSSettings(servers: ["8.8.8.8"])
        dns.matchDomains = [""]
        settings.dnsSettings = dns

        setTunnelNetworkSettings(settings) { error in
            if error != nil {
                TunnelBridge.setError("Could not create the VPN interface")
                completionHandler(error)
                return
            }
            let handler = TunnelHandler(flow: self.packetFlow)
            handler.ensureDb()
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
            self.reading = true
            self.readLoop()
            completionHandler(nil)
        }
    }

    override func stopTunnel(with reason: NEProviderStopReason, completionHandler: @escaping () -> Void) {
        reading = false
        Engine.stop()
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
    private var dbStamp: Date? = .distantPast

    init(flow: NEPacketTunnelFlow) {
        self.flow = flow
    }

    func ensureDb() {
        let file = ipregionFile()
        let attrs = try? FileManager.default.attributesOfItem(atPath: file.path)
        let stamp = attrs?[.modificationDate] as? Date
        if stamp == dbStamp { return }
        dbStamp = stamp
        if stamp != nil { openIpregionDb() }
    }

    func onLog(_ line: String?) {
        if let line, !line.isEmpty { TunnelBridge.appendConnection(line) }
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
        ensureDb()
        let address = address ?? ""
        let network = network ?? ""
        let use = shouldProxy(address)
        let tag = use ? "proxy" : "direct"
        let line = network.isEmpty ? "\(tag) \(address)" : "\(tag) \(network) \(address)"
        if !address.isEmpty { TunnelBridge.appendConnection(line) }
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

    private func shouldProxy(_ address: String) -> Bool {
        let rules = Prefs().loadRules()
        switch rules.geoMode {
        case .global: return true
        case .none: return false
        case .proxy: return inSelectedAreas(address, rules.selectedAreaIds)
        case .bypass: return !inSelectedAreas(address, rules.selectedAreaIds)
        }
    }

    private func inSelectedAreas(_ address: String, _ ids: Set<Int64>) -> Bool {
        if ids.isEmpty { return false }
        return Engine.lookup(address).contains { ids.contains($0.id_) }
    }
}
