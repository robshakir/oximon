import SwiftUI

/// Root dashboard view assembling the header, metric cards, and waveform —
/// a 1:1 layout match of the web UI's index.html.
///
/// Generic over `Source` so it works with both `OximeterManager` (real BLE)
/// and `SimulatedOximeterManager` (synthetic data for Simulator testing).
struct DashboardView<Source: OximeterDataSource>: View {
    @StateObject private var manager: Source

    /// Heartbeat animation trigger — toggled on each new pulse reading.
    @State private var isBeating = false
    /// Track the last pulse value to detect new readings.
    @State private var lastPulseCount = 0

    init(manager: @autoclosure @escaping () -> Source) {
        _manager = StateObject(wrappedValue: manager())
    }

    var body: some View {
        VStack(spacing: 12) {
            header
            metricsRow
            waveformCard
        }
        .padding(.horizontal, 16)
        .padding(.top, 8)
        .padding(.bottom, 8)
        .frame(maxHeight: .infinity, alignment: .top)
        .background(Theme.background.ignoresSafeArea())
        .onChange(of: manager.pulseHistory.count) { _, newCount in
            guard newCount > lastPulseCount else {
                lastPulseCount = newCount
                return
            }
            lastPulseCount = newCount
            triggerHeartbeat()
        }
    }

    // MARK: - Header

    private var header: some View {
        HStack {
            VStack(alignment: .leading, spacing: 2) {
                Text("Oximon")
                    .font(.system(size: 28, weight: .heavy))
                    .foregroundStyle(Theme.textPrimary)

                if let name = manager.peripheralName, manager.state == .connected {
                    Text(name)
                        .font(.system(size: 12, weight: .medium))
                        .foregroundStyle(Theme.textSecondary)
                }
            }

            Spacer()

            ConnectionStatusView(state: manager.state)
        }
    }

    // MARK: - Metrics Grid

    private var metricsRow: some View {
        HStack(spacing: 12) {
            MetricCardView(
                title: "SpO₂",
                value: manager.currentSpO2,
                unit: "%",
                color: Theme.spo2,
                history: manager.spo2History,
                trendMin: 90,
                trendMax: 100
            )

            MetricCardView(
                title: "Heart Rate",
                value: manager.currentPulse,
                unit: "bpm",
                color: Theme.pulse,
                history: manager.pulseHistory,
                trendMin: 50,
                trendMax: 120,
                isBeating: isBeating
            )
        }
    }

    // MARK: - Waveform

    private var waveformCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Plethysmograph Waveform")
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(Theme.textSecondary)
                .textCase(.uppercase)
                .tracking(1)

            WaveformView(data: manager.waveformBuffer)
                .frame(maxHeight: .infinity)
        }
        .cardStyle()
        .frame(maxHeight: .infinity)
    }

    // MARK: - Heartbeat Animation

    private func triggerHeartbeat() {
        withAnimation(.easeOut(duration: 0.1)) { isBeating = true }
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.15) {
            withAnimation(.easeIn(duration: 0.1)) { isBeating = false }
        }
    }
}

#Preview {
    DashboardView(manager: SimulatedOximeterManager())
}
