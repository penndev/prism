import Foundation

enum AppGroup {
    // 接 Packet Tunnel 时再改回 App Group：group.com.penndev.prism
    static let id = "group.com.penndev.prism"

    static var defaults: UserDefaults { .standard }

    static var container: URL {
        FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
    }
}

final class Prefs {
    private let defaults: UserDefaults

    init(defaults: UserDefaults = AppGroup.defaults) {
        self.defaults = defaults
    }

    func loadSettings() -> AppSettings {
        guard let data = defaults.data(forKey: Key.settings) else { return AppSettings() }
        return (try? JSONDecoder().decode(AppSettings.self, from: data)) ?? AppSettings()
    }

    func saveSettings(_ settings: AppSettings) {
        if let data = try? JSONEncoder().encode(settings) {
            defaults.set(data, forKey: Key.settings)
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

    func loadRules() -> RuleDraft {
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
        ]
        if let data = try? JSONSerialization.data(withJSONObject: obj),
           let raw = String(data: data, encoding: .utf8) {
            defaults.set(raw, forKey: Key.rules)
        }
    }

    private enum Key {
        static let servers = "servers"
        static let selected = "selected_id"
        static let settings = "settings"
        static let rules = "rules"
    }
}
