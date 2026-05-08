import SwiftUI

/// Connection status pill matching the web UI's indicator badge.
struct ConnectionStatusView: View {
    let state: OximeterManager.ConnectionState

    var body: some View {
        HStack(spacing: 6) {
            Circle()
                .fill(dotColor)
                .frame(width: 8, height: 8)
                .shadow(color: dotColor.opacity(0.6), radius: 4)

            Text(state.rawValue)
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(Theme.textSecondary)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 6)
        .background(
            Capsule()
                .fill(Theme.cardBackground)
                .overlay(Capsule().stroke(Theme.cardBorder, lineWidth: 1))
        )
    }

    private var dotColor: Color {
        switch state {
        case .connected:
            Theme.connectedGreen
        case .scanning, .connecting, .discovering, .calibrating:
            .orange
        case .disconnected, .poweredOff:
            Theme.pulse
        }
    }
}
