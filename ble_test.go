package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestParsePacket validates the extraction logic for raw BLE payloads.
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
			got := ParsePacket(tt.payload)
			if diff := cmp.Diff(tt.expected, got); diff != "" {
				t.Errorf("ParsePacket() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
