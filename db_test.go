package main

import (
	"testing"
	"time"
)

func TestDBWorker(t *testing.T) {
	// Initialize an in-memory database
	db, err := InitDB("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Failed to init in-memory DB: %v", err)
	}
	defer db.Close()

	// Insert data asynchronously
	err = db.InsertReading(98, 75)
	if err != nil {
		t.Fatalf("InsertReading failed: %v", err)
	}
	err = db.InsertWaveform(150)
	if err != nil {
		t.Fatalf("InsertWaveform failed: %v", err)
	}

	// Wait briefly for the worker to process channels
	time.Sleep(100 * time.Millisecond)

	// Verify Reading
	var spo2, pulse int
	err = db.conn.QueryRow("SELECT spo2, pulse FROM readings ORDER BY id DESC LIMIT 1").Scan(&spo2, &pulse)
	if err != nil {
		t.Fatalf("Failed to query reading: %v", err)
	}
	if spo2 != 98 || pulse != 75 {
		t.Errorf("Expected reading 98/75, got %v/%v", spo2, pulse)
	}

	// Verify Waveform
	var amp int
	err = db.conn.QueryRow("SELECT amplitude FROM waveforms ORDER BY id DESC LIMIT 1").Scan(&amp)
	if err != nil {
		t.Fatalf("Failed to query waveform: %v", err)
	}
	if amp != 150 {
		t.Errorf("Expected amplitude 150, got %v", amp)
	}
}
