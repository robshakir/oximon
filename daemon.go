package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func runDaemon(targetMAC, targetName string) {
	fmt.Println("Starting oximon in background...")

	args := []string{"run"}
	if targetName != "" {
		args = append(args, "-name", targetName)
	}
	if targetMAC != "" {
		args = append(args, "-mac", targetMAC)
	}

	cmd := exec.Command(os.Args[0], args...)
	
	// Create log file for background process
	logFile, err := os.OpenFile("oximon.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}
	
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	err = cmd.Start()
	if err != nil {
		log.Fatalf("failed to start daemon: %v", err)
	}

	fmt.Printf("Daemon started with PID: %d. Logs are in oximon.log\n", cmd.Process.Pid)
}
