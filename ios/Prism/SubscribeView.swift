import SwiftUI
import UniformTypeIdentifiers

struct SubscribeView: View {
    @EnvironmentObject private var store: Store
    @Environment(\.dismiss) private var dismiss
    @State private var subscribeType = ""
    @State private var subscribeUrl = ""
    @State private var confirmImport = false
    @State private var pickingType = false
    @State private var importingFile = false
    @State private var exporting = false

    var body: some View {
        ScrollView {
            VStack(spacing: 12) {
                PreferenceGroup(title: store.t("subscribe_import_title")) {
                    PreferenceRow(
                        title: store.t("subscribe_type"),
                        value: typeLabel,
                    ) { pickingType = true }
                    if subscribeType == "json" {
                        PreferenceDivider()
                        PreferenceRow(
                            title: store.t("subscribe_pick_file"),
                            description: store.t("subscribe_pick_file_hint"),
                        ) { importingFile = true }
                    } else if subscribeType == "Prism" || subscribeType == "Shadowrocket" {
                        PreferenceDivider()
                        TextField(store.t("subscribe_url_placeholder"), text: $subscribeUrl)
                            .textInputAutocapitalization(.never)
                            .autocorrectionDisabled()
                            .padding(.horizontal, 16)
                            .padding(.vertical, 12)
                        PreferenceDivider()
                        Button {
                            store.parseSubscription(type: subscribeType, source: subscribeUrl)
                        } label: {
                            if store.importing {
                                ProgressView()
                                    .frame(maxWidth: .infinity)
                            } else {
                                Text(store.t("subscribe_parse"))
                                    .frame(maxWidth: .infinity)
                            }
                        }
                        .buttonStyle(.borderedProminent)
                        .padding(16)
                        .disabled(store.importing)
                    }
                }

                if !store.importPreview.isEmpty {
                    PreferenceGroup(title: store.t("subscribe_preview_title")) {
                        VStack(alignment: .leading, spacing: 10) {
                            Text(store.t(
                                "subscribe_preview_meta",
                                store.importSource ?? store.t("subscribe_source_url"),
                                store.importPreview.count,
                            ))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            Button(store.t("subscribe_confirm")) {
                                confirmImport = true
                            }
                            .buttonStyle(.borderedProminent)
                            .frame(maxWidth: .infinity)
                        }
                        .padding(16)
                        ForEach(store.importPreview) { server in
                            PreferenceDivider()
                            VStack(alignment: .leading, spacing: 2) {
                                Text(server.displayName)
                                    .lineLimit(1)
                                Text("\(server.protocolName)  \(server.host)")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                    .lineLimit(1)
                            }
                            .padding(.horizontal, 16)
                            .padding(.vertical, 10)
                            .frame(maxWidth: .infinity, alignment: .leading)
                        }
                    }
                }

                PreferenceGroup(title: store.t("subscribe_list_title")) {
                    VStack(alignment: .leading, spacing: 10) {
                        Text(store.t("subscribe_current_count", store.servers.count))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        Button(store.t("subscribe_export")) {
                            if store.servers.isEmpty {
                                store.snack("subscribe_error_export_empty")
                            } else {
                                exporting = true
                            }
                        }
                        .buttonStyle(.bordered)
                        .frame(maxWidth: .infinity)
                    }
                    .padding(16)
                }
            }
            .padding(.vertical, 8)
            .padding(.bottom, 16)
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle(store.t("subscribe_title"))
        .navigationBarTitleDisplayMode(.inline)
        .confirmationDialog(store.t("subscribe_type"), isPresented: $pickingType, titleVisibility: .visible) {
            Button(store.t("subscribe_import_file")) {
                subscribeType = "json"
                importingFile = true
            }
            Button("Prism") { subscribeType = "Prism" }
            Button("Shadowrocket") { subscribeType = "Shadowrocket" }
            Button(store.t("server_list_delete_cancel"), role: .cancel) {}
        }
        .fileImporter(
            isPresented: $importingFile,
            allowedContentTypes: [.json, .plainText, .data],
        ) { result in
            switch result {
            case .success(let url):
                let accessed = url.startAccessingSecurityScopedResource()
                defer { if accessed { url.stopAccessingSecurityScopedResource() } }
                let data = try? Data(contentsOf: url)
                let text = data.flatMap { SubscriptionParser.decodeText($0) }?
                    .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
                if !text.isEmpty {
                    store.parseSubscription(type: "Prism", source: text, fromFile: true)
                } else {
                    store.snack("subscribe_error_file")
                }
            case .failure:
                store.snack("subscribe_error_file")
            }
        }
        .fileExporter(
            isPresented: $exporting,
            document: JSONTextDocument(text: store.exportServers()),
            contentType: .json,
            defaultFilename: "prism-servers",
        ) { result in
            if case .success = result {
                store.snack("subscribe_exported")
            } else if case .failure = result {
                store.snack("subscribe_error_file")
            }
        }
        .alert(store.t("subscribe_confirm_title"), isPresented: $confirmImport) {
            Button(store.t("server_list_delete_cancel"), role: .cancel) {}
            Button(store.t("subscribe_confirm_ok")) {
                store.confirmImport()
                dismiss()
            }
        } message: {
            Text(store.t("subscribe_confirm_content", store.importPreview.count))
        }
    }

    private var typeLabel: String {
        switch subscribeType {
        case "json": return store.t("subscribe_import_file")
        case "Prism": return "Prism"
        case "Shadowrocket": return "Shadowrocket"
        default: return store.t("subscribe_type_placeholder")
        }
    }
}

private struct JSONTextDocument: FileDocument {
    static var readableContentTypes: [UTType] { [.json] }
    var text: String

    init(text: String) { self.text = text }

    init(configuration: ReadConfiguration) throws {
        text = String(data: configuration.file.regularFileContents ?? Data(), encoding: .utf8) ?? ""
    }

    func fileWrapper(configuration: WriteConfiguration) throws -> FileWrapper {
        FileWrapper(regularFileWithContents: Data(text.utf8))
    }
}
