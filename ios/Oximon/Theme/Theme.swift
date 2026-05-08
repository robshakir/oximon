import SwiftUI

/// Design tokens matching the oximon web UI's dark slate aesthetic.
enum Theme {
    // MARK: - Backgrounds
    static let background = Color(red: 0.06, green: 0.09, blue: 0.16)       // #0f172a
    static let cardBackground = Color(red: 0.12, green: 0.16, blue: 0.23)   // #1e293b
    static let cardBorder = Color(red: 0.20, green: 0.25, blue: 0.35)       // #334155

    // MARK: - Accent Colours
    static let spo2 = Color(red: 0.05, green: 0.65, blue: 0.91)            // #0ea5e9
    static let pulse = Color(red: 0.94, green: 0.27, blue: 0.27)           // #ef4444
    static let connectedGreen = Color(red: 0.20, green: 0.83, blue: 0.60)  // #34d399

    // MARK: - Text
    static let textPrimary = Color(red: 0.89, green: 0.91, blue: 0.94)     // #e2e8f0
    static let textSecondary = Color(red: 0.58, green: 0.64, blue: 0.72)   // #94a3b8

    // MARK: - Card Style
    static let cardCornerRadius: CGFloat = 20
    static let cardPadding: CGFloat = 16

    /// Standard glassmorphic card background modifier.
    struct CardModifier: ViewModifier {
        func body(content: Content) -> some View {
            content
                .padding(Theme.cardPadding)
                .background(
                    RoundedRectangle(cornerRadius: Theme.cardCornerRadius)
                        .fill(Theme.cardBackground)
                        .overlay(
                            RoundedRectangle(cornerRadius: Theme.cardCornerRadius)
                                .stroke(Theme.cardBorder, lineWidth: 1)
                        )
                )
        }
    }
}

extension View {
    func cardStyle() -> some View {
        modifier(Theme.CardModifier())
    }
}
