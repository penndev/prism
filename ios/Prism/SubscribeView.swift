import AVFoundation
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
    @State private var scanning = false

    var body: some View {
        VStack(spacing: 0) {
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
                }
                .padding(.vertical, 8)
                .padding(.bottom, 16)
            }
            if editing || !store.importPreview.isEmpty {
                VStack(spacing: 8) {
                    Button {
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
                    } label: {
                        Text(store.importPreview.isEmpty ? store.t("server_list_save") : store.t("subscribe_confirm"))
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.borderedProminent)
                    if editing {
                        Button(role: .destructive) {
                            pendingDelete = true
                        } label: {
                            Text(store.t("subscribe_delete"))
                                .frame(maxWidth: .infinity)
                        }
                        .buttonStyle(.bordered)
                    }
                }
                .padding(.horizontal, 16)
                .padding(.top, 8)
                .padding(.bottom, 12)
            }
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle(store.t("subscribe_title"))
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    startScan()
                } label: {
                    Image(systemName: "qrcode.viewfinder")
                }
                .accessibilityLabel(store.t("subscribe_scan"))
            }
        }
        .fullScreenCover(isPresented: $scanning) {
            QrScanner(
                prompt: store.t("subscribe_scan_prompt"),
                cancelTitle: store.t("subscribe_scan_cancel"),
                onCode: { text in
                    subscribeUrl = text
                    scanning = false
                    store.snack("subscribe_scan_saved")
                },
                onCancel: { scanning = false },
            )
            .ignoresSafeArea()
        }
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

    private func startScan() {
        guard AVCaptureDevice.default(for: .video) != nil else {
            store.snack("subscribe_scan_unavailable")
            return
        }
        switch AVCaptureDevice.authorizationStatus(for: .video) {
        case .authorized:
            scanning = true
        case .notDetermined:
            AVCaptureDevice.requestAccess(for: .video) { ok in
                DispatchQueue.main.async {
                    if ok {
                        scanning = true
                    } else {
                        store.snack("subscribe_scan_denied")
                    }
                }
            }
        default:
            store.snack("subscribe_scan_denied")
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

private struct QrScanner: UIViewControllerRepresentable {
    var prompt: String
    var cancelTitle: String
    var onCode: (String) -> Void
    var onCancel: () -> Void

    func makeUIViewController(context: Context) -> QrScannerController {
        let controller = QrScannerController()
        controller.prompt = prompt
        controller.cancelTitle = cancelTitle
        controller.onCode = onCode
        controller.onCancel = onCancel
        return controller
    }

    func updateUIViewController(_ uiViewController: QrScannerController, context: Context) {}
}

private final class QrScannerController: UIViewController, AVCaptureMetadataOutputObjectsDelegate {
    var prompt = ""
    var cancelTitle = ""
    var onCode: ((String) -> Void)?
    var onCancel: (() -> Void)?

    private let session = AVCaptureSession()
    private let sessionQueue = DispatchQueue(label: "prism.qr.session")
    private let preview = AVCaptureVideoPreviewLayer()
    private let output = AVCaptureMetadataOutput()
    private var started = false
    private var done = false

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .black
        preview.videoGravity = .resizeAspectFill
        view.layer.insertSublayer(preview, at: 0)

        let cancel = UIButton(type: .system)
        var cancelConfig = UIButton.Configuration.plain()
        cancelConfig.title = cancelTitle
        cancelConfig.baseForegroundColor = .white
        cancelConfig.contentInsets = NSDirectionalEdgeInsets(top: 12, leading: 16, bottom: 12, trailing: 16)
        cancelConfig.titleTextAttributesTransformer = UIConfigurationTextAttributesTransformer { incoming in
            var outgoing = incoming
            outgoing.font = .systemFont(ofSize: 17, weight: .semibold)
            return outgoing
        }
        cancel.configuration = cancelConfig
        cancel.addTarget(self, action: #selector(cancelTapped), for: .touchUpInside)
        cancel.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(cancel)

        let label = UILabel()
        label.text = prompt
        label.textColor = .white
        label.font = .systemFont(ofSize: 15)
        label.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(label)

        NSLayoutConstraint.activate([
            cancel.leadingAnchor.constraint(equalTo: view.safeAreaLayoutGuide.leadingAnchor),
            cancel.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
            label.centerXAnchor.constraint(equalTo: view.centerXAnchor),
            label.bottomAnchor.constraint(equalTo: view.safeAreaLayoutGuide.bottomAnchor, constant: -32),
        ])
    }

    override func viewDidLayoutSubviews() {
        super.viewDidLayoutSubviews()
        preview.frame = view.bounds
    }

    override func viewDidAppear(_ animated: Bool) {
        super.viewDidAppear(animated)
        guard !started else { return }
        started = true
        let session = session
        let output = output
        let preview = preview
        sessionQueue.async {
            session.beginConfiguration()
            session.sessionPreset = .high
            if let device = AVCaptureDevice.default(.builtInWideAngleCamera, for: .video, position: .back)
                ?? AVCaptureDevice.default(for: .video),
               let input = try? AVCaptureDeviceInput(device: device),
               session.canAddInput(input) {
                session.addInput(input)
            }
            if session.canAddOutput(output) {
                session.addOutput(output)
                output.setMetadataObjectsDelegate(self, queue: .main)
                if output.availableMetadataObjectTypes.contains(.qr) {
                    output.metadataObjectTypes = [.qr]
                }
            }
            preview.session = session
            session.commitConfiguration()
            if !session.isRunning { session.startRunning() }
        }
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        let session = session
        sessionQueue.async {
            if session.isRunning { session.stopRunning() }
        }
    }

    @objc private func cancelTapped() {
        onCancel?()
    }

    func metadataOutput(
        _ output: AVCaptureMetadataOutput,
        didOutput metadataObjects: [AVMetadataObject],
        from connection: AVCaptureConnection
    ) {
        guard !done,
              let code = metadataObjects.compactMap({ $0 as? AVMetadataMachineReadableCodeObject }).first?.stringValue
        else { return }
        done = true
        onCode?(code)
    }
}
