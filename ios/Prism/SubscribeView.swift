import SwiftUI
import UIKit

struct SubscribeView: View {
    @EnvironmentObject private var store: Store
    @Environment(\.dismiss) private var dismiss
    let editing: Bool

    @State private var subscribeType = ""
    @State private var subscribeUrl = ""
    @State private var subscribeRemark = ""
    @State private var pickingType = false
    @State private var pendingDelete = false
    @State private var filled = false

    var body: some View {
        ScrollViewReader { proxy in
            ScrollView {
                VStack(spacing: 12) {
                    VStack(spacing: 0) {
                        PreferenceRow(
                            title: store.t("subscribe_type"),
                            value: subscribeType.isEmpty ? store.t("subscribe_type_placeholder") : subscribeType,
                        ) { pickingType = true }
                        PreferenceDivider()
                        labeledField(
                            store.t("subscribe_url_label"),
                            text: $subscribeUrl,
                            placeholder: store.t("subscribe_url_placeholder"),
                        )
                        PreferenceDivider()
                        labeledField(
                            store.t("subscribe_remark"),
                            text: $subscribeRemark,
                            placeholder: store.t("server_list_remark_placeholder"),
                        )
                    }
                    .background(Color(.secondarySystemGroupedBackground))
                    .clipShape(RoundedRectangle(cornerRadius: 12))
                    .padding(.horizontal, 16)

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
                    .buttonStyle(.bordered)
                    .padding(.horizontal, 16)
                    .disabled(store.importing)

                    if !store.importPreview.isEmpty {
                        Text(store.t(
                            "subscribe_preview_meta",
                            store.importSource ?? store.t("subscribe_source_url"),
                            store.importPreview.count,
                        ))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(.horizontal, 20)
                        VStack(spacing: 0) {
                            ForEach(Array(store.importPreview.enumerated()), id: \.element.id) { index, server in
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
                                if index < store.importPreview.count - 1 {
                                    PreferenceDivider()
                                }
                            }
                        }
                        .background(Color(.secondarySystemGroupedBackground))
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                        .padding(.horizontal, 16)
                    }

                    if editing || !store.importPreview.isEmpty {
                        VStack(spacing: 8) {
                            Button(store.importPreview.isEmpty ? store.t("server_list_save") : store.t("subscribe_confirm")) {
                                if store.importPreview.isEmpty {
                                    store.updateSubscriptionInfo(
                                        type: subscribeType,
                                        url: subscribeUrl,
                                        remark: subscribeRemark,
                                    )
                                } else {
                                    store.confirmImport(
                                        type: subscribeType,
                                        url: subscribeUrl,
                                        remark: subscribeRemark,
                                    )
                                }
                                dismiss()
                            }
                            .buttonStyle(.borderedProminent)
                            .frame(maxWidth: .infinity)
                            if editing {
                                Button(store.t("subscribe_delete"), role: .destructive) {
                                    pendingDelete = true
                                }
                                .buttonStyle(.borderedProminent)
                                .tint(.red)
                                .frame(maxWidth: .infinity)
                            }
                        }
                        .padding(.horizontal, 16)
                        .padding(.top, 8)
                        .id("save")
                    }
                }
                .padding(.vertical, 8)
                .padding(.bottom, 16)
            }
            .onChange(of: store.importPreview) { preview in
                guard !preview.isEmpty else { return }
                UIApplication.shared.sendAction(
                    #selector(UIResponder.resignFirstResponder),
                    to: nil,
                    from: nil,
                    for: nil,
                )
                withAnimation {
                    proxy.scrollTo("save", anchor: .bottom)
                }
            }
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle(store.t("subscribe_title"))
        .navigationBarTitleDisplayMode(.inline)
        .confirmationDialog(store.t("subscribe_type"), isPresented: $pickingType, titleVisibility: .visible) {
            Button("Prism") { subscribeType = "Prism" }
            Button("Shadowrocket") { subscribeType = "Shadowrocket" }
            Button(store.t("server_list_delete_cancel"), role: .cancel) {}
        }
        .alert(store.t("subscribe_delete_title"), isPresented: $pendingDelete) {
            Button(store.t("server_list_delete_cancel"), role: .cancel) {}
            Button(store.t("server_list_delete_ok"), role: .destructive) {
                store.deleteSubscription()
                dismiss()
            }
        } message: {
            Text(store.t("subscribe_delete_content"))
        }
        .onAppear {
            guard editing, !filled, let sub = store.subscription else { return }
            filled = true
            subscribeType = sub.type
            subscribeUrl = sub.url
            subscribeRemark = sub.remark
        }
    }

    private func labeledField(_ label: String, text: Binding<String>, placeholder: String) -> some View {
        HStack(spacing: 12) {
            Text(label)
                .frame(width: 80, alignment: .leading)
            TextField(placeholder, text: text)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 14)
    }
}
