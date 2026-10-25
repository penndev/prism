import SwiftUI

struct HomeView: View {
    @EnvironmentObject private var store: Store
    @Binding var path: [AppRoute]
    @State private var selecting = false
    @State private var checkedIds: Set<String> = []
    @State private var pendingBatchDelete = false
    @State private var pendingDelete: ServerItem?
    @State private var pulse = false
    @State private var subExpanded = true

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                statusCard
                    .padding(.horizontal, 16)
                    .padding(.vertical, 8)
                serverSections
            }
            .padding(.bottom, 24)
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle(selecting ? store.t("server_list_selected_count", checkedIds.count) : store.t("app_title"))
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            if selecting {
                ToolbarItem(placement: .cancellationAction) {
                    Button {
                        selecting = false
                        checkedIds = []
                    } label: {
                        Image(systemName: "xmark")
                    }
                }
                ToolbarItem(placement: .primaryAction) {
                    HStack {
                        Button(store.t("server_list_select_all")) {
                            if checkedIds.count == store.servers.count {
                                checkedIds = []
                            } else {
                                checkedIds = Set(store.servers.map(\.id))
                            }
                        }
                        Button {
                            pendingBatchDelete = true
                        } label: {
                            Image(systemName: "trash")
                                .foregroundStyle(checkedIds.isEmpty ? Color.secondary : Color.latencyBad)
                        }
                        .disabled(checkedIds.isEmpty)
                    }
                }
            } else {
                ToolbarItem(placement: .topBarTrailing) {
                    HStack(spacing: 18) {
                        Button {
                            path.append(.serverEdit(id: nil))
                        } label: {
                            Image(systemName: "plus")
                                .frame(width: 24, height: 24)
                        }
                        .accessibilityLabel(store.t("server_list_add"))
                        Button {
                            path.append(.subscribe(edit: false))
                        } label: {
                            Image(systemName: "dot.radiowaves.up.forward")
                                .frame(width: 24, height: 24)
                        }
                        .accessibilityLabel(store.t("server_list_import"))
                    }
                    .font(.body.weight(.medium))
                    .foregroundStyle(.primary)
                    .buttonStyle(.plain)
                }
                .hideSharedBackgroundIfAvailable()
            }
        }
        .alert(store.t("server_list_delete_title"), isPresented: Binding(
            get: { pendingDelete != nil },
            set: { if !$0 { pendingDelete = nil } },
        )) {
            Button(store.t("server_list_delete_cancel"), role: .cancel) { pendingDelete = nil }
            Button(store.t("server_list_delete_ok"), role: .destructive) {
                if let id = pendingDelete?.id { store.deleteServer(id: id) }
                pendingDelete = nil
            }
        } message: {
            if let server = pendingDelete {
                Text(store.t("server_list_delete_content", server.displayName))
            }
        }
        .alert(store.t("server_list_delete_title"), isPresented: $pendingBatchDelete) {
            Button(store.t("server_list_delete_cancel"), role: .cancel) {}
            Button(store.t("server_list_delete_ok"), role: .destructive) {
                store.deleteServers(ids: checkedIds)
                pendingBatchDelete = false
                selecting = false
                checkedIds = []
            }
        } message: {
            Text(store.t("server_list_batch_delete_content", checkedIds.count))
        }
        .onAppear { pulse = true }
    }

    @ViewBuilder
    private var serverSections: some View {
        let manual = store.servers.filter { $0.subscriptionId == nil }
        let sub = store.subscription
        let subNodes: [ServerItem] = sub == nil ? [] : store.servers.filter { $0.subscriptionId == sub?.id }
        if !manual.isEmpty || sub == nil {
            HStack {
                Text(store.t("server_list_title"))
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(.secondary)
                Spacer()
                if !manual.isEmpty && !selecting {
                    pingButton(running: store.pingingGroup == pingManual) {
                        store.pingGroup(nil)
                    }
                }
            }
            .padding(.leading, 20)
            .padding(.trailing, 12)
            .padding(.top, 8)
            .padding(.bottom, 6)
        }
        if manual.isEmpty && sub == nil {
            VStack(alignment: .leading, spacing: 6) {
                Text(store.t("server_list_empty"))
                    .font(.body.weight(.medium))
                Text(store.t("server_list_empty_hint"))
                    .foregroundStyle(.secondary)
            }
            .padding(.horizontal, 20)
            .padding(.vertical, 20)
        }
        if !manual.isEmpty {
            VStack(spacing: 0) {
                ForEach(Array(manual.enumerated()), id: \.element.id) { index, server in
                    nodeRow(server, index: index, count: manual.count, groupedTop: true)
                }
            }
            .padding(.horizontal, 16)
        }
        if let sub {
            let joined = subExpanded && !subNodes.isEmpty
            HStack(spacing: 0) {
                Button {
                    subExpanded.toggle()
                } label: {
                    HStack(spacing: 8) {
                        Image(systemName: subExpanded ? "chevron.down" : "chevron.right")
                            .font(.system(size: 11, weight: .semibold))
                            .frame(width: 12, height: 12)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(sub.remark.isEmpty ? (sub.type.isEmpty ? store.t("subscribe_title") : sub.type) : sub.remark)
                                .font(.body.weight(.medium))
                                .foregroundStyle(.primary)
                                .lineLimit(1)
                            if !sub.url.isEmpty {
                                Text(sub.url)
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                    .lineLimit(1)
                            }
                        }
                        Spacer(minLength: 8)
                        Text("\(subNodes.count)")
                            .foregroundStyle(.secondary)
                    }
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                if !selecting {
                    headerIcon("arrow.clockwise", label: store.t("subscribe_refresh"), enabled: !store.importing) {
                        store.refreshSubscription()
                    }
                    headerIcon("info.circle", label: store.t("subscribe_edit")) {
                        path.append(.subscribe(edit: true))
                    }
                    if !subNodes.isEmpty {
                        pingButton(running: store.pingingGroup == sub.id, compact: true) {
                            store.pingGroup(sub.id)
                        }
                    }
                }
            }
            .padding(.leading, 14)
            .padding(.trailing, 4)
            .padding(.vertical, 10)
            .background(Color(.secondarySystemGroupedBackground))
            .clipShape(UnevenRoundedRectangle(
                topLeadingRadius: 12,
                bottomLeadingRadius: joined ? 0 : 12,
                bottomTrailingRadius: joined ? 0 : 12,
                topTrailingRadius: 12,
            ))
            .padding(.horizontal, 16)
            .padding(.top, 12)
            if joined {
                VStack(spacing: 0) {
                    ForEach(Array(subNodes.enumerated()), id: \.element.id) { index, server in
                        nodeRow(server, index: index, count: subNodes.count, groupedTop: false)
                    }
                }
                .padding(.horizontal, 16)
            }
        }
    }

    private func nodeRow(_ server: ServerItem, index: Int, count: Int, groupedTop: Bool) -> some View {
        ServerRow(
            server: server,
            current: server.id == store.selectedId,
            checking: selecting,
            checked: checkedIds.contains(server.id),
            isFirst: groupedTop && index == 0,
            isLast: index == count - 1,
            showDivider: index < count - 1,
            onClick: {
                if selecting {
                    if checkedIds.contains(server.id) {
                        checkedIds.remove(server.id)
                    } else {
                        checkedIds.insert(server.id)
                    }
                } else {
                    store.selectServer(server)
                }
            },
            onLongClick: {
                selecting = true
                checkedIds.insert(server.id)
            },
            onEdit: { path.append(.serverEdit(id: server.id)) },
            onDelete: { pendingDelete = server },
            onPing: { store.pingServer(id: server.id) },
        )
    }

    private func pingButton(running: Bool, compact: Bool = false, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Group {
                if running {
                    ProgressView()
                        .controlSize(.small)
                } else {
                    Image(systemName: "speedometer")
                }
            }
            .frame(width: compact ? 26 : 32, height: compact ? 26 : 32)
        }
        .buttonStyle(.plain)
        .disabled(store.pingingGroup != nil)
        .accessibilityLabel(store.t("server_list_ping"))
    }

    private func headerIcon(_ name: String, label: String, enabled: Bool = true, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Image(systemName: name)
                .frame(width: 26, height: 26)
                .opacity(enabled ? 1 : 0.35)
        }
        .buttonStyle(.plain)
        .disabled(!enabled)
        .accessibilityLabel(label)
    }

    private var statusCard: some View {
        let switchOn = store.vpnBusy ? store.vpnDesired : store.running
        let statusTitle: String = {
            if store.vpnBusy && store.vpnDesired { return store.t("proxy_connecting") }
            if store.vpnBusy && !store.vpnDesired { return store.t("proxy_stopping") }
            if store.running { return store.t("proxy_connected") }
            return store.t("proxy_disconnected")
        }()
        let dotColor: Color = {
            if store.running && !store.vpnBusy { return .latencyGood }
            if store.vpnBusy { return .prismBlue }
            return Color(.tertiaryLabel)
        }()
        return HStack(spacing: 12) {
            Circle()
                .fill(dotColor)
                .frame(width: 10, height: 10)
                .opacity(store.vpnBusy ? (pulse ? 1 : 0.4) : 1)
                .animation(store.vpnBusy ? .easeInOut(duration: 0.7).repeatForever(autoreverses: true) : .default, value: pulse)
            VStack(alignment: .leading, spacing: 2) {
                Text(statusTitle)
                    .font(.subheadline.weight(.semibold))
                Text(store.selectedServer?.displayName ?? store.t("proxy_no_selected"))
                    .font(.subheadline)
                    .foregroundStyle(store.selectedServer == nil ? Color.secondary : Color.prismBlue)
                    .lineLimit(1)
            }
            Spacer()
            if store.vpnBusy {
                ProgressView()
                    .padding(.trailing, 4)
            }
            Toggle("", isOn: Binding(
                get: { switchOn },
                set: { wantOn in
                    if !wantOn {
                        store.stop()
                    } else if store.selectedServer == nil {
                        store.snack("proxy_need_node")
                    } else {
                        store.start()
                    }
                },
            ))
            .labelsHidden()
            .tint(.prismBlue)
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 10)
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 12))
    }
}

