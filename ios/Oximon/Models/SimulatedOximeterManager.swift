import Combine
import Foundation

/// A simulated oximeter that generates realistic synthetic telemetry data,
/// allowing full UI testing in the iOS Simulator or SwiftUI Previews
/// where CoreBluetooth is unavailable.
final class SimulatedOximeterManager: ObservableObject, OximeterDataSource {

    // MARK: - Published Properties (same interface as OximeterManager)

    @Published var state: OximeterManager.ConnectionState = .disconnected
    @Published var currentSpO2: Int = 0
    @Published var currentPulse: Int = 0
    @Published var spo2History: [Double] = []
    @Published var pulseHistory: [Double] = []
    @Published var waveformBuffer: [Double] = []
    @Published var peripheralName: String? = "Simulated Oximeter"

    // MARK: - Private

    private var waveformTimer: Timer?
    private var readingTimer: Timer?
    private var phase: Double = 0

    /// Base SpO2 wanders slowly around 97.
    private var spo2Base: Double = 97
    /// Base pulse wanders slowly around 72.
    private var pulseBase: Double = 72

    // MARK: - Lifecycle

    init() {
        // Simulate the connection sequence with realistic delays.
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) { [weak self] in
            self?.state = .scanning
        }
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.5) { [weak self] in
            self?.state = .connecting
        }
        DispatchQueue.main.asyncAfter(deadline: .now() + 2.0) { [weak self] in
            self?.state = .discovering
        }
        DispatchQueue.main.asyncAfter(deadline: .now() + 2.5) { [weak self] in
            self?.state = .connected
            self?.startGenerating()
        }
    }

    deinit {
        waveformTimer?.invalidate()
        readingTimer?.invalidate()
    }

    // MARK: - Data Generation

    private func startGenerating() {
        // Waveform at ~50 Hz (matching the real oximeter).
        waveformTimer = Timer.scheduledTimer(withTimeInterval: 0.02, repeats: true) { [weak self] _ in
            self?.generateWaveformSample()
        }

        // Readings at ~1 Hz (matching the real oximeter).
        readingTimer = Timer.scheduledTimer(withTimeInterval: 1.0, repeats: true) { [weak self] _ in
            self?.generateReading()
        }
    }

    /// Generates a plethysmograph-like waveform: a sharp systolic peak followed
    /// by a dicrotic notch, similar to a real photoplethysmogram.
    private func generateWaveformSample() {
        phase += 0.08
        if phase > .pi * 2 { phase -= .pi * 2 }

        // Composite waveform: main pulse + dicrotic notch + baseline noise.
        let systolic = max(0, sin(phase)) * 180
        let dicrotic = max(0, sin(phase * 2 - 1.2)) * 40
        let noise = Double.random(in: -3...3)
        let baseline = 30.0

        let value = baseline + systolic + dicrotic + noise

        waveformBuffer.append(value)
        if waveformBuffer.count > OximeterManager.maxWaveformPoints {
            waveformBuffer.removeFirst()
        }
    }

    /// Generates realistic SpO2 and pulse readings with slow physiological drift.
    private func generateReading() {
        // Random walk for physiological realism.
        spo2Base += Double.random(in: -0.3...0.3)
        spo2Base = min(max(spo2Base, 94), 100)

        pulseBase += Double.random(in: -1.0...1.0)
        pulseBase = min(max(pulseBase, 58), 95)

        currentSpO2 = Int(spo2Base.rounded())
        currentPulse = Int(pulseBase.rounded())

        spo2History.append(Double(currentSpO2))
        pulseHistory.append(Double(currentPulse))

        if spo2History.count > OximeterManager.maxTrendPoints {
            spo2History.removeFirst()
        }
        if pulseHistory.count > OximeterManager.maxTrendPoints {
            pulseHistory.removeFirst()
        }
    }

    // MARK: - Public API (matching OximeterManager)

    func startScanning() {
        // No-op in simulation.
    }

    func disconnect() {
        waveformTimer?.invalidate()
        readingTimer?.invalidate()
        state = .disconnected
    }
}
