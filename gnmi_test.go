package main

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

// TestGNMIBroadcaster verifies the pub/sub channels within the GNMIServer.
func TestGNMIBroadcaster(t *testing.T) {
	srv := NewGNMIServer()

	ch := srv.SubscribeChan()

	update := TelemetryUpdate{
		Timestamp: time.Now().UTC(),
		Type:      "spo2",
		Value:     99,
	}
	srv.Broadcast(update)

	select {
	case received := <-ch:
		if diff := cmp.Diff(update, received); diff != "" {
			t.Errorf("Broadcast() mismatch (-want +got):\n%s", diff)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for broadcast")
	}

	srv.UnsubscribeChan(ch)

	_, ok := <-ch
	if ok {
		t.Error("Expected channel to be closed after UnsubscribeChan")
	}
}
