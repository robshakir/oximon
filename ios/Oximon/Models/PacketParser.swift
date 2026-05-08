import Foundation

/// Classification of raw Bluetooth packets from the BerryMed/Innovo oximeter.
enum PacketType {
    /// The packet could not be decoded.
    case unknown
    /// A 50 Hz plethysmograph amplitude sample.
    case waveform
    /// A calculated SpO₂ + pulse reading.
    case reading
    /// The sensor is active but hasn't locked onto a pulse.
    case calibrating
}

/// Decoded telemetry values from a single raw BLE payload.
struct ParsedPacket {
    let type: PacketType
    let amplitude: Int
    let spo2: Int
    let pulse: Int
}

/// Stateless parser for raw oximeter BLE characteristic data.
///
/// Packet formats (BerryMed chipset, characteristic 0xFFF1):
///   - Waveform: 2 bytes  — [0x01, amplitude]
///   - Reading:  13 bytes — [0x3E, spo2, _, pulse, …]
enum PacketParser {

    /// Inspect a raw byte buffer and decode it into a ``ParsedPacket``.
    static func parse(_ data: Data) -> ParsedPacket {
        // Waveform packet: 2 bytes, first byte is 0x01.
        if data.count == 2 && data[0] == 0x01 {
            return ParsedPacket(
                type: .waveform,
                amplitude: Int(data[1]),
                spo2: 0,
                pulse: 0
            )
        }

        // Reading packet: 13 bytes, first byte is 0x3E.
        if data.count == 13 && data[0] == 0x3E {
            let spo2 = Int(data[1])
            let pulse = Int(data[3])
            if spo2 > 0 && pulse > 0 {
                return ParsedPacket(
                    type: .reading,
                    amplitude: 0,
                    spo2: spo2,
                    pulse: pulse
                )
            }
            return ParsedPacket(type: .calibrating, amplitude: 0, spo2: 0, pulse: 0)
        }

        return ParsedPacket(type: .unknown, amplitude: 0, spo2: 0, pulse: 0)
    }
}
