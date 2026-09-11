import SwiftUI

extension Color {
    static let prismBlue = Color(red: 0, green: 136 / 255, blue: 204 / 255)
    static let prismBlueDark = Color(red: 51 / 255, green: 163 / 255, blue: 217 / 255)
    static let latencyGood = Color(red: 82 / 255, green: 196 / 255, blue: 26 / 255)
    static let latencyMedium = Color(red: 250 / 255, green: 173 / 255, blue: 20 / 255)
    static let latencyBad = Color(red: 1, green: 77 / 255, blue: 79 / 255)
}

enum Theme {
    static func colorScheme(_ mode: ThemeMode) -> ColorScheme? {
        switch mode {
        case .system: return nil
        case .light: return .light
        case .dark: return .dark
        }
    }
}
