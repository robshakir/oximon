import Foundation
import Testing

@testable import Oximon

/// Port of the Go TestParsePacket table-driven tests — validates the same
/// byte patterns and expected outputs to ensure parity with the daemon.
struct PacketParserTests {

    @Test("Valid waveform packet")
    func validWaveform() {
        let data = Data([0x01, 150])
        let packet = PacketParser.parse(data)
        #expect(packet.type == .waveform)
        #expect(packet.amplitude == 150)
        #expect(packet.spo2 == 0)
        #expect(packet.pulse == 0)
    }

    @Test("Valid reading packet")
    func validReading() {
        let data = Data([0x3E, 98, 0x00, 75, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00])
        let packet = PacketParser.parse(data)
        #expect(packet.type == .reading)
        #expect(packet.spo2 == 98)
        #expect(packet.pulse == 75)
    }

    @Test("Calibrating reading with zero values")
    func calibratingReading() {
        let data = Data([0x3E, 0, 0x00, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00])
        let packet = PacketParser.parse(data)
        #expect(packet.type == .calibrating)
    }

    @Test("Unknown — wrong length for waveform")
    func unknownLength() {
        let data = Data([0x01, 150, 0x00])
        let packet = PacketParser.parse(data)
        #expect(packet.type == .unknown)
    }

    @Test("Unknown — wrong header byte")
    func unknownHeader() {
        let data = Data([0xFF, 98, 0x00, 75, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00])
        let packet = PacketParser.parse(data)
        #expect(packet.type == .unknown)
    }

    @Test("Unknown — empty data")
    func emptyPacket() {
        let data = Data()
        let packet = PacketParser.parse(data)
        #expect(packet.type == .unknown)
    }

    @Test("Waveform amplitude boundary — zero")
    func waveformZeroAmplitude() {
        let data = Data([0x01, 0x00])
        let packet = PacketParser.parse(data)
        #expect(packet.type == .waveform)
        #expect(packet.amplitude == 0)
    }

    @Test("Waveform amplitude boundary — max (255)")
    func waveformMaxAmplitude() {
        let data = Data([0x01, 0xFF])
        let packet = PacketParser.parse(data)
        #expect(packet.type == .waveform)
        #expect(packet.amplitude == 255)
    }

    @Test("Reading with real device data — SpO₂ 95%, Pulse 61 bpm")
    func realDeviceReading() {
        // Actual packet captured from iP900BPB: 3E 5F 00 3D 00 0E 20 00 00 00 00 29 F0
        let data = Data([0x3E, 0x5F, 0x00, 0x3D, 0x00, 0x0E, 0x20, 0x00, 0x00, 0x00, 0x00, 0x29, 0xF0])
        let packet = PacketParser.parse(data)
        #expect(packet.type == .reading)
        #expect(packet.spo2 == 95)  // 0x5F = 95
        #expect(packet.pulse == 61) // 0x3D = 61
    }

    @Test("Calibrating — SpO₂ zero, Pulse nonzero is still calibrating")
    func calibratingSpo2Zero() {
        let data = Data([0x3E, 0, 0x00, 75, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00])
        let packet = PacketParser.parse(data)
        #expect(packet.type == .calibrating)
    }

    @Test("Calibrating — SpO₂ nonzero, Pulse zero is still calibrating")
    func calibratingPulseZero() {
        let data = Data([0x3E, 98, 0x00, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00])
        let packet = PacketParser.parse(data)
        #expect(packet.type == .calibrating)
    }

    @Test("Single byte is unknown")
    func singleByte() {
        let data = Data([0x01])
        let packet = PacketParser.parse(data)
        #expect(packet.type == .unknown)
    }
}
