import SwiftUI
import UniformTypeIdentifiers

struct ServerEditView: View {
    @EnvironmentObject private var store: Store
    @Environment(\.dismiss) private var dismiss
    let server: ServerItem?

    @State private var host = ""
    @State private var remark = ""
    @State private var protocolName = "socks5"
    @State private var username = ""
    @State private var password = ""
    @State private var importing = false

    var body: some View {
        Form {
            TextField(store.t("server_list_host"), text: $host)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
            TextField(store.t("server_list_remark"), text: $remark)
            Picker(store.t("server_list_protocol"), selection: $protocolName) {
                ForEach(proxySchemes, id: \.self) { Text($0).tag($0) }
            }
            TextField(store.t("settings_username"), text: $username)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
            SecureField(store.t("settings_password"), text: $password)
            Button(store.t("server_list_save")) {
                let ok = store.addOrUpdateServer(
                    editingId: server?.id,
                    host: host,
                    remark: remark,
                    protocolName: protocolName,
                    username: username,
                    password: password,
                )
                if ok { dismiss() }
            }
        }
        .navigationTitle(server == nil ? store.t("server_list_add_title") : store.t("server_list_edit_title"))
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            if server == nil {
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        importing = true
                    } label: {
                        Image(systemName: "doc")
                    }
                    .accessibilityLabel(store.t("subscribe_import_file"))
                }
            }
        }
        .fileImporter(
            isPresented: $importing,
            allowedContentTypes: [.json, .plainText, .data],
        ) { result in
            switch result {
            case .success(let url):
                let accessed = url.startAccessingSecurityScopedResource()
                defer { if accessed { url.stopAccessingSecurityScopedResource() } }
                let data = try? Data(contentsOf: url)
                let text = data.flatMap { SubscriptionParser.decodeText($0) }?
                    .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
                if text.isEmpty {
                    store.snack("subscribe_error_file")
                } else {
                    store.importManualNodes(text) { dismiss() }
                }
            case .failure(let error):
                let ns = error as NSError
                if ns.domain == NSCocoaErrorDomain && ns.code == NSUserCancelledError { return }
                store.snack("subscribe_error_file")
            }
        }
        .onAppear {
            host = server?.host ?? ""
            remark = server?.remark ?? ""
            protocolName = server?.protocolName ?? "socks5"
            username = server?.username ?? ""
            password = server?.password ?? ""
        }
    }
}
