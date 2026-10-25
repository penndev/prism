import Foundation

/// 主 App 和 Packet Tunnel 之间的流量、连接日志、启动错误。只扩展进程写计数和日志。
enum TunnelBridge {
    private static let upKey = "tunnel_up"
    private static let downKey = "tunnel_down"
    private static let errorKey = "tunnel_error"
    private static let lock = NSLock()
    private static let logLimit = 1000

    private static var logURL: URL {
        AppGroup.container.appendingPathComponent("connections.log")
    }

    static func resetTraffic() {
        let defaults = AppGroup.defaults
        defaults.set(0, forKey: upKey)
        defaults.set(0, forKey: downKey)
        defaults.synchronize()
    }

    static func addUpload(_ n: Int64) { add(upKey, n) }
    static func addDownload(_ n: Int64) { add(downKey, n) }

    static func counters() -> (up: Int64, down: Int64) {
        let defaults = AppGroup.defaults
        defaults.synchronize()
        return (number(defaults, upKey), number(defaults, downKey))
    }

    static func setError(_ message: String) {
        let defaults = AppGroup.defaults
        defaults.set(message, forKey: errorKey)
        defaults.synchronize()
    }

    static func takeError() -> String? {
        let defaults = AppGroup.defaults
        defaults.synchronize()
        guard let value = defaults.string(forKey: errorKey), !value.isEmpty else { return nil }
        defaults.removeObject(forKey: errorKey)
        defaults.synchronize()
        return value
    }

    static func appendConnection(_ line: String) {
        let text = line.trimmingCharacters(in: .whitespacesAndNewlines)
        if text.isEmpty { return }
        if !Prefs().loadSettings().system.enableLogRecording { return }
        lock.lock()
        var lines = readLines()
        lines.append(text)
        if lines.count > logLimit {
            lines = Array(lines.suffix(logLimit))
        }
        writeLines(lines)
        lock.unlock()
    }

    static func connections() -> [String] {
        lock.lock()
        let lines = readLines()
        lock.unlock()
        return lines
    }

    static func clearConnections() {
        lock.lock()
        writeLines([])
        lock.unlock()
    }

    private static func add(_ key: String, _ n: Int64) {
        if n <= 0 { return }
        lock.lock()
        let defaults = AppGroup.defaults
        defaults.synchronize()
        defaults.set(NSNumber(value: number(defaults, key) + n), forKey: key)
        defaults.synchronize()
        lock.unlock()
    }

    private static func number(_ defaults: UserDefaults, _ key: String) -> Int64 {
        (defaults.object(forKey: key) as? NSNumber)?.int64Value ?? 0
    }

    private static func readLines() -> [String] {
        let url = logURL
        if !FileManager.default.fileExists(atPath: url.path) { return [] }
        var lines: [String] = []
        let coordinator = NSFileCoordinator()
        var error: NSError?
        coordinator.coordinate(readingItemAt: url, options: [], error: &error) { item in
            guard let data = try? Data(contentsOf: item),
                  let text = String(data: data, encoding: .utf8)
            else { return }
            lines = text.split(separator: "\n", omittingEmptySubsequences: true).map(String.init)
        }
        return lines
    }

    private static func writeLines(_ lines: [String]) {
        let url = logURL
        let data = lines.joined(separator: "\n").data(using: .utf8) ?? Data()
        let coordinator = NSFileCoordinator()
        var error: NSError?
        coordinator.coordinate(writingItemAt: url, options: .forReplacing, error: &error) { item in
            try? data.write(to: item, options: .atomic)
        }
    }
}
