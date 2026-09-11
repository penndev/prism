import SwiftUI

@main
struct PrismApp: App {
    @StateObject private var store = Store()

    var body: some Scene {
        WindowGroup {
            RootView()
                .environmentObject(store)
                .preferredColorScheme(Theme.colorScheme(store.settings.system.themeMode))
        }
    }
}
