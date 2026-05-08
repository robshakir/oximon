import Charts
import SwiftUI

/// Line chart showing the recent trend of a metric, matching the web UI's
/// canvas-based trend visualisation with a glow effect.
struct TrendChartView: View {
    let data: [Double]
    let color: Color
    let defaultMin: Double
    let defaultMax: Double

    var body: some View {
        Chart {
            ForEach(Array(data.enumerated()), id: \.offset) { index, value in
                LineMark(
                    x: .value("Time", index),
                    y: .value("Value", value)
                )
                .foregroundStyle(color)
                .lineStyle(StrokeStyle(lineWidth: 2.5, lineCap: .round, lineJoin: .round))
                .interpolationMethod(.catmullRom)

                AreaMark(
                    x: .value("Time", index),
                    y: .value("Value", value)
                )
                .foregroundStyle(
                    LinearGradient(
                        colors: [color.opacity(0.25), color.opacity(0.0)],
                        startPoint: .top,
                        endPoint: .bottom
                    )
                )
                .interpolationMethod(.catmullRom)
            }
        }
        .chartXAxis(.hidden)
        .chartYAxis(.hidden)
        .chartXScale(domain: 0 ... max(OximeterManager.maxTrendPoints - 1, 1))
        .chartYScale(domain: yMin ... yMax)
        .clipped()
    }

    // MARK: - Y-axis domain

    private var yMin: Double {
        min(data.min() ?? defaultMin, defaultMin)
    }

    private var yMax: Double {
        let upper = max(data.max() ?? defaultMax, defaultMax)
        return upper <= yMin ? yMin + 5 : upper
    }
}
