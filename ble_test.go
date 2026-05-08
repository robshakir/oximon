package main

import (
	"testing"
)

func TestParsePacket(t *testing.T) {
	tests := []struct {
		name     string
		payload  []byte
		expected ParsedPacket
	}{
		{
			name:     "Valid Waveform",
			payload:  []byte{0x01, 150},
			expected: ParsedPacket{Type: PacketWaveform, Amplitude: 150},
		},
		{
			name:     "Valid Reading",
			payload:  []byte{0x3e, 98, 0x00, 75, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: ParsedPacket{Type: PacketReading, SpO2: 98, Pulse: 75},
		},
		{
			name:     "Calibrating Reading (Zeros)",
			payload:  []byte{0x3e, 0, 0x00, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: ParsedPacket{Type: PacketCalibrating},
		},
		{
			name:     "Unknown Length",
			payload:  []byte{0x01, 150, 0x00},
			expected: ParsedPacket{Type: PacketUnknown},
		},
		{
			name:     "Unknown Header",
			payload:  []byte{0xff, 98, 0x00, 75, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: ParsedPacket{Type: PacketUnknown},
		},
		{
			name:     "Empty Packet",
			payload:  []byte{},
			expected: ParsedPacket{Type: PacketUnknown},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParsePacket(tt.payload)
			if result.Type != tt.expected.Type {
				t.Errorf("Expected type %v, got %v", tt.expected.Type, result.Type)
			}
			if result.Amplitude != tt.expected.Amplitude {
				t.Errorf("Expected amplitude %v, got %v", tt.expected.Amplitude, result.Amplitude)
			}
			if result.SpO2 != tt.expected.SpO2 {
				t.Errorf("Expected SpO2 %v, got %v", tt.expected.SpO2, result.SpO2)
			}
			if result.Pulse != tt.expected.Pulse {
				t.Errorf("Expected Pulse %v, got %v", tt.expected.Pulse, result.Pulse)
			}
		})
	}
}
