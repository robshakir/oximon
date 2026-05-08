import CoreBluetooth
import Foundation

/// Manages the CoreBluetooth lifecycle for connecting to a BerryMed/Innovo
/// pulse oximeter and publishing decoded telemetry to SwiftUI views.
final class OximeterManager: NSObject, ObservableObject {

    // MARK: - BLE Identifiers

    /// BerryMed oximeter service UUID (short form of 0000FFF0-0000-1000-8000-00805F9B34FB).
    static let serviceUUID = CBUUID(string: "FFF0")
    /// Notification characteristic carrying waveform and reading packets.
    static let characteristicUUID = CBUUID(string: "FFF1")

    // MARK: - Buffer Limits

    /// Number of trend readings to retain for charts.
    static let maxTrendPoints = 60
    /// Number of waveform samples to retain for the plethysmograph.
    static let maxWaveformPoints = 300

    // MARK: - Connection State

    enum ConnectionState: String {
        case poweredOff    = "Bluetooth Off"
        case disconnected  = "Disconnected"
        case scanning      = "Scanning…"
        case connecting    = "Connecting…"
        case discovering   = "Discovering…"
        case connected     = "Connected"
        case calibrating   = "Calibrating…"
    }

    // MARK: - Published Properties

    @Published var state: ConnectionState = .disconnected
    @Published var currentSpO2: Int = 0
    @Published var currentPulse: Int = 0
    @Published var spo2History: [Double] = []
    @Published var pulseHistory: [Double] = []
    @Published var waveformBuffer: [Double] = []
    @Published var peripheralName: String?

    // MARK: - Private

    private var centralManager: CBCentralManager!
    private var peripheral: CBPeripheral?

    // MARK: - Lifecycle

    override init() {
        super.init()
        centralManager = CBCentralManager(delegate: self, queue: .main)
    }

    /// Known device name patterns for BerryMed/Innovo pulse oximeters.
    static let knownNamePatterns = ["ip900", "berrymed", "berry", "oximeter", "innovo"]

    /// Begin scanning for oximeter peripherals.
    /// Scans for all BLE devices since many oximeters don't advertise service UUIDs.
    func startScanning() {
        guard centralManager.state == .poweredOn else { return }
        state = .scanning
        centralManager.scanForPeripherals(
            withServices: nil,
            options: [CBCentralManagerScanOptionAllowDuplicatesKey: false]
        )
    }

    /// Disconnect from the current peripheral and stop scanning.
    func disconnect() {
        centralManager.stopScan()
        if let peripheral {
            centralManager.cancelPeripheralConnection(peripheral)
        }
        state = .disconnected
    }
}

// MARK: - CBCentralManagerDelegate

extension OximeterManager: CBCentralManagerDelegate {

    func centralManagerDidUpdateState(_ central: CBCentralManager) {
        switch central.state {
        case .poweredOn:
            startScanning()
        case .poweredOff:
            state = .poweredOff
        default:
            state = .disconnected
        }
    }

    func centralManager(
        _ central: CBCentralManager,
        didDiscover peripheral: CBPeripheral,
        advertisementData: [String: Any],
        rssi RSSI: NSNumber
    ) {
        // Filter by name — only connect to known oximeter devices.
        guard let name = peripheral.name?.lowercased(),
              Self.knownNamePatterns.contains(where: { name.contains($0) }) else {
            return
        }

        self.peripheral = peripheral
        self.peripheralName = peripheral.name
        central.stopScan()
        state = .connecting
        central.connect(peripheral, options: nil)
    }

    func centralManager(_ central: CBCentralManager, didConnect peripheral: CBPeripheral) {
        peripheral.delegate = self
        state = .discovering
        // Discover all services — the oximeter may not use FFF0.
        peripheral.discoverServices(nil)
    }

    func centralManager(
        _ central: CBCentralManager,
        didDisconnectPeripheral peripheral: CBPeripheral,
        error: Error?
    ) {
        state = .disconnected
        // Auto-reconnect after a short delay.
        DispatchQueue.main.asyncAfter(deadline: .now() + 3) { [weak self] in
            self?.startScanning()
        }
    }

    func centralManager(
        _ central: CBCentralManager,
        didFailToConnect peripheral: CBPeripheral,
        error: Error?
    ) {
        state = .disconnected
        DispatchQueue.main.asyncAfter(deadline: .now() + 3) { [weak self] in
            self?.startScanning()
        }
    }
}

// MARK: - CBPeripheralDelegate

extension OximeterManager: CBPeripheralDelegate {

    func peripheral(_ peripheral: CBPeripheral, didDiscoverServices error: Error?) {
        guard let services = peripheral.services else { return }
        for service in services {
            peripheral.discoverCharacteristics(nil, for: service)
        }
    }

    func peripheral(
        _ peripheral: CBPeripheral,
        didDiscoverCharacteristicsFor service: CBService,
        error: Error?
    ) {
        guard let characteristics = service.characteristics else { return }
        for char in characteristics {
            if char.properties.contains(.notify) || char.properties.contains(.indicate) {
                peripheral.setNotifyValue(true, for: char)
            }
        }
        state = .connected
    }

    func peripheral(
        _ peripheral: CBPeripheral,
        didUpdateValueFor characteristic: CBCharacteristic,
        error: Error?
    ) {
        guard let data = characteristic.value else { return }
        let packet = PacketParser.parse(data)

        switch packet.type {
        case .waveform:
            waveformBuffer.append(Double(packet.amplitude))
            if waveformBuffer.count > Self.maxWaveformPoints {
                waveformBuffer.removeFirst()
            }

        case .reading:
            currentSpO2 = packet.spo2
            currentPulse = packet.pulse
            spo2History.append(Double(packet.spo2))
            pulseHistory.append(Double(packet.pulse))
            if spo2History.count > Self.maxTrendPoints {
                spo2History.removeFirst()
            }
            if pulseHistory.count > Self.maxTrendPoints {
                pulseHistory.removeFirst()
            }
            state = .connected

        case .calibrating:
            state = .calibrating

        case .unknown:
            break
        }
    }
}
