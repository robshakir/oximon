package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn       *sql.DB
	readingCh  chan reading
	waveformCh chan waveform
}

type reading struct {
	spo2  int
	pulse int
}

type waveform struct {
	amplitude int
}

func InitDB(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Optimize for fast insertions
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
		return nil, fmt.Errorf("failed to initialize db schema: %v", err)
	}

	instance := &DB{
		conn:       db,
		readingCh:  make(chan reading, 1000),
		waveformCh: make(chan waveform, 5000),
	}
	
	go instance.worker()

	return instance, nil
}

func (db *DB) worker() {
	// Simple asynchronous worker to avoid blocking BLE callback
	for {
		select {
		case r := <-db.readingCh:
			db.conn.Exec("INSERT INTO readings (timestamp, spo2, pulse) VALUES (?, ?, ?)", time.Now().UTC(), r.spo2, r.pulse)
		case w := <-db.waveformCh:
			db.conn.Exec("INSERT INTO waveforms (timestamp, amplitude) VALUES (?, ?)", time.Now().UTC(), w.amplitude)
		}
	}
}

func (db *DB) InsertReading(spo2, pulse int) error {
	select {
	case db.readingCh <- reading{spo2, pulse}:
	default:
	}
	return nil
}

func (db *DB) InsertWaveform(amplitude int) error {
	select {
	case db.waveformCh <- waveform{amplitude}:
	default:
	}
	return nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}
