package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"k8s.io/klog/v2"
	"tinygo.org/x/bluetooth"
)

// PacketType represents the classification of a raw Bluetooth packet.
type PacketType int

const (
	// PacketUnknown is used when a packet cannot be decoded.
	PacketUnknown PacketType = iota
	// PacketWaveform represents a high-frequency plethysmograph amplitude sample.
	PacketWaveform
	// PacketReading represents a low-frequency calculated SpO2 and Pulse rate packet.
	PacketReading
	// PacketCalibrating represents a packet sent when the sensor is active but hasn't locked onto a pulse.
	PacketCalibrating
)

// String returns the string representation of the PacketType enum.
func (p PacketType) String() string {
	switch p {
	case PacketWaveform:
		return "Waveform"
	case PacketReading:
		return "Reading"
	case PacketCalibrating:
		return "Calibrating"
	default:
		return "Unknown"
	}
}

// ParsedPacket holds the extracted telemetry values from a raw Bluetooth payload.
type ParsedPacket struct {
	// Type defines the classification of this packet.
	Type PacketType
	// Amplitude is the 50Hz raw plethysmograph value (0-255).
	Amplitude int
	// SpO2 is the calculated blood oxygen saturation percentage (0-100).
	SpO2 int
	// Pulse is the calculated heart rate in beats per minute.
	Pulse int
}

// ParsePacket inspects a raw byte slice from the oximeter and decodes it into a ParsedPacket.
func ParsePacket(buf []byte) ParsedPacket {
	if len(buf) == 2 && buf[0] == 0x01 {
		return ParsedPacket{Type: PacketWaveform, Amplitude: int(buf[1])}
	}

	if len(buf) == 13 && buf[0] == 0x3e {
		spo2 := int(buf[1])
		pulse := int(buf[3])
		if spo2 > 0 && pulse > 0 {
			return ParsedPacket{Type: PacketReading, SpO2: spo2, Pulse: pulse}
		}
		return ParsedPacket{Type: PacketCalibrating}
	}

	return ParsedPacket{Type: PacketUnknown}
}

// startForeground initialises the daemon subsystems and blocks, streaming
// data from the Bluetooth device until interrupted.
func startForeground(ctx context.Context, targetMAC, targetName string) {
	db, err := InitDB("oximon.db")
	if err != nil {
		klog.Fatalf("failed to initialise db: %v", err)
	}
	defer db.Close()
	db.StartWorker(ctx)

	gnmiSrv := NewGNMIServer()
	if err := StartGRPCServer(9339, gnmiSrv); err != nil {
		klog.Fatalf("failed to start gnmi server: %v", err)
	}

	StartWebServer(8080, gnmiSrv)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		klog.Infof("Interrupt received. Exiting...")
		db.Close()
		os.Exit(0)
	}()

	klog.Infof("gNMI Streaming active on port 9339")
	klog.Infof("Connecting to device...")

	for {
		if ctx.Err() != nil {
			return
		}

		err := connectAndListen(ctx, targetMAC, targetName, db, gnmiSrv)
		if err != nil {
			klog.Errorf("connection lost or failed: %v. retrying in 5 seconds...", err)

			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return
			}
		}
	}
}

// connectAndListen scans for the target device, establishes a BLE connection,
// subscribes to all characteristics, and routes incoming notifications to the DB and gNMI server.
func connectAndListen(ctx context.Context, targetMAC, targetName string, db *DB, gnmiSrv *GNMIServer) error {
	var targetDevice bluetooth.ScanResult
	var found bool

	err := bleAdapter.Scan(func(a *bluetooth.Adapter, device bluetooth.ScanResult) {
		if found {
			return
		}
		match := false
		if targetMAC != "" && strings.EqualFold(device.Address.String(), targetMAC) {
			match = true
		}
		if targetName != "" && strings.EqualFold(device.LocalName(), targetName) {
			match = true
		}

		if match {
			klog.Infof("Found target device: %s [%s]", device.LocalName(), device.Address.String())
			targetDevice = device
			found = true
			a.StopScan()
		}
	})

	if err != nil {
		return fmt.Errorf("scan error: %w", err)
	}

	if !found {
		return fmt.Errorf("could not find target device within timeout")
	}

	klog.Infof("Connecting...")
	dev, err := bleAdapter.Connect(targetDevice.Address, bluetooth.ConnectionParams{})
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer dev.Disconnect()
	klog.Infof("Connected. Discovering services...")

	services, err := dev.DiscoverServices(nil)
	if err != nil {
		return fmt.Errorf("failed to discover services: %w", err)
	}

	for _, srv := range services {
		chars, err := srv.DiscoverCharacteristics(nil)
		if err != nil {
			continue
		}
		for _, char := range chars {
			c := char
			uuidStr := c.UUID().String()

			time.Sleep(50 * time.Millisecond)

			err := c.EnableNotifications(func(buf []byte) {
				if uuidStr != "0000fff1-0000-1000-8000-00805f9b34fb" {
					return
				}

				now := time.Now().UTC()
				packet := ParsePacket(buf)

				switch packet.Type {
				case PacketWaveform:
					db.InsertWaveform(packet.Amplitude)
					gnmiSrv.Broadcast(TelemetryUpdate{Timestamp: now, Type: "waveform", Value: packet.Amplitude})
				case PacketReading:
					db.InsertReading(packet.SpO2, packet.Pulse)
					gnmiSrv.Broadcast(TelemetryUpdate{Timestamp: now, Type: "spo2", Value: packet.SpO2})
					gnmiSrv.Broadcast(TelemetryUpdate{Timestamp: now, Type: "pulse", Value: packet.Pulse})
					klog.Infof("Pulse: %3d bpm | SpO2: %3d%%", packet.Pulse, packet.SpO2)
				case PacketCalibrating:
					klog.Infof("Calibrating... (waiting for pulse lock)")
				case PacketUnknown:
					// Ignored
				}
			})
			if err == nil {
				klog.Infof("Successfully subscribed to %s", uuidStr)
			}
		}
	}

	klog.Infof("Notifications enabled. Waiting for data...")

	<-ctx.Done()
	return ctx.Err()
}
