package main

import (
	"testing"
	"time"
)

func TestGNMIBroadcaster(t *testing.T) {
	srv := NewGNMIServer()

	// Create a subscriber
	ch := srv.SubscribeChan()

	// Broadcast an update
	update := TelemetryUpdate{
		Timestamp: time.Now().UTC(),
		Type:      "spo2",
		Value:     99,
	}
	srv.Broadcast(update)

	// Verify the subscriber received it
	select {
	case received := <-ch:
		if received.Type != "spo2" || received.Value != 99 {
			t.Errorf("Expected spo2=99, got %v=%v", received.Type, received.Value)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for broadcast")
	}

	// Verify Unsubscribe
	srv.UnsubscribeChan(ch)
	
	// Channel should be closed, reading from it should yield zero value and false
	_, ok := <-ch
	if ok {
		t.Error("Expected channel to be closed after UnsubscribeChan")
	}
}
