import SwiftUI
import UniformTypeIdentifiers

struct RulesView: View {
    @EnvironmentObject private var store: Store
    @Binding var path: [AppRoute]

    var body: some View {
        ScrollView {
            PreferenceGroup(title: store.t("rules_section")) {
                PreferenceRow(
                    title: store.t("rules_entry_domain"),
                    value: domainValue,
                    description: store.t("rules_entry_domain_desc"),
                ) { path.append(.domainRules) }
                PreferenceDivider()
                PreferenceRow(
                    title: store.t("rules_entry_geo"),
                    value: geoValue,
                    description: store.t("rules_entry_geo_desc"),
                ) { path.append(.geoRules) }
            }
            .padding(.top, 8)
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle(store.t("nav_rules"))
        .navigationBarTitleDisplayMode(.inline)
    }

    private var domainValue: String {
        let count = store.rules.domains.filter { isDomainName($0) }.count
        return count == 0 ? store.t("rules_domain_empty") : store.t("rules_domain_count", count)
    }

    private var geoValue: String {
        switch store.rules.geoMode {
        case .global: return store.t("rules_mode_global")
        case .none: return store.t("rules_mode_none")
        case .proxy: return store.t("rules_mode_proxy")
        case .bypass: return store.t("rules_mode_bypass")
        }
    }
}

struct DomainRulesView: View {
    @EnvironmentObject private var store: Store
    @Environment(\.dismiss) private var dismiss
    @State private var text = ""
    @State private var sourceUrl = ""
    @State private var fetching = false
    @State private var importingFile = false

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 12) {
                Text(store.t("rules_domain_hint"))
                    .font(.caption)
                    .foregroundStyle(.secondary)
                TextField(store.t("rules_domain_url_placeholder"), text: $sourceUrl)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .padding(12)
                    .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 10))
                HStack(spacing: 10) {
                    Button {
                        fetchUrl()
                    } label: {
                        if fetching {
                            ProgressView()
                                .frame(maxWidth: .infinity)
                        } else {
                            Text(store.t("rules_domain_fetch"))
                                .frame(maxWidth: .infinity)
                        }
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(fetching)
                    Button {
                        importingFile = true
                    } label: {
                        Text(store.t("rules_domain_import_file"))
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.bordered)
                    .disabled(fetching)
                }
                TextEditor(text: $text)
                    .font(.body)
                    .frame(minHeight: 240)
                    .padding(8)
                    .overlay(
                        RoundedRectangle(cornerRadius: 8)
                            .stroke(Color(.separator), lineWidth: 1)
                    )
            }
            .padding(16)
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle(store.t("rules_domain_title"))
        .navigationBarTitleDisplayMode(.inline)
        .navigationBarBackButtonHidden(true)
        .toolbar {
            ToolbarItem(placement: .cancellationAction) {
                Button {
                    saveAndBack()
                } label: {
                    Image(systemName: "chevron.left")
                }
            }
        }
        .fileImporter(
            isPresented: $importingFile,
            allowedContentTypes: [.plainText, .text, .data],
        ) { result in
            switch result {
            case .success(let url):
                let accessed = url.startAccessingSecurityScopedResource()
                defer { if accessed { url.stopAccessingSecurityScopedResource() } }
                let data = try? Data(contentsOf: url)
                let raw = data.flatMap { SubscriptionParser.decodeText($0) }?
                    .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
                if raw.isEmpty {
                    store.snack("subscribe_error_file")
                } else {
                    mergeContent(raw)
                }
            case .failure:
                store.snack("subscribe_error_file")
            }
        }
        .onAppear {
            text = store.rules.domains.joined(separator: "\n")
        }
    }

    private func saveAndBack() {
        store.updateRules {
            var next = $0
            next.domains = parseDomainList(text)
            return next
        }
        dismiss()
    }

    private func mergeContent(_ raw: String) {
        let body = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        if body.isEmpty { return }
        if text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            text = body
        } else {
            text += "\n" + body
        }
    }

    private func fetchUrl() {
        let url = sourceUrl.trimmingCharacters(in: .whitespacesAndNewlines)
        if url.isEmpty {
            store.snack("subscribe_error_url")
            return
        }
        fetching = true
        Task.detached {
            do {
                let body = try fetchPlainText(url)
                await MainActor.run {
                    fetching = false
                    mergeContent(body)
                }
            } catch {
                await MainActor.run {
                    fetching = false
                    store.snack("rules_domain_fetch_error")
                }
            }
        }
    }
}

