package main

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

// TestDBWorker verifies that the asynchronous DB insertion worker correctly
// saves incoming readings to the SQLite database without blocking.
func TestDBWorker(t *testing.T) {
	db, err := InitDB("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db.StartWorker(ctx)

	if err := db.InsertReading(98, 75); err != nil {
		t.Fatalf("InsertReading failed: %v", err)
	}
	if err := db.InsertWaveform(150); err != nil {
		t.Fatalf("InsertWaveform failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	var spo2, pulse int
	if err := db.conn.QueryRow("SELECT spo2, pulse FROM readings ORDER BY id DESC LIMIT 1").Scan(&spo2, &pulse); err != nil {
		t.Fatalf("QueryRow(readings) failed: %v", err)
	}

	wantReading := reading{spo2: 98, pulse: 75}
	gotReading := reading{spo2: spo2, pulse: pulse}
	if diff := cmp.Diff(wantReading, gotReading, cmp.AllowUnexported(reading{})); diff != "" {
		t.Errorf("readings mismatch (-want +got):\n%s", diff)
	}

	var amp int
	if err := db.conn.QueryRow("SELECT amplitude FROM waveforms ORDER BY id DESC LIMIT 1").Scan(&amp); err != nil {
		t.Fatalf("QueryRow(waveforms) failed: %v", err)
	}

	wantWaveform := waveform{amplitude: 150}
	gotWaveform := waveform{amplitude: amp}
	if diff := cmp.Diff(wantWaveform, gotWaveform, cmp.AllowUnexported(waveform{})); diff != "" {
		t.Errorf("waveforms mismatch (-want +got):\n%s", diff)
	}
}
