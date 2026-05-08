import SwiftUI

/// Real-time plethysmograph waveform rendered with SwiftUI Canvas,
/// matching the web UI's waveform-canvas visualisation.
struct WaveformView: View {
    let data: [Double]
    let color: Color
    let maxPoints: Int

    init(data: [Double], color: Color = Theme.pulse, maxPoints: Int = OximeterManager.maxWaveformPoints) {
        self.data = data
        self.color = color
        self.maxPoints = maxPoints
    }

    var body: some View {
        Canvas { context, size in
            guard data.count >= 2 else { return }

            let minVal = data.min()!
            let maxVal = data.max()!
            let range = max(maxVal - minVal, 10)
            let stepX = size.width / Double(maxPoints)

            var path = Path()
            for (i, value) in data.enumerated() {
                let x = Double(i) * stepX
                let normalised = (value - minVal) / range
                let y = size.height - (normalised * size.height * 0.8) - (size.height * 0.1)
                if i == 0 {
                    path.move(to: CGPoint(x: x, y: y))
                } else {
                    path.addLine(to: CGPoint(x: x, y: y))
                }
            }

            // Glow layer (wider, semi-transparent).
            context.addFilter(.shadow(color: color.opacity(0.5), radius: 8))
            context.stroke(
                path,
                with: .color(color),
                style: StrokeStyle(lineWidth: 2.5, lineCap: .round, lineJoin: .round)
            )
        }
    }
}