private func parseDomainList(_ raw: String) -> [String] {
    var seen = [String]()
    var set = Set<String>()
    for item in raw.split(whereSeparator: { $0 == "\n" || $0 == "\r" }) {
        let d = item.trimmingCharacters(in: .whitespacesAndNewlines)
            .trimmingCharacters(in: CharacterSet(charactersIn: "."))
            .lowercased()
        if !d.isEmpty && set.insert(d).inserted {
            seen.append(d)
        }
    }
    return seen
}

private func fetchPlainText(_ url: String) throws -> String {
    guard let uri = URL(string: url), let scheme = uri.scheme?.lowercased(),
          scheme == "http" || scheme == "https"
    else { throw EngineError("bad url") }
    var request = URLRequest(url: uri, timeoutInterval: 15)
    request.setValue("Prism-iOS/0.1", forHTTPHeaderField: "User-Agent")
    let sem = DispatchSemaphore(value: 0)
    var result: Result<String, Error> = .failure(EngineError("fetch failed"))
    let task = URLSession.shared.dataTask(with: request) { data, response, error in
        defer { sem.signal() }
        if let error {
            result = .failure(error)
            return
        }
        let code = (response as? HTTPURLResponse)?.statusCode ?? 0
        let text = data.flatMap { SubscriptionParser.decodeText($0) }?
            .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !(200...299).contains(code) || text.isEmpty {
            result = .failure(EngineError("fetch failed"))
            return
        }
        result = .success(text)
    }
    task.resume()
    if sem.wait(timeout: .now() + 20) == .timedOut {
        task.cancel()
        throw EngineError("timeout")
    }
    return try result.get()
}

struct GeoRulesView: View {
    @EnvironmentObject private var store: Store
    @State private var areaFilter = ""
    @State private var expandedIds: Set<Int64> = []
    @State private var editingDb = false
    @State private var dbUrlDraft = ""
    @State private var importing = false

