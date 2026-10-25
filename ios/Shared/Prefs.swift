import Foundation

enum AppGroup {
    static let id = "group.com.penndev.prism"

    static var defaults: UserDefaults {
        UserDefaults(suiteName: id) ?? .standard
    }

    static var container: URL {
        if let url = FileManager.default.containerURL(forSecurityApplicationGroupIdentifier: id) {
            return url
        }
        return FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
    }

    /// 扩展和主 App 不共享沙盒。把以前写在 standard / Documents 里的配置和 IP 库拷进 App Group。
    static func migrateIfNeeded() {
        let suite = defaults
        let standard = UserDefaults.standard
        let keys = ["servers", "subscription", "selected_id", "settings", "rules"]
        if suite.object(forKey: "servers") == nil, standard.object(forKey: "servers") != nil {
            for key in keys {
                if suite.object(forKey: key) == nil, let value = standard.object(forKey: key) {
                    suite.set(value, forKey: key)
                }
            }
            suite.synchronize()
        }
        let docs = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
            .appendingPathComponent(ipregionDB)
        let dest = container.appendingPathComponent(ipregionDB)
        if !FileManager.default.fileExists(atPath: dest.path),
           FileManager.default.fileExists(atPath: docs.path) {
            try? FileManager.default.copyItem(at: docs, to: dest)
        }
    }
}

final class Prefs {
    private let defaults: UserDefaults

    init(defaults: UserDefaults = AppGroup.defaults) {
        AppGroup.migrateIfNeeded()
        self.defaults = defaults
    }

    func loadSettings() -> AppSettings {
        defaults.synchronize()
        guard let data = defaults.data(forKey: Key.settings) else { return AppSettings() }
        return (try? JSONDecoder().decode(AppSettings.self, from: data)) ?? AppSettings()
    }

    func saveSettings(_ settings: AppSettings) {
        if let data = try? JSONEncoder().encode(settings) {
            defaults.set(data, forKey: Key.settings)
            defaults.synchronize()
        }
    }

    func loadServers() -> [ServerItem] {
        guard let data = defaults.data(forKey: Key.servers) else { return [] }
        let servers = (try? JSONDecoder().decode([ServerItem].self, from: data)) ?? []
        return servers.map { server in
            var next = server
            if let decoded = next.remark.removingPercentEncoding {
                next.remark = decoded
            }
            return next
        }
    }

    func saveServers(_ servers: [ServerItem]) {
        if let data = try? JSONEncoder().encode(servers) {
            defaults.set(data, forKey: Key.servers)
        }
    }

    func loadSelectedId() -> String? {
        defaults.string(forKey: Key.selected)
    }

    func saveSelectedId(_ id: String?) {
        if let id {
            defaults.set(id, forKey: Key.selected)
        } else {
            defaults.removeObject(forKey: Key.selected)
        }
    }

    func loadSubscription() -> Subscription? {
        guard let data = defaults.data(forKey: Key.subscription),
              var sub = try? JSONDecoder().decode(Subscription.self, from: data)
        else { return nil }
        if sub.type.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            && sub.url.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return nil
        }
        if sub.id.isEmpty { sub.id = Subscription.defaultId }
        return sub
    }

    func saveSubscription(_ subscription: Subscription) {
        if let data = try? JSONEncoder().encode(subscription) {
            defaults.set(data, forKey: Key.subscription)
        }
    }

    func clearSubscription() {
        defaults.removeObject(forKey: Key.subscription)
    }

    func loadRules() -> RuleDraft {
        defaults.synchronize()
        guard let raw = defaults.string(forKey: Key.rules),
              let data = raw.data(using: .utf8),
              let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any]
        else { return RuleDraft() }
        let mode = GeoMode(rawValue: obj["geoMode"] as? String ?? "Global") ?? .global
        let areasSrc = obj["selectedAreaIds"] as? [Any] ?? obj["selectedAreas"] as? [Any] ?? []
        var areaIds = Set<Int64>()
        for item in areasSrc {
            let id: Int64
            if let n = item as? Int64 { id = n }
            else if let n = item as? Int { id = Int64(n) }
            else if let n = item as? NSNumber { id = n.int64Value }
            else { continue }
            if id > 0 { areaIds.insert(id) }
        }
        let domainsSrc = obj["domains"] as? [Any] ?? []
        var domains: [String] = []
        for item in domainsSrc {
            guard let s = item as? String else { continue }
            let d = s.trimmingCharacters(in: .whitespacesAndNewlines)
                .trimmingCharacters(in: CharacterSet(charactersIn: "."))
                .lowercased()
            if !d.isEmpty { domains.append(d) }
        }
        let seeded = obj["domainsSeeded"] as? Bool ?? false
        if !seeded && !domains.contains("google.com") {
            domains.insert("google.com", at: 0)
        }
        return RuleDraft(
            geoMode: mode,
            selectedAreaIds: areaIds,
            dbUrl: obj["dbUrl"] as? String ?? "",
            domains: domains,
        )
    }

    func saveRules(_ draft: RuleDraft) {
        let obj: [String: Any] = [
            "geoMode": draft.geoMode.rawValue,
            "selectedAreaIds": draft.selectedAreaIds.sorted().map { NSNumber(value: $0) },
            "dbUrl": draft.dbUrl,
            "domains": draft.domains,
            "domainsSeeded": true,
        ]
        if let data = try? JSONSerialization.data(withJSONObject: obj),
           let raw = String(data: data, encoding: .utf8) {
            defaults.set(raw, forKey: Key.rules)
            defaults.synchronize()
        }
    }

    private enum Key {
        static let servers = "servers"
        static let subscription = "subscription"
        static let selected = "selected_id"
        static let settings = "settings"
        static let rules = "rules"
    }
}
