import Foundation

struct ServerItem: Identifiable, Hashable, Codable {
    var id: String
    var host: String
    var remark: String = ""
    var username: String = ""
    var password: String = ""
    var protocolName: String = "socks5"
    var latencyMs: Int? = nil

    var displayName: String { remark.isEmpty ? host : remark }

    enum CodingKeys: String, CodingKey {
        case id, host, remark, username, password, protocolName = "protocol", latencyMs
    }

    func toProxyURL() -> String {
        let scheme = protocolName.lowercased()
        let userinfo: String
        if username.isEmpty && password.isEmpty {
            userinfo = ""
        } else {
            userinfo = "\(encodeUserinfo(username)):\(encodeUserinfo(password))@"
        }
        return "\(scheme)://\(userinfo)\(host)"
    }
}

enum ThemeMode: String, Codable, CaseIterable {
    case system = "System"
    case light = "Light"
    case dark = "Dark"
}

enum GeoMode: String, Codable, CaseIterable {
    case global = "Global"
    case none = "None"
    case proxy = "Proxy"
    case bypass = "Bypass"
}

struct LatencyTestSettings: Codable, Equatable {
    var host: String = "google.com"
    var sortAfterPing: Bool = true
}

let languageSystem = "system"

struct SystemSettings: Codable, Equatable {
    var language: String = languageSystem
    var themeMode: ThemeMode = .system
    var enableLogRecording: Bool = true
}

struct AppSettings: Codable, Equatable {
    var latencyTest: LatencyTestSettings = LatencyTestSettings()
    var system: SystemSettings = SystemSettings()
}

struct AreaUi: Identifiable, Hashable {
    var id: Int64
    var parentId: Int64
    var name: String
    var children: [AreaUi] = []
}

struct RuleDraft: Equatable {
    var geoMode: GeoMode = .global
    var selectedAreaIds: Set<Int64> = []
    var dbUrl: String = ""
    var domains: [String] = []
}

struct TrafficUi: Equatable {
    var downSpeed: String = "0 B/s"
    var upSpeed: String = "0 B/s"
    var downTotal: String = "0 B"
    var upTotal: String = "0 B"
}

let proxySchemes = ["socks5", "socks5s", "http", "https"]

let hostPattern = try! NSRegularExpression(pattern: #"^(\[[^\]]+]|[^:\[\]]+):\d{1,5}$"#)

func hostMatches(_ host: String) -> Bool {
    let range = NSRange(host.startIndex..., in: host)
    return hostPattern.firstMatch(in: host, range: range) != nil
}

func formatBytes(_ bytes: Int64) -> String {
    if bytes < 1024 { return "\(bytes) B" }
    let units = ["KB", "MB", "GB", "TB"]
    var value = Double(bytes) / 1024
    var index = 0
    while value >= 1024 && index < units.count - 1 {
        value /= 1024
        index += 1
    }
    return String(format: "%.1f %@", value, units[index])
}

func isDomainName(_ value: String) -> Bool {
    let d = value.trimmingCharacters(in: CharacterSet(charactersIn: ".")).lowercased()
    if d.isEmpty || d.count > 253 { return false }
    let parts = d.split(separator: ".")
    if parts.count < 2 { return false }
    return parts.allSatisfy { part in
        if part.isEmpty || part.count > 63 { return false }
        return part.allSatisfy { $0.isLetter || $0.isNumber || $0 == "-" }
    }
}

private func encodeUserinfo(_ value: String) -> String {
    var allowed = CharacterSet.urlUserAllowed
    allowed.remove(charactersIn: "+:")
    return value.addingPercentEncoding(withAllowedCharacters: allowed) ?? value
}