    var body: some View {
        let rules = store.rules
        let showAreas = rules.geoMode == .proxy || rules.geoMode == .bypass
        let hasDb = !store.geoAreas.isEmpty
        let showDbEditor = !store.dbReady || editingDb
        let rows = flattenAreas(store.geoAreas, expanded: expandedIds, filter: areaFilter, selectedIds: rules.selectedAreaIds)

        ScrollView {
            VStack(spacing: 12) {
                PreferenceGroup(
                    title: store.t("rules_db_label"),
                    actions: store.dbReady ? AnyView(
                        Button(store.t(editingDb ? "rules_db_cancel" : "rules_db_replace")) {
                            editingDb.toggle()
                        }
                        .disabled(store.dbBusy)
                    ) : nil,
                ) {
                    Text(store.dbStatus.isEmpty ? store.t("rules_db_missing") : store.dbStatus)
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                        .padding(.horizontal, 16)
                        .padding(.vertical, 12)
                        .frame(maxWidth: .infinity, alignment: .leading)
                    if store.dbBusy {
                        ProgressView()
                            .progressViewStyle(.linear)
                            .padding(.horizontal, 16)
                            .padding(.bottom, 12)
                    }
                    if showDbEditor {
                        PreferenceDivider()
                        VStack(alignment: .leading, spacing: 6) {
                            Text(store.t("rules_db_url_label"))
                                .font(.caption)
                                .foregroundStyle(.secondary)
                            TextField(store.t("rules_db_url"), text: $dbUrlDraft)
                                .textInputAutocapitalization(.never)
                                .autocorrectionDisabled()
                                .disabled(store.dbBusy)
                        }
                        .padding(.horizontal, 16)
                        .padding(.vertical, 12)
                        PreferenceDivider()
                        VStack(spacing: 10) {
                            Button {
                                store.updateRules {
                                    var next = $0
                                    next.dbUrl = dbUrlDraft
                                    return next
                                }
                                store.downloadDb()
                            } label: {
                                Text(store.t("rules_download"))
                                    .frame(maxWidth: .infinity)
                            }
                            .buttonStyle(.borderedProminent)
                            .disabled(store.dbBusy)
                            Button {
                                importing = true
                            } label: {
                                Text(store.t("rules_upload"))
                                    .frame(maxWidth: .infinity)
                            }
                            .buttonStyle(.bordered)
                            .disabled(store.dbBusy)
                        }
                        .padding(16)
                    }
                }

                PreferenceGroup(title: store.t("rules_geo_title")) {
                    Text(store.t("rules_geo_desc"))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .padding(.horizontal, 16)
                        .padding(.vertical, 8)
                    modeRow(.global, enabled: true)
                    PreferenceDivider()
                    modeRow(.none, enabled: true)
                    PreferenceDivider()
                    modeRow(.proxy, enabled: hasDb)
                    PreferenceDivider()
                    modeRow(.bypass, enabled: hasDb)
                }

                if showAreas {
                    VStack(alignment: .leading, spacing: 8) {
                        TextField(store.t("rules_area_filter"), text: $areaFilter)
                            .textFieldStyle(.roundedBorder)
                        Text(store.t("rules_selected_count", rules.selectedAreaIds.count))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    .padding(.horizontal, 16)

                    if !hasDb {
                        Text(store.t("rules_area_empty"))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .padding(.horizontal, 16)
                    } else {
                        VStack(spacing: 0) {
                            ForEach(rows, id: \.area.id) { row in
                                AreaTreeRow(row: row, onToggleExpand: {
                                    if expandedIds.contains(row.area.id) {
                                        expandedIds.remove(row.area.id)
                                    } else {
                                        expandedIds.insert(row.area.id)
                                    }
                                }, onToggleSelect: {
                                    store.updateRules { draft in
                                        var next = draft
                                        next.selectedAreaIds = toggleAreaSelection(
                                            tree: store.geoAreas,
                                            selected: draft.selectedAreaIds,
                                            node: row.area,
                                            currentlyChecked: row.selected,
                                        )
                                        return next
                                    }
                                })
                            }
                        }
                    }
                }
            }
            .padding(.vertical, 8)
            .padding(.bottom, 24)
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle(store.t("rules_geo_title"))
        .navigationBarTitleDisplayMode(.inline)
        .fileImporter(isPresented: $importing, allowedContentTypes: [.data, .item]) { result in
            if case .success(let url) = result {
                store.importDb(url: url)
            }
        }
        .onAppear {
            dbUrlDraft = store.rules.dbUrl
            expandedIds = topLevelIdsWithSelected(store.geoAreas, selected: store.rules.selectedAreaIds)
        }
        .onChange(of: store.dbBusy) { busy in
            if !busy && store.dbReady { editingDb = false }
        }
    }

    @ViewBuilder
    private func modeRow(_ mode: GeoMode, enabled: Bool) -> some View {
        let title: String = {
            switch mode {
            case .global: return store.t("rules_mode_global")
            case .none: return store.t("rules_mode_none")
            case .proxy: return store.t("rules_mode_proxy")
            case .bypass: return store.t("rules_mode_bypass")
            }
        }()
        let desc: String = {
            switch mode {
            case .global: return store.t("rules_mode_global_desc")
            case .none: return store.t("rules_mode_none_desc")
            case .proxy: return store.t("rules_mode_proxy_desc")
            case .bypass: return store.t("rules_mode_bypass_desc")
            }
        }()
        Button {
            store.updateRules {
                var next = $0
                next.geoMode = mode
                return next
            }
        } label: {
            HStack {
                Image(systemName: store.rules.geoMode == mode ? "largecircle.fill.circle" : "circle")
                    .foregroundStyle(enabled ? Color.prismBlue : Color.secondary.opacity(0.38))
                VStack(alignment: .leading, spacing: 2) {
                    Text(title)
                        .foregroundStyle(enabled ? Color.primary : Color.primary.opacity(0.38))
                    Text(desc)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer()
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 8)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .disabled(!enabled)
    }
}

private struct AreaRow {
    let area: AreaUi
    let depth: Int
    let expandable: Bool
    let expanded: Bool
    let selected: Bool
}

private struct AreaTreeRow: View {
    let row: AreaRow
    let onToggleExpand: () -> Void
    let onToggleSelect: () -> Void

    var body: some View {
        HStack(alignment: .center, spacing: 0) {
            Color.clear.frame(width: CGFloat(12 + row.depth * 16), height: 1)
            ZStack {
                if row.expandable {
                    Image(systemName: row.expanded ? "chevron.down" : "chevron.right")
                        .font(.footnote.weight(.semibold))
                        .foregroundStyle(.secondary)
                }
            }
            .frame(width: 28, height: 36)
            .contentShape(Rectangle())
            .onTapGesture {
                if row.expandable { onToggleExpand() }
            }
            Button(action: onToggleSelect) {
                HStack {
                    Image(systemName: row.selected ? "checkmark.square.fill" : "square")
                        .foregroundStyle(row.selected ? Color.prismBlue : Color.secondary)
                    Text(row.area.name)
                    Spacer()
                }
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
        }
        .padding(.trailing, 8)
    }
}

private func selfAndDescendantIds(_ node: AreaUi) -> Set<Int64> {
    var out = Set<Int64>()
    func walk(_ n: AreaUi) {
        out.insert(n.id)
        n.children.forEach(walk)
    }
    walk(node)
    return out
}

private func toggleAreaSelection(
    tree: [AreaUi],
    selected: Set<Int64>,
    node: AreaUi,
    currentlyChecked: Bool,
) -> Set<Int64> {
    var next = selected
    if !currentlyChecked {
        next.insert(node.id)
        return next
    }
    if next.contains(node.id) {
        next.subtract(selfAndDescendantIds(node))
        return next
    }
    guard let path = pathFromRoot(tree, id: node.id) else {
        next.subtract(selfAndDescendantIds(node))
        return next
    }
    for i in stride(from: path.count - 1, through: 0, by: -1) {
        let cur = path[i]
        guard next.contains(cur.id) else { continue }
        next.remove(cur.id)
        let skip = i + 1 < path.count ? path[i + 1].id : node.id
        for child in cur.children where child.id != skip {
            next.insert(child.id)
        }
    }
    next.subtract(selfAndDescendantIds(node))
    return next
}

private func pathFromRoot(_ nodes: [AreaUi], id: Int64) -> [AreaUi]? {
    func walk(_ n: AreaUi, acc: [AreaUi]) -> [AreaUi]? {
        let next = acc + [n]
        if n.id == id { return next }
        for c in n.children {
            if let found = walk(c, acc: next) { return found }
        }
        return nil
    }
    for n in nodes {
        if let found = walk(n, acc: []) { return found }
    }
    return nil
}

private func topLevelIdsWithSelected(_ nodes: [AreaUi], selected: Set<Int64>) -> Set<Int64> {
    if selected.isEmpty { return [] }
    func containsSelected(_ node: AreaUi) -> Bool {
        selected.contains(node.id) || node.children.contains(where: containsSelected)
    }
    return Set(nodes.compactMap { containsSelected($0) ? $0.id : nil })
}

private func flattenAreas(
    _ nodes: [AreaUi],
    expanded: Set<Int64>,
    filter: String,
    selectedIds: Set<Int64>,
) -> [AreaRow] {
    let query = filter.trimmingCharacters(in: .whitespacesAndNewlines)
    var out: [AreaRow] = []
    var matched: [Int64: Bool] = [:]
    func computeMatches(_ node: AreaUi) -> Bool {
        let selfMatch = query.isEmpty
            || node.name.localizedCaseInsensitiveContains(query)
            || String(node.id).contains(query)
        var any = false
        for child in node.children {
            if computeMatches(child) { any = true }
        }
        let result = selfMatch || any
        matched[node.id] = result
        return result
    }
    nodes.forEach { _ = computeMatches($0) }

    func walk(_ node: AreaUi, depth: Int, ancestorSelected: Bool) {
        guard matched[node.id] == true else { return }
        let expandable = !node.children.isEmpty
        let open = expandable && (!query.isEmpty || expanded.contains(node.id))
        let selected = ancestorSelected || selectedIds.contains(node.id)
        out.append(AreaRow(area: node, depth: depth, expandable: expandable, expanded: open, selected: selected))
        if open {
            node.children.forEach { walk($0, depth: depth + 1, ancestorSelected: selected) }
        }
    }
    nodes.forEach { walk($0, depth: 0, ancestorSelected: false) }
    return out
}
