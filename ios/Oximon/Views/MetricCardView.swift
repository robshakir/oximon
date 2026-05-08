import SwiftUI

/// Reusable metric card displaying a live value, min/mean/max statistics,
/// trend chart, and distribution histogram — matching the web UI cards.
struct MetricCardView: View {
    let title: String
    let value: Int
    let unit: String
    let color: Color
    let history: [Double]
    let trendMin: Double
    let trendMax: Double

    /// Optional heartbeat animation trigger.
    var isBeating: Bool = false

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            // Title
            Text(title)
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(Theme.textSecondary)
                .textCase(.uppercase)
                .tracking(0.8)

            // Big value
            HStack(alignment: .firstTextBaseline, spacing: 4) {
                Text(value > 0 ? "\(value)" : "--")
                    .font(.system(size: 40, weight: .heavy, design: .rounded))
                    .foregroundStyle(Theme.textPrimary)
                    .contentTransition(.numericText(value: Double(value)))
                    .animation(.easeInOut(duration: 0.3), value: value)
                    .scaleEffect(isBeating ? 1.08 : 1.0)
                    .animation(.easeOut(duration: 0.15), value: isBeating)

                Text(unit)
                    .font(.system(size: 18, weight: .semibold))
                    .foregroundStyle(Theme.textSecondary)
            }

            // Min / Mean / Max
            if !history.isEmpty {
                HStack(spacing: 0) {
                    StatItem(label: "Min", value: Int(history.min()!))
                    Spacer()
                    StatItem(label: "Mean", value: Int(history.reduce(0, +) / Double(history.count)))
                    Spacer()
                    StatItem(label: "Max", value: Int(history.max()!))
                }
            }

            // Trend chart
            TrendChartView(
                data: history,
                color: color,
                defaultMin: trendMin,
                defaultMax: trendMax
            )
            .frame(height: 60)

            // Histogram
            HistogramView(data: history, color: color)
                .frame(height: 50)
        }
        .cardStyle()
    }
}

/// A single stat value with label, used in the Min/Mean/Max row.
private struct StatItem: View {
    let label: String
    let value: Int

    var body: some View {
        VStack(spacing: 2) {
            Text(label)
                .font(.system(size: 10, weight: .medium))
                .foregroundStyle(Theme.textSecondary)
                .textCase(.uppercase)
                .tracking(0.5)
            Text("\(value)")
                .font(.system(size: 16, weight: .bold, design: .rounded))
                .foregroundStyle(Theme.textPrimary)
                .contentTransition(.numericText(value: Double(value)))
                .animation(.easeInOut(duration: 0.3), value: value)
        }
    }
}
