import SwiftUI

struct LogsView: View {
    @EnvironmentObject private var store: Store
    @State private var tab = 0

    var body: some View {
        VStack(spacing: 0) {
            PreferenceGroup(title: store.t("log_traffic_title")) {
                PreferenceRow(
                    title: store.t("log_traffic_down"),
                    value: "\(store.traffic.downSpeed) · \(store.traffic.downTotal)",
                    showChevron: false,
                )
                PreferenceDivider()
                PreferenceRow(
                    title: store.t("log_traffic_up"),
                    value: "\(store.traffic.upSpeed) · \(store.traffic.upTotal)",
                    showChevron: false,
                )
            }
            .padding(.top, 8)

            if !store.settings.system.enableLogRecording {
                Text(store.t("log_disabled_hint"))
                    .foregroundStyle(.secondary)
                    .padding(24)
                Spacer()
            } else {
                Picker("", selection: $tab) {
                    Text(store.t("log_status_title")).tag(0)
                    Text(store.t("log_connection_title")).tag(1)
                }
                .pickerStyle(.segmented)
                .padding(.horizontal, 16)
                .padding(.top, 12)

                let lines = tab == 0 ? store.statusLogs : store.connectionLogs
                let empty = tab == 0 ? store.t("log_status_empty") : store.t("log_connection_empty")
                ScrollView {
                    LazyVStack(alignment: .leading, spacing: 2) {
                        if lines.isEmpty {
                            Text(empty)
                                .font(.system(.caption, design: .monospaced))
                        } else {
                            ForEach(Array(lines.enumerated()), id: \.offset) { _, line in
                                Text(line)
                                    .font(.system(.caption, design: .monospaced))
                                    .frame(maxWidth: .infinity, alignment: .leading)
                            }
                        }
                    }
                    .padding(14)
                }
                .background(Color(.secondarySystemGroupedBackground))
                .clipShape(RoundedRectangle(cornerRadius: 12))
                .padding(16)
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
        .background(Color(.systemGroupedBackground))
        .navigationTitle(store.t("nav_logs"))
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            if store.settings.system.enableLogRecording {
                Button(store.t("log_clear")) {
                    if tab == 0 { store.clearStatusLogs() } else { store.clearConnectionLogs() }
                }
            }
        }
    }
}
