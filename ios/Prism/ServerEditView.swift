import SwiftUI

struct ServerEditView: View {
    @EnvironmentObject private var store: Store
    @Environment(\.dismiss) private var dismiss
    let server: ServerItem?

    @State private var host = ""
    @State private var remark = ""
    @State private var protocolName = "socks5"
    @State private var username = ""
    @State private var password = ""

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
        .onAppear {
            host = server?.host ?? ""
            remark = server?.remark ?? ""
            protocolName = server?.protocolName ?? "socks5"
            username = server?.username ?? ""
            password = server?.password ?? ""
        }
    }
}
