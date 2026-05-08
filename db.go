package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"k8s.io/klog/v2"
	_ "modernc.org/sqlite"
)

// DB manages the asynchronous SQLite connection and insertion queues.
type DB struct {
	// conn holds the active database connection.
	conn *sql.DB
	// readingCh buffers incoming SpO2 and pulse metrics.
	readingCh chan reading
	// waveformCh buffers incoming high-frequency waveform amplitudes.
	waveformCh chan waveform
}

// reading represents an internal payload for SpO2 and pulse data.
type reading struct {
	// spo2 is the blood oxygen percentage.
	spo2 int
	// pulse is the heart rate in beats per minute.
	pulse int
}

// waveform represents an internal payload for a single plethysmograph point.
type waveform struct {
	// amplitude is the raw sensor reading (0-255).
	amplitude int
}

// InitDB initialises the SQLite database at the specified path, configures
// its schema, and returns a pointer to the DB struct.
func InitDB(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	_, err = db.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		
		CREATE TABLE IF NOT EXISTS readings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME,
			spo2 INTEGER,
			pulse INTEGER
		);
		CREATE TABLE IF NOT EXISTS waveforms (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME,
			amplitude INTEGER
		);
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to initialise db schema: %w", err)
	}

	return &DB{
		conn:       db,
		readingCh:  make(chan reading, 1000),
		waveformCh: make(chan waveform, 5000),
	}, nil
}

// StartWorker spins up a background goroutine that pulls payloads from the
// internal channels and inserts them into SQLite. It exits when the context is canceled.
func (db *DB) StartWorker(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case r := <-db.readingCh:
				_, err := db.conn.Exec("INSERT INTO readings (timestamp, spo2, pulse) VALUES (?, ?, ?)", time.Now().UTC(), r.spo2, r.pulse)
				if err != nil {
					klog.Errorf("failed to insert reading: %v", err)
				}
			case w := <-db.waveformCh:
				_, err := db.conn.Exec("INSERT INTO waveforms (timestamp, amplitude) VALUES (?, ?)", time.Now().UTC(), w.amplitude)
				if err != nil {
					klog.Errorf("failed to insert waveform: %v", err)
				}
			}
		}
	}()
}

// InsertReading places a new SpO2 and pulse reading into the asynchronous
// database insertion queue. It drops the reading if the queue is full.
func (db *DB) InsertReading(spo2, pulse int) error {
	select {
	case db.readingCh <- reading{spo2: spo2, pulse: pulse}:
	default:
		klog.Warningf("reading channel full, dropping data point")
	}
	return nil
}

// InsertWaveform places a new waveform amplitude into the asynchronous
// database insertion queue. It drops the amplitude if the queue is full.
func (db *DB) InsertWaveform(amplitude int) error {
	select {
	case db.waveformCh <- waveform{amplitude: amplitude}:
	default:
		// We drop silently for waveforms to avoid flooding the logs
	}
	return nil
}

// Close gracefully shuts down the SQLite connection.
func (db *DB) Close() error {
	return db.conn.Close()
}
