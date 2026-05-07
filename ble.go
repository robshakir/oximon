package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"tinygo.org/x/bluetooth"
)

func startForeground(targetMAC, targetName string) {
	db, err := InitDB("oximon.db")
	if err != nil {
		fmt.Printf("Failed to initialize DB: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	gnmiSrv := NewGNMIServer()
	err = StartGRPCServer(9339, gnmiSrv)
	if err != nil {
		fmt.Printf("Failed to start gNMI server: %v\n", err)
		os.Exit(1)
	}
	
	StartWebServer(8080, gnmiSrv)
	
	fmt.Println("📡 gNMI Streaming on port 9339")
	fmt.Println("Connecting to device...")

	// Connect loop to handle disconnects
	for {
		err := connectAndListen(targetMAC, targetName, db, gnmiSrv)
		if err != nil {
			fmt.Printf("\nConnection lost or failed: %v. Retrying in 5 seconds...\n", err)
			time.Sleep(5 * time.Second)
		}
	}
}

func connectAndListen(targetMAC, targetName string, db *DB, gnmiSrv *GNMIServer) error {
	var targetDevice bluetooth.ScanResult
	var found bool

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := adapter.Scan(func(a *bluetooth.Adapter, device bluetooth.ScanResult) {
		if found {
			return
		}
		match := false
		if targetMAC != "" && strings.EqualFold(device.Address.String(), targetMAC) {
			match = true
		} else if targetName != "" && strings.EqualFold(device.LocalName(), targetName) {
			match = true
		}

		if match {
			fmt.Printf("Found target device: %s [%s]\n", device.LocalName(), device.Address.String())
			targetDevice = device
			found = true
			a.StopScan()
		}
	})

	if err != nil {
		return fmt.Errorf("scan error: %v", err)
	}

	if !found {
		return fmt.Errorf("could not find target device within timeout")
	}

	fmt.Println("Connecting...")
	dev, err := adapter.Connect(targetDevice.Address, bluetooth.ConnectionParams{})
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	defer dev.Disconnect()
	fmt.Println("Connected. Discovering services...")

	services, err := dev.DiscoverServices(nil)
	if err != nil {
		return fmt.Errorf("failed to discover services: %v", err)
	}

	for _, srv := range services {
		chars, err := srv.DiscoverCharacteristics(nil)
		if err != nil {
			continue
		}
		for _, char := range chars {
			// Copy for closure
			c := char
			uuidStr := c.UUID().String()
			
			// tiny delay to prevent overwhelming the macos ble stack
			time.Sleep(50 * time.Millisecond)
			
			err := c.EnableNotifications(func(buf []byte) {
				if uuidStr != "0000fff1-0000-1000-8000-00805f9b34fb" {
					return
				}
				
				now := time.Now().UTC()
				if len(buf) == 2 && buf[0] == 0x01 {
					amp := int(buf[1])
					db.InsertWaveform(amp)
					gnmiSrv.Broadcast(TelemetryUpdate{Timestamp: now, Type: "waveform", Value: amp})
				} else if len(buf) == 13 && buf[0] == 0x3e {
					spo2 := int(buf[1])
					pulse := int(buf[3])
					
					if spo2 > 0 && pulse > 0 {
						db.InsertReading(spo2, pulse)
						gnmiSrv.Broadcast(TelemetryUpdate{Timestamp: now, Type: "spo2", Value: spo2})
						gnmiSrv.Broadcast(TelemetryUpdate{Timestamp: now, Type: "pulse", Value: pulse})
						fmt.Printf("❤️  Pulse: %3d bpm   |   🩸 SpO2: %3d%%\n", pulse, spo2)
					} else {
						fmt.Printf("⏳ Calibrating... (waiting for pulse lock)\n")
					}
				}
			})
			if err == nil {
				fmt.Printf("Successfully subscribed to %s\n", uuidStr)
			}
		}
	}

	fmt.Println("Notifications enabled. Waiting for data...")

	// Wait for disconnection or interruption
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	select {
	case <-ctx.Done():
		// Context was only for scan timeout, we don't exit on it after connection
	case <-c:
		fmt.Println("\nExiting.")
		os.Exit(0)
	}
	// Note: We don't have a great way to detect disconnects from tinygo.org/x/bluetooth right now
	// without reading, but let's assume it exits or errors eventually, or we just block forever.
	// Actually, if the device disconnects, the library usually doesn't cleanly notify us in a cross-platform way unless we poll.
	// For now, we block.
	select {}
}
