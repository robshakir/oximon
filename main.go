// Package main provides the oximon daemon for capturing, persisting, and
// streaming real-time telemetry from Innovo Bluetooth oximeter devices.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"k8s.io/klog/v2"
	"tinygo.org/x/bluetooth"
)

// bleAdapter is the global Bluetooth adapter instance used to scan and connect to devices.
var bleAdapter = bluetooth.DefaultAdapter

// main is the entry point for the oximon application.
func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	if err := bleAdapter.Enable(); err != nil {
		klog.Fatalf("failed to enable ble adapter: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	switch os.Args[1] {
	case "scan":
		scanCmd := flag.NewFlagSet("scan", flag.ExitOnError)
		scanCmd.Parse(os.Args[2:])
		runScan(ctx)
	case "run":
		runCmd := flag.NewFlagSet("run", flag.ExitOnError)
		macFlag := runCmd.String("mac", "", "MAC address of the device")
		nameFlag := runCmd.String("name", "", "LocalName of the device")
		runCmd.Parse(os.Args[2:])

		if *macFlag == "" && *nameFlag == "" {
			klog.Fatalf("must provide either -mac or -name flag")
		}
		startForeground(ctx, *macFlag, *nameFlag)
	case "daemon":
		daemonCmd := flag.NewFlagSet("daemon", flag.ExitOnError)
		macFlag := daemonCmd.String("mac", "", "MAC address of the device")
		nameFlag := daemonCmd.String("name", "", "LocalName of the device")
		daemonCmd.Parse(os.Args[2:])

		if *macFlag == "" && *nameFlag == "" {
			klog.Fatalf("must provide either -mac or -name flag")
		}
		runDaemon(ctx, *macFlag, *nameFlag)
	default:
		printUsage()
		os.Exit(1)
	}
}

// printUsage outputs the available CLI commands and flags to standard output.
func printUsage() {
	fmt.Println("Usage: oximon <command> [options]")
	fmt.Println("Commands:")
	fmt.Println("  scan              Scan for BLE devices")
	fmt.Println("  run -name <name>  Run the data logger in foreground")
	fmt.Println("  daemon -name <n>  Run the data logger in background")
}

// runScan begins scanning for BLE devices and logs their names and MAC addresses
// until the provided context is canceled.
func runScan(ctx context.Context) {
	klog.Infof("Scanning for BLE devices... (Press Ctrl+C to stop)")
	err := bleAdapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
		name := device.LocalName()
		if name == "" {
			name = "(unknown)"
		}
		klog.Infof("Found device: %s [%s] RSSI: %d", name, device.Address.String(), device.RSSI)
	})
	if err != nil {
		klog.Fatalf("failed to scan for ble devices: %v", err)
	}
}
