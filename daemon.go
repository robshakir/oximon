package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"k8s.io/klog/v2"
)

// runDaemon launches the oximon executable as a background process and detaches it.
func runDaemon(ctx context.Context, targetMAC, targetName string) {
	fmt.Println("Starting oximon in background...")

	args := []string{"run"}
	if targetName != "" {
		args = append(args, "-name", targetName)
	}
	if targetMAC != "" {
		args = append(args, "-mac", targetMAC)
	}

	cmd := exec.Command(os.Args[0], args...)

	logFile, err := os.OpenFile("oximon.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		klog.Fatalf("failed to open log file: %v", err)
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		klog.Fatalf("failed to start daemon process: %v", err)
	}

	fmt.Printf("Daemon started with PID: %d. Logs are in oximon.log\n", cmd.Process.Pid)
}
