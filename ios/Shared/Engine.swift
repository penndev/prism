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

    static func pingAsync(_ proxy: String, _ latencyHost: String, _ cb: (any EnginePingCallbackProtocol)?) {
        EnginePingAsync(proxy, latencyHost, cb)
    }
}
