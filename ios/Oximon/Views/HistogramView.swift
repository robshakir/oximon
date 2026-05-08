import Charts
import SwiftUI

/// Distribution histogram with adaptive bin count based on the actual data range.
struct HistogramView: View {
    let data: [Double]
    let color: Color

    var body: some View {
        Chart {
            ForEach(bins) { bin in
                BarMark(
                    x: .value("Value", bin.label),
                    y: .value("Count", bin.count)
                )
                .foregroundStyle(
                    LinearGradient(
                        colors: [color, color.opacity(0.6)],
                        startPoint: .top,
                        endPoint: .bottom
                    )
                )
                .cornerRadius(2)
            }
        }
        .chartXAxis {
            AxisMarks { value in
                AxisValueLabel()
                    .foregroundStyle(Theme.textSecondary)
                    .font(.system(size: 9))
            }
        }
        .chartYAxis(.hidden)
    }

    // MARK: - Adaptive Binning

    private var bins: [Bin] {
        guard data.count >= 2 else { return [] }
        let minimum = data.min()!
        let maximum = data.max()!
        let intRange = Int(maximum) - Int(minimum)

        // Adaptive bin count: use integer-width bins for small ranges,
        // scale up for larger ranges, cap at 15.
        let numBins = max(min(intRange + 1, 15), 1)
        let range = (maximum - minimum) == 0 ? 1.0 : (maximum - minimum)
        let binSize = range / Double(numBins)

        var counts = [Int](repeating: 0, count: numBins)
        for value in data {
            var index = Int((value - minimum) / binSize)
            if index >= numBins { index = numBins - 1 }
            counts[index] += 1
        }

        return counts.enumerated().map { i, count in
            let lo = minimum + Double(i) * binSize
            let label: String
            if numBins <= 6 {
                // Show exact integer values for narrow ranges.
                label = "\(Int((lo + binSize / 2).rounded()))"
            } else {
                label = "\(Int(lo.rounded()))"
            }
            return Bin(id: i, label: label, count: count)
        }
    }
}

/// A single histogram bin.
struct Bin: Identifiable {
    let id: Int
    let label: String
    let count: Int
}
