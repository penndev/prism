import Foundation

@_exported import Engine

struct EngineError: LocalizedError {
    let message: String
    init(_ message: String) { self.message = message }
    var errorDescription: String? { message }
}

/// 转发到 gomobile 生成的 `Engine.xcframework`。
/// Start 挂 gVisor；包经 `writePacket` / `Handler.writePacket` 进出。
enum Engine {
    static func start(_ opt: EngineOptions) throws {
        var err: NSError?
        if !EngineStart(opt, &err) {
            throw err ?? EngineError("start failed")
        }
    }

    static func stop() {
        EngineStop()
    }

    static func writePacket(_ pkt: Data?) {
        EngineWritePacket(pkt)
    }

    static func ping(_ proxy: String, _ latencyHost: String) -> Int64 {
        EnginePing(proxy, latencyHost)
    }

    static func setIpregionDB(_ path: String) throws {
        var err: NSError?
        if !EngineSetIpregionDB(path, &err) {
            throw err ?? EngineError("ipregion.db not found")
        }
    }

    static func dbStatus() -> EngineDbStatus {
        EngineDBStatus() ?? EngineDbStatus()
    }

    static func areaTree() -> String {
        EngineAreaTree()
    }

    static func lookup(_ address: String) -> [EngineArea] {
        guard let list = EngineLookup(address) else { return [] }
        var out: [EngineArea] = []
        let n = list.len()
        var i: Int64 = 0
        while i < n {
            if let item = list.get(i) {
                out.append(item)
            }
            i += 1
        }
        return out
    }
}
