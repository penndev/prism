import Foundation

/// 主 App 和 Packet Tunnel 之间的流量、连接日志、启动错误。只扩展进程写计数和日志。
enum TunnelBridge {
    private static let upKey = "tunnel_up"
    private static let downKey = "tunnel_down"
    private static let errorKey = "tunnel_error"
    private static let stopReasonKey = "tunnel_stop_reason"
    private static let lock = NSLock()
    private static let logLimit = 1000
    private static let clock: DateFormatter = {
        let f = DateFormatter()
        f.locale = Locale(identifier: "en_US_POSIX")
        f.dateFormat = "HH:mm:ss.SSS"
        return f
    }()

    private static var logURL: URL {
        AppGroup.container.appendingPathComponent("connections.log")
    }

    private static var dnsURL: URL {
        AppGroup.container.appendingPathComponent("dns.log")
    }

    private static var statusURL: URL {
        AppGroup.container.appendingPathComponent("status.log")
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

    static func setStopReason(_ message: String) {
        let defaults = AppGroup.defaults
        defaults.set(message, forKey: stopReasonKey)
        defaults.synchronize()
    }

    static func takeStopReason() -> String? {
        let defaults = AppGroup.defaults
        defaults.synchronize()
        guard let value = defaults.string(forKey: stopReasonKey), !value.isEmpty else { return nil }
        defaults.removeObject(forKey: stopReasonKey)
        defaults.synchronize()
        return value
    }

    // 拨号发生在 append 返回之后。这里只放进内存，整文件重写放到每秒一次的 flush。
    private static var pendingLogs: [String] = []
    private static var pendingDns: [String] = []
    private static var pendingStatus: [String] = []
    private static var logTimer: DispatchSourceTimer?

    private enum LogKind { case connection, dns, status }

    static func appendConnection(_ line: String) {
        append(.connection, line)
    }

    static func appendDns(_ line: String) {
        append(.dns, line)
    }

    static func appendStatus(_ line: String) {
        append(.status, line)
    }

    private static func append(_ kind: LogKind, _ line: String) {
        let text = line.trimmingCharacters(in: .whitespacesAndNewlines)
        if text.isEmpty { return }
        if !Prefs().loadSettings().system.enableLogRecording { return }
        lock.lock()
        let stamped = "\(clock.string(from: Date())) \(text)"
        switch kind {
        case .connection: pendingLogs.append(stamped)
        case .dns: pendingDns.append(stamped)
        case .status: pendingStatus.append(stamped)
        }
        if logTimer == nil {
            let timer = DispatchSource.makeTimerSource(queue: .global(qos: .utility))
            timer.schedule(deadline: .now() + 1, repeating: 1)
            timer.setEventHandler {
                flushConnections()
                flushDns()
                flushStatus()
            }
            timer.resume()
            logTimer = timer
        }
        lock.unlock()
    }

    /// 把内存里攒的日志写入文件。停隧道前要调一次，不然最后一秒的日志会丢。
    static func flushConnections() {
        flush(.connection)
    }

    static func flushDns() {
        flush(.dns)
    }

    static func flushStatus() {
        flush(.status)
    }

    private static func flush(_ kind: LogKind) {
        lock.lock()
        defer { lock.unlock() }
        let url: URL
        let pending: [String]
        switch kind {
        case .connection:
            if pendingLogs.isEmpty { return }
            pending = pendingLogs
            pendingLogs.removeAll()
            url = logURL
        case .dns:
            if pendingDns.isEmpty { return }
            pending = pendingDns
            pendingDns.removeAll()
            url = dnsURL
        case .status:
            if pendingStatus.isEmpty { return }
            pending = pendingStatus
            pendingStatus.removeAll()
            url = statusURL
        }
        var lines = readLines(url)
        lines.append(contentsOf: pending)
        if lines.count > logLimit {
            lines = Array(lines.suffix(logLimit))
        }
        writeLines(lines, url)
    }

    static func connections() -> [String] {
        lock.lock()
        let lines = readLines(logURL)
        lock.unlock()
        return lines
    }

    static func dnsLogs() -> [String] {
        lock.lock()
        let lines = readLines(dnsURL)
        lock.unlock()
        return lines
    }

    static func statusLogs() -> [String] {
        lock.lock()
        let lines = readLines(statusURL)
        lock.unlock()
        return lines
    }

    static func clearConnections() {
        lock.lock()
        pendingLogs.removeAll()
        writeLines([], logURL)
        lock.unlock()
    }

    static func clearDns() {
        lock.lock()
        pendingDns.removeAll()
        writeLines([], dnsURL)
        lock.unlock()
    }

    static func clearStatus() {
        lock.lock()
        pendingStatus.removeAll()
        writeLines([], statusURL)
        lock.unlock()
    }

    // 每次 Read/Write 都会回调，直接写跨进程的 defaults 太重。先在内存里累加，每秒落一次盘。
    private static var pendingUp: Int64 = 0
    private static var pendingDown: Int64 = 0
    private static var flushTimer: DispatchSourceTimer?

    private static func add(_ key: String, _ n: Int64) {
        if n <= 0 { return }
        lock.lock()
        if key == upKey { pendingUp += n } else { pendingDown += n }
        if flushTimer == nil {
            let timer = DispatchSource.makeTimerSource(queue: .global(qos: .utility))
            timer.schedule(deadline: .now() + 1, repeating: 1)
            timer.setEventHandler { flushTraffic() }
            timer.resume()
            flushTimer = timer
        }
        lock.unlock()
    }

    /// 把内存里攒的流量写进 defaults。停隧道前要调一次，不然最后一秒的计数会丢。
    static func flushTraffic() {
        lock.lock()
        defer { lock.unlock() }
        if pendingUp == 0 && pendingDown == 0 { return }
        let defaults = AppGroup.defaults
        defaults.synchronize()
        defaults.set(NSNumber(value: number(defaults, upKey) + pendingUp), forKey: upKey)
        defaults.set(NSNumber(value: number(defaults, downKey) + pendingDown), forKey: downKey)
        defaults.synchronize()
        pendingUp = 0
        pendingDown = 0
    }

    private static func number(_ defaults: UserDefaults, _ key: String) -> Int64 {
        (defaults.object(forKey: key) as? NSNumber)?.int64Value ?? 0
    }

    private static func readLines(_ url: URL) -> [String] {
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

    private static func writeLines(_ lines: [String], _ url: URL) {
        let data = lines.joined(separator: "\n").data(using: .utf8) ?? Data()
        let coordinator = NSFileCoordinator()
        var error: NSError?
        coordinator.coordinate(writingItemAt: url, options: .forReplacing, error: &error) { item in
            try? data.write(to: item, options: .atomic)
        }
    }
}
