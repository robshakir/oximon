package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"tinygo.org/x/bluetooth"
)

var adapter = bluetooth.DefaultAdapter

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	err := adapter.Enable()
	if err != nil {
		log.Fatalf("failed to enable adapter: %v", err)
	}

	switch os.Args[1] {
	case "scan":
		scanCmd := flag.NewFlagSet("scan", flag.ExitOnError)
		scanCmd.Parse(os.Args[2:])
		runScan()
	case "run":
		runCmd := flag.NewFlagSet("run", flag.ExitOnError)
		macFlag := runCmd.String("mac", "", "MAC address of the device")
		nameFlag := runCmd.String("name", "", "LocalName of the device")
		runCmd.Parse(os.Args[2:])

		if *macFlag == "" && *nameFlag == "" {
			fmt.Println("Please provide either -mac or -name flag")
			os.Exit(1)
		}
		startForeground(*macFlag, *nameFlag)
	case "daemon":
		daemonCmd := flag.NewFlagSet("daemon", flag.ExitOnError)
		macFlag := daemonCmd.String("mac", "", "MAC address of the device")
		nameFlag := daemonCmd.String("name", "", "LocalName of the device")
		daemonCmd.Parse(os.Args[2:])

		if *macFlag == "" && *nameFlag == "" {
			fmt.Println("Please provide either -mac or -name flag")
			os.Exit(1)
		}
		runDaemon(*macFlag, *nameFlag)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: oximon <command> [options]")
	fmt.Println("Commands:")
	fmt.Println("  scan              Scan for BLE devices")
	fmt.Println("  run -name <name>  Run the data logger in foreground")
	fmt.Println("  daemon -name <n>  Run the data logger in background")
}

func runScan() {
	fmt.Println("Scanning for BLE devices... (Press Ctrl+C to stop)")
	err := adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
		name := device.LocalName()
		if name == "" {
			name = "(unknown)"
		}
		fmt.Printf("Found device: %s [%s] RSSI: %d\n", name, device.Address.String(), device.RSSI)
	})
	if err != nil {
		log.Fatalf("failed to scan: %v", err)
	}
}
