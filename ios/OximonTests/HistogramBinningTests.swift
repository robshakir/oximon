import Foundation
import Testing

@testable import Oximon

/// Tests for the adaptive histogram binning logic.
struct HistogramBinningTests {

    @Test("Single value produces one bin")
    func singleValue() {
        let data = [97.0, 97.0, 97.0, 97.0]
        let bins = computeBins(data: data)
        #expect(bins.count == 1)
        #expect(bins[0].count == 4)
    }

    @Test("Two distinct values produce two bins")
    func twoValues() {
        let data = [97.0, 97.0, 98.0, 98.0]
        let bins = computeBins(data: data)
        #expect(bins.count == 2)
    }

    @Test("Narrow SpO₂ range (98-100) produces 3 bins, not 15")
    func narrowRange() {
        let data = [98.0, 99.0, 99.0, 100.0, 99.0, 100.0]
        let bins = computeBins(data: data)
        // Range is 2, so bins = min(2+1, 15) = 3
        #expect(bins.count == 3)
        // All data should be accounted for.
        let totalCount = bins.reduce(0) { $0 + $1.count }
        #expect(totalCount == data.count)
    }

    @Test("Wide heart rate range uses more bins")
    func wideRange() {
        let data = Array(stride(from: 60.0, through: 90.0, by: 1.0))
        let bins = computeBins(data: data)
        // Range is 30, so bins = min(30+1, 15) = 15
        #expect(bins.count == 15)
    }

    @Test("Empty data returns no bins")
    func emptyData() {
        let bins = computeBins(data: [])
        #expect(bins.isEmpty)
    }

    @Test("Insufficient data (1 point) returns no bins")
    func insufficientData() {
        let bins = computeBins(data: [97.0])
        #expect(bins.isEmpty)
    }

    @Test("All counts sum to data count")
    func countsMatchDataCount() {
        let data = [95.0, 96.0, 97.0, 97.0, 98.0, 98.0, 98.0, 99.0, 99.0, 100.0]
        let bins = computeBins(data: data)
        let totalCount = bins.reduce(0) { $0 + $1.count }
        #expect(totalCount == data.count)
    }

    // MARK: - Helper

    /// Replicates the HistogramView binning logic for testability.
    private func computeBins(data: [Double]) -> [Bin] {
        guard data.count >= 2 else { return [] }
        let minimum = data.min()!
        let maximum = data.max()!
        let intRange = Int(maximum) - Int(minimum)

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
                label = "\(Int((lo + binSize / 2).rounded()))"
            } else {
                label = "\(Int(lo.rounded()))"
            }
            return Bin(id: i, label: label, count: count)
        }
    }
}
