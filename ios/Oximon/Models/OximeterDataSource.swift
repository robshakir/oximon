import Combine
import CoreBluetooth
import Foundation

/// Protocol defining the interface shared by both the real and simulated oximeter managers.
/// This allows DashboardView to work with either implementation.
protocol OximeterDataSource: ObservableObject {
    var state: OximeterManager.ConnectionState { get }
    var currentSpO2: Int { get }
    var currentPulse: Int { get }
    var spo2History: [Double] { get }
    var pulseHistory: [Double] { get }
    var waveformBuffer: [Double] { get }
    var peripheralName: String? { get }

    func startScanning()
    func disconnect()
}

// Conform the real manager to the protocol.
extension OximeterManager: OximeterDataSource {}
