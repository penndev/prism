import Foundation

struct SubscriptionError: LocalizedError {
    let key: String
    let intArg: Int?

    init(_ key: String, _ intArg: Int? = nil) {
        self.key = key
        self.intArg = intArg
    }

    var errorDescription: String? { key }
}

enum SubscriptionParser {
    static func parseContent(type: String, raw: String) throws -> [ServerItem] {
        let text = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        if text.isEmpty {
            throw SubscriptionError("subscribe_error_empty")
        }
        if text.hasPrefix("[") {
            return try serversFromJson(text)
        }
        switch type.lowercased() {
        case "prism":
            return try parsePrism(text)
        case "shadowrocket":
            return try parseShadowrocket(text)
        default:
            throw SubscriptionError("subscribe_error_type")
        }
    }

    static func exportJson(_ servers: [ServerItem]) -> String {
        let arr: [[String: String]] = servers.map { server in
            [
                "host": server.host,
                "remark": server.remark,
                "username": server.username,
                "password": server.password,
                "protocol": server.protocolName,
            ]
        }
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys, .withoutEscapingSlashes]
        guard let data = try? encoder.encode(arr),
              let text = String(data: data, encoding: .utf8)
        else { return "[]" }
        return text
    }

    static func normalizeProtocol(_ raw: String) -> String {
        let key = raw.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        return proxySchemes.contains(key) ? key : "socks5"
    }

    static func identity(host: String, protocolName: String, username: String, password: String) -> String {
        let scheme = normalizeProtocol(protocolName).lowercased()
        return "\(scheme)://\(username):\(password)@\(host)"
    }

    private static func parsePrism(_ text: String) throws -> [ServerItem] {
        let json = text.hasPrefix("[") ? text : decodeMaybeBase64(text)
        return try serversFromJson(json)
    }

    private static func parseShadowrocket(_ text: String) throws -> [ServerItem] {
        let payload = decodeMaybeBase64(text)
        var seen = Set<String>()
        let servers = payload
            .replacingOccurrences(of: "\r\n", with: "\n")
            .split(separator: "\n", omittingEmptySubsequences: false)
            .compactMap { parseShadowrocketLine(String($0)) }
            .filter { seen.insert($0.id).inserted }
        if servers.isEmpty {
            throw SubscriptionError("subscribe_error_no_nodes")
        }
        return servers
    }

    private static func parseShadowrocketLine(_ raw: String) -> ServerItem? {
        let line = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let uri = parseNodeUri(line), let scheme = uri.scheme?.lowercased() else { return nil }
        let protocolName: String
        switch scheme {
        case "https": protocolName = "https"
        case "socks5s": protocolName = "socks5s"
        case "socks5", "socks": protocolName = "socks5"
        default: return nil
        }
        guard let hostname = uri.host, let port = uri.port, port > 0 else { return nil }
        let host = hostname.contains(":") ? "[\(hostname)]:\(port)" : "\(hostname):\(port)"
        if !hostMatches(host) { return nil }
        let username = uri.user ?? ""
        let password = uri.password ?? ""
        // URL.fragment 会留下 %E9%A6%99…；URLComponents.fragment 才是解码后的中文
        let remark = (uri.fragment ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        return ServerItem(
            id: identity(host: host, protocolName: protocolName, username: username, password: password),
            host: host,
            remark: remark.isEmpty ? host : remark,
            username: username,
            password: password,
            protocolName: protocolName,
        )
    }

    private static func parseNodeUri(_ line: String) -> URLComponents? {
        if line.isEmpty || !line.contains("://") { return nil }
        let scheme = String(line.prefix { $0 != ":" && $0 != "/" })
        guard let range = line.range(of: "://") else { return URLComponents(string: line) }
        let payload = String(line[range.upperBound...])
        if scheme.isEmpty || payload.isEmpty { return URLComponents(string: line) }
        let rebuilt: String
        if let decoded = decodeBase64Bytes(payload), let text = decodeText(decoded) {
            rebuilt = "\(scheme)://\(text.trimmingCharacters(in: .whitespacesAndNewlines))"
        } else {
            rebuilt = line
        }
        if let components = URLComponents(string: rebuilt) { return components }
        guard let url = URL(string: rebuilt) else { return nil }
        var components = URLComponents()
        components.scheme = url.scheme
        components.host = url.host
        components.port = url.port
        components.user = url.user.flatMap { $0.removingPercentEncoding } ?? url.user
        components.password = url.password.flatMap { $0.removingPercentEncoding } ?? url.password
        components.fragment = url.fragment.flatMap { $0.removingPercentEncoding } ?? url.fragment
        return components
    }

    private static func serversFromJson(_ json: String) throws -> [ServerItem] {
        guard let data = json.data(using: .utf8),
              let arr = try? JSONSerialization.jsonObject(with: data) as? [[String: Any]]
        else { throw SubscriptionError("subscribe_error_json") }
        var seen = Set<String>()
        var servers: [ServerItem] = []
        for obj in arr {
            let host = (obj["host"] as? String ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
            if host.isEmpty { continue }
            let protocolName = normalizeProtocol(obj["protocol"] as? String ?? "socks5")
            let username = obj["username"] as? String ?? ""
            let password = obj["password"] as? String ?? ""
            let id = identity(host: host, protocolName: protocolName, username: username, password: password)
            if !seen.insert(id).inserted { continue }
            let remark = obj["remark"] as? String ?? ""
            servers.append(
                ServerItem(
                    id: id,
                    host: host,
                    remark: remark.removingPercentEncoding ?? remark,
                    username: username,
                    password: password,
                    protocolName: protocolName,
                )
            )
        }
        if servers.isEmpty {
            throw SubscriptionError("subscribe_error_no_nodes")
        }
        return servers
    }

    private static func decodeMaybeBase64(_ text: String) -> String {
        if let data = decodeBase64Bytes(text), let s = decodeText(data) {
            return s
        }
        return text
    }

    static func decodeText(_ data: Data) -> String? {
        if let s = String(data: data, encoding: .utf8) { return s }
        let gb = CFStringConvertEncodingToNSStringEncoding(
            CFStringEncoding(CFStringEncodings.GB_18030_2000.rawValue)
        )
        return String(data: data, encoding: String.Encoding(rawValue: gb))
    }

    private static func decodeBase64Bytes(_ text: String) -> Data? {
        let compact = text.replacingOccurrences(of: "\\s", with: "", options: .regularExpression)
        let pad = String(repeating: "=", count: (4 - compact.count % 4) % 4)
        let padded = compact + pad
        if let data = Data(base64Encoded: padded) { return data }
        let urlSafe = padded.replacingOccurrences(of: "-", with: "+").replacingOccurrences(of: "_", with: "/")
        return Data(base64Encoded: urlSafe)
    }
}
