import SwiftUI

struct HomeView: View {
    @EnvironmentObject private var store: Store
    @Binding var path: [AppRoute]
    @State private var selecting = false
    @State private var checkedIds: Set<String> = []
    @State private var pendingBatchDelete = false
    @State private var pendingDelete: ServerItem?
    @State private var pulse = false

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                statusCard
                    .padding(.horizontal, 16)
                    .padding(.vertical, 8)
                Text(store.t("server_list_title"))
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(.secondary)
                    .padding(.horizontal, 20)
                    .padding(.top, 8)
                    .padding(.bottom, 6)
                if store.servers.isEmpty {
                    VStack(alignment: .leading, spacing: 6) {
                        Text(store.t("server_list_empty"))
                            .font(.body.weight(.medium))
                        Text(store.t("server_list_empty_hint"))
                            .foregroundStyle(.secondary)
                    }
                    .padding(.horizontal, 20)
                    .padding(.vertical, 20)
                } else {
                    VStack(spacing: 0) {
                        ForEach(Array(store.servers.enumerated()), id: \.element.id) { index, server in
                            ServerRow(
                                server: server,
                                current: server.id == store.selectedId,
                                checking: selecting,
                                checked: checkedIds.contains(server.id),
                                isFirst: index == 0,
                                isLast: index == store.servers.count - 1,
                                showDivider: index < store.servers.count - 1,
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
                            )
                        }
                    }
                    .padding(.horizontal, 16)
                }
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
                            path.append(.subscribe)
                        } label: {
                            Image(systemName: "square.and.arrow.down")
                                .frame(width: 24, height: 24)
                        }
                        .accessibilityLabel(store.t("server_list_import"))
                        Button {
                            if !store.pingingAll { store.pingAll() }
                        } label: {
                            Image(systemName: "arrow.clockwise")
                                .opacity(store.pingingAll ? 0.35 : 1)
                                .frame(width: 24, height: 24)
                        }
                        .disabled(store.servers.isEmpty)
                        .accessibilityLabel(store.t("server_list_ping_all"))
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
                        Button(store.t("server_list_edit_title"), action: onEdit)
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
