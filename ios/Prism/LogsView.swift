import SwiftUI
import UIKit

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
                    Text(store.t("log_dns_title")).tag(2)
                }
                .pickerStyle(.segmented)
                .padding(.horizontal, 16)
                .padding(.top, 12)

                let lines = tab == 0 ? store.statusLogs : (tab == 1 ? store.connectionLogs : store.dnsLogs)
                let empty = tab == 0 ? store.t("log_status_empty") : (tab == 1 ? store.t("log_connection_empty") : store.t("log_dns_empty"))
                SelectableLog(lines: lines.isEmpty ? [empty] : lines)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
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
                ToolbarItem(placement: .topBarTrailing) {
                    Button(store.t("log_clear")) {
                        if tab == 0 { store.clearStatusLogs() }
                        else if tab == 1 { store.clearConnectionLogs() }
                        else { store.clearDnsLogs() }
                    }
                }
            }
        }
    }
}

private struct SelectableLog: UIViewRepresentable {
    let lines: [String]

    func makeUIView(context: Context) -> LogListView {
        LogListView()
    }

    func updateUIView(_ view: LogListView, context: Context) {
        view.setLines(lines)
    }

    func sizeThatFits(_ proposal: ProposedViewSize, uiView: LogListView, context: Context) -> CGSize? {
        guard let width = proposal.width, let height = proposal.height, width > 0, height > 0 else { return nil }
        return CGSize(width: width, height: height)
    }
}

private final class LogListView: UIScrollView {
    private let textView = UITextView()
    private var shown: [String] = []
    private var contentDirty = true
    private var lastBounds: CGSize = .init(width: -1, height: -1)

    private static let attrs: [NSAttributedString.Key: Any] = {
        let style = NSMutableParagraphStyle()
        style.lineBreakMode = .byClipping
        return [
            .font: UIFont.monospacedSystemFont(ofSize: 12, weight: .regular),
            .foregroundColor: UIColor.label,
            .paragraphStyle: style,
        ]
    }()

    override init(frame: CGRect) {
        super.init(frame: frame)
        bounces = false
        alwaysBounceVertical = false
        alwaysBounceHorizontal = false
        isDirectionalLockEnabled = true
        showsHorizontalScrollIndicator = true
        showsVerticalScrollIndicator = true
        contentInsetAdjustmentBehavior = .never
        backgroundColor = .clear

        textView.isEditable = false
        textView.isSelectable = true
        textView.isScrollEnabled = false
        textView.backgroundColor = .clear
        textView.textContainerInset = UIEdgeInsets(top: 14, left: 10, bottom: 14, right: 10)
        textView.textContainer.lineFragmentPadding = 0
        textView.textContainer.widthTracksTextView = false
        textView.textContainer.lineBreakMode = .byClipping
        textView.autoresizingMask = []
        addSubview(textView)
        textView.panGestureRecognizer.isEnabled = false
    }

    required init?(coder: NSCoder) { nil }

    func setLines(_ lines: [String]) {
        if lines != shown {
            let old = shown
            shown = lines
            if let first = lines.first, let k = old.firstIndex(of: first),
               old.count - k <= lines.count,
               old[k...].elementsEqual(lines[..<(old.count - k)]) {
                let removed = old[..<k].reduce(0) { $0 + ($1 as NSString).length + 1 }
                let added = lines[(old.count - k)...].map { "\n" + $0 }.joined()
                var sel = textView.selectedRange
                textView.textStorage.beginEditing()
                textView.textStorage.deleteCharacters(in: NSRange(location: 0, length: removed))
                textView.textStorage.append(NSAttributedString(string: added, attributes: Self.attrs))
                textView.textStorage.endEditing()
                if sel.location >= removed {
                    sel.location -= removed
                } else {
                    sel = NSRange(location: 0, length: 0)
                }
                textView.selectedRange = sel
            } else {
                textView.attributedText = NSAttributedString(string: lines.joined(separator: "\n"), attributes: Self.attrs)
            }
            contentDirty = true
        }
        applyContentSize()
    }

    override func layoutSubviews() {
        super.layoutSubviews()
        if bounds.size != lastBounds { applyContentSize() }
    }

    private func applyContentSize() {
        if !contentDirty, bounds.size == lastBounds { return }
        contentDirty = false
        lastBounds = bounds.size
        textView.panGestureRecognizer.isEnabled = false
        let font = UIFont.monospacedSystemFont(ofSize: 12, weight: .regular)
        var textWidth: CGFloat = 1
        for line in shown {
            textWidth = max(textWidth, (line as NSString).size(withAttributes: [.font: font]).width)
        }
        let inset = textView.textContainerInset
        let contentWidth = textWidth + inset.left + inset.right
        textView.textContainer.size = CGSize(width: textWidth, height: 1_000_000)
        let fit = textView.sizeThatFits(CGSize(width: contentWidth, height: 1_000_000))
        let width = max(fit.width, bounds.width)
        let height = max(fit.height, bounds.height)
        let size = CGSize(width: width, height: height)
        if textView.frame.size != size {
            textView.frame = CGRect(origin: .zero, size: size)
        }
        if contentSize != size {
            contentSize = size
        }
    }
}
