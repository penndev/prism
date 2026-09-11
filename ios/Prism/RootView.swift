import SwiftUI

enum AppTab: Hashable {
    case home, rules, logs, settings
}

enum AppRoute: Hashable {
    case subscribe
    case serverEdit(id: String?)
    case domainRules
    case geoRules
}

struct RootView: View {
    @EnvironmentObject private var store: Store
    @State private var tab: AppTab = .home
    @State private var homePath: [AppRoute] = []
    @State private var rulesPath: [AppRoute] = []

    var body: some View {
        TabView(selection: $tab) {
            NavigationStack(path: $homePath) {
                HomeView(path: $homePath)
                    .navigationDestination(for: AppRoute.self) { route in
                        switch route {
                        case .subscribe:
                            SubscribeView()
                        case .serverEdit(let id):
                            ServerEditView(server: store.servers.first { $0.id == id })
                        default:
                            EmptyView()
                        }
                    }
            }
            .tabItem { Label(store.t("nav_home"), systemImage: "house") }
            .tag(AppTab.home)

            NavigationStack(path: $rulesPath) {
                RulesView(path: $rulesPath)
                    .navigationDestination(for: AppRoute.self) { route in
                        switch route {
                        case .domainRules: DomainRulesView()
                        case .geoRules: GeoRulesView()
                        default: EmptyView()
                        }
                    }
            }
            .tabItem { Label(store.t("nav_rules"), systemImage: "line.3.horizontal.decrease") }
            .tag(AppTab.rules)

            NavigationStack {
                LogsView()
            }
            .tabItem { Label(store.t("nav_logs"), systemImage: "list.bullet") }
            .tag(AppTab.logs)

            NavigationStack {
                SettingsView()
            }
            .tabItem { Label(store.t("nav_settings"), systemImage: "gearshape") }
            .tag(AppTab.settings)
        }
        .tint(.prismBlue)
        .overlay(alignment: .bottom) {
            if let snackbar = store.snackbar {
                Text(snackbar)
                    .font(.subheadline)
                    .padding(.horizontal, 16)
                    .padding(.vertical, 10)
                    .background(.ultraThinMaterial, in: Capsule())
                    .padding(.bottom, 56)
                    .onAppear {
                        DispatchQueue.main.asyncAfter(deadline: .now() + 2) {
                            store.consumeSnackbar()
                        }
                    }
                    .id(snackbar)
            }
        }
    }
}
