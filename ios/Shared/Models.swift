import Foundation

struct ServerItem: Identifiable, Hashable, Codable {
    var id: String
    var host: String
    var remark: String
    var username: String
    var password: String
    var protocolName: String
    var latencyMs: Int?
    var subscriptionId: String?

    var displayName: String { remark.isEmpty ? host : remark }

    init(
        id: String,
        host: String,
        remark: String = "",
        username: String = "",
        password: String = "",
        protocolName: String = "socks5",
        latencyMs: Int? = nil,
        subscriptionId: String? = nil,
    ) {
        self.id = id
        self.host = host
        self.remark = remark
        self.username = username
        self.password = password
        self.protocolName = protocolName
        self.latencyMs = latencyMs
        self.subscriptionId = subscriptionId
    }

    enum CodingKeys: String, CodingKey {
        case id, host, remark, username, password, protocolName = "protocol", latencyMs, subscriptionId
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        host = try c.decode(String.self, forKey: .host)
        remark = try c.decodeIfPresent(String.self, forKey: .remark) ?? ""
        username = try c.decodeIfPresent(String.self, forKey: .username) ?? ""
        password = try c.decodeIfPresent(String.self, forKey: .password) ?? ""
        protocolName = try c.decodeIfPresent(String.self, forKey: .protocolName) ?? "socks5"
        latencyMs = try c.decodeIfPresent(Int.self, forKey: .latencyMs)
        let sub = try c.decodeIfPresent(String.self, forKey: .subscriptionId)
        subscriptionId = (sub?.isEmpty == false) ? sub : nil
    }

    func encode(to encoder: Encoder) throws {
        var c = encoder.container(keyedBy: CodingKeys.self)
        try c.encode(id, forKey: .id)
        try c.encode(host, forKey: .host)
        try c.encode(remark, forKey: .remark)
        try c.encode(username, forKey: .username)
        try c.encode(password, forKey: .password)
        try c.encode(protocolName, forKey: .protocolName)
        try c.encodeIfPresent(latencyMs, forKey: .latencyMs)
        try c.encodeIfPresent(subscriptionId, forKey: .subscriptionId)
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

struct Subscription: Codable, Equatable {
    static let defaultId = "default"

    var id: String = defaultId
    var type: String
    var url: String
    var remark: String = ""
}

struct RuleDraft: Equatable {
    var geoMode: GeoMode = .global
    var selectedAreaIds: Set<Int64> = []
    var dbUrl: String = ""
    var domains: [String] = ["google.com"]
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