private struct ServerRow: View {
    let server: ServerItem
    let current: Bool
    let checking: Bool
    let checked: Bool
    let isFirst: Bool
    let isLast: Bool
    let showDivider: Bool
    let onClick: () -> Void
    let onLongClick: () -> Void
    let onEdit: () -> Void
    let onDelete: () -> Void
    let onPing: () -> Void

    @EnvironmentObject private var store: Store

    var body: some View {
        VStack(spacing: 0) {
            HStack(spacing: 0) {
                if checking {
                    Image(systemName: checked ? "checkmark.circle.fill" : "circle")
                        .foregroundStyle(checked ? Color.prismBlue : Color.secondary)
                        .padding(.leading, 12)
                } else {
                    Rectangle()
                        .fill(current ? Color.prismBlue : Color.clear)
                        .frame(width: 3, height: 28)
                        .padding(.vertical, 12)
                }
                Button(action: onClick) {
                    VStack(alignment: .leading, spacing: 2) {
                        HStack {
                            Text(server.displayName)
                                .fontWeight(current && !checking ? .semibold : .medium)
                                .lineLimit(1)
                            Spacer()
                            LatencyLabel(latencyMs: server.latencyMs)
                        }
                        Text(server.protocolName)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    .padding(.leading, checking ? 8 : 13)
                    .padding(.vertical, 12)
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .simultaneousGesture(LongPressGesture().onEnded { _ in onLongClick() })
                if !checking {
                    Menu {
                        Button(store.t("server_list_edit"), action: onEdit)
                        Button(store.t("server_list_ping"), action: onPing)
                        Button(store.t("server_list_delete_ok"), role: .destructive, action: onDelete)
                    } label: {
                        Image(systemName: "ellipsis")
                            .frame(width: 36, height: 36)
                    }
                    .padding(.trailing, 4)
                }
            }
            if showDivider { PreferenceDivider() }
        }
        .background(current && !checking ? Color.prismBlue.opacity(0.12) : Color(.secondarySystemGroupedBackground))
        .clipShape(UnevenRoundedRectangle(
            topLeadingRadius: isFirst ? 12 : 0,
            bottomLeadingRadius: isLast ? 12 : 0,
            bottomTrailingRadius: isLast ? 12 : 0,
            topTrailingRadius: isFirst ? 12 : 0,
        ))
    }
}

private struct LatencyLabel: View {
    let latencyMs: Int?
    @EnvironmentObject private var store: Store

    var body: some View {
        if let latencyMs {
            let text: String = latencyMs < 0 ? store.t("server_list_ping_failed") : store.t("server_list_ms", latencyMs)
            let color: Color = {
                if latencyMs < 0 { return .latencyBad }
                if latencyMs < 100 { return .latencyGood }
                if latencyMs < 300 { return .latencyMedium }
                return .latencyBad
            }()
            Text(text)
                .font(.caption.weight(.medium))
                .foregroundStyle(color)
                .padding(.leading, 8)
        }
    }
}

private extension ToolbarContent {
    @ToolbarContentBuilder
    func hideSharedBackgroundIfAvailable() -> some ToolbarContent {
        if #available(iOS 26.0, *) {
            sharedBackgroundVisibility(.hidden)
        } else {
            self
        }
    }
}
