import SwiftUI

struct SettingsView: View {
    @EnvironmentObject private var store: Store
    @State private var dialog: SettingsDialog = .none
    @State private var latencyDraft = ""

    var body: some View {
        ScrollView {
            VStack(spacing: 16) {
                PreferenceGroup(title: store.t("settings_latency_test")) {
                    PreferenceRow(
                        title: store.t("settings_latency_test_host"),
                        value: store.settings.latencyTest.host.isEmpty ? "—" : store.settings.latencyTest.host,
                    ) { dialog = .latencyHost }
                    PreferenceDivider()
                    PreferenceSwitch(
                        title: store.t("settings_sort_by_latency"),
                        description: store.t("settings_sort_by_latency_desc"),
                        isOn: Binding(
                            get: { store.settings.latencyTest.sortAfterPing },
                            set: { checked in
                                store.updateLatencySettings {
                                    var next = $0
                                    next.sortAfterPing = checked
                                    return next
                                }
                            },
                        ),
                    )
                }
                PreferenceGroup(title: store.t("settings_system")) {
                    PreferenceRow(
                        title: store.t("settings_system_language"),
                        value: languageLabel,
                    ) { dialog = .language }
                    PreferenceDivider()
                    PreferenceRow(
                        title: store.t("settings_theme"),
                        value: themeLabel,
                    ) { dialog = .theme }
                    PreferenceDivider()
                    PreferenceSwitch(
                        title: store.t("settings_enable_log"),
                        description: store.t("settings_enable_log_desc"),
                        isOn: Binding(
                            get: { store.settings.system.enableLogRecording },
                            set: { checked in
                                store.updateSystemSettings {
                                    var next = $0
                                    next.enableLogRecording = checked
                                    return next
                                }
                            },
                        ),
                    )
                }
            }
            .padding(.vertical, 8)
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle(store.t("nav_settings"))
        .navigationBarTitleDisplayMode(.inline)
        .confirmationDialog(store.t("settings_select_language"), isPresented: Binding(
            get: { dialog == .language },
            set: { if !$0 { dialog = .none } },
        ), titleVisibility: .visible) {
            Button(store.t("lang_system")) { setLanguage(languageSystem) }
            Button(store.t("lang_zh")) { setLanguage("zh-CN") }
            Button(store.t("lang_en")) { setLanguage("en") }
            Button(store.t("server_list_delete_cancel"), role: .cancel) { dialog = .none }
        }
        .confirmationDialog(store.t("settings_theme"), isPresented: Binding(
            get: { dialog == .theme },
            set: { if !$0 { dialog = .none } },
        ), titleVisibility: .visible) {
            Button(store.t("settings_theme_system")) { setTheme(.system) }
            Button(store.t("settings_theme_light")) { setTheme(.light) }
            Button(store.t("settings_theme_dark")) { setTheme(.dark) }
            Button(store.t("server_list_delete_cancel"), role: .cancel) { dialog = .none }
        }
        .alert(store.t("settings_latency_test_host"), isPresented: Binding(
            get: { dialog == .latencyHost },
            set: { if !$0 { dialog = .none } },
        )) {
            TextField(store.t("settings_latency_test_host_placeholder"), text: $latencyDraft)
            Button(store.t("settings_confirm")) {
                store.updateLatencySettings {
                    var next = $0
                    next.host = latencyDraft.trimmingCharacters(in: .whitespacesAndNewlines)
                    return next
                }
                dialog = .none
            }
            Button(store.t("server_list_delete_cancel"), role: .cancel) { dialog = .none }
        }
        .onChange(of: dialog) { value in
            if value == .latencyHost {
                latencyDraft = store.settings.latencyTest.host
            }
        }
    }

    private var languageLabel: String {
        switch store.settings.system.language {
        case "zh-CN": return store.t("lang_zh")
        case "en": return store.t("lang_en")
        default: return store.t("lang_system")
        }
    }

    private var themeLabel: String {
        switch store.settings.system.themeMode {
        case .system: return store.t("settings_theme_system")
        case .light: return store.t("settings_theme_light")
        case .dark: return store.t("settings_theme_dark")
        }
    }

    private func setLanguage(_ tag: String) {
        store.updateSystemSettings {
            var next = $0
            next.language = tag
            return next
        }
        dialog = .none
    }

    private func setTheme(_ mode: ThemeMode) {
        store.updateSystemSettings {
            var next = $0
            next.themeMode = mode
            return next
        }
        dialog = .none
    }
}

private enum SettingsDialog { case none, language, theme, latencyHost }
