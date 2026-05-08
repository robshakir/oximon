package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/gorilla/websocket"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestIntegrationPipeline verifies the entire end-to-end flow from packet decoding
// to database storage, gNMI stream, and WebSocket delivery.
func TestIntegrationPipeline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := InitDB("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()
	db.StartWorker(ctx)

	gnmiSrv := NewGNMIServer()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	gnmi.RegisterGNMIServer(grpcServer, gnmiSrv)
	go grpcServer.Serve(lis)
	defer grpcServer.Stop()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsHandler(w, r, gnmiSrv)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	gnmiCtx, gnmiCancel := context.WithTimeout(ctx, 5*time.Second)
	defer gnmiCancel()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc dial failed: %v", err)
	}
	defer conn.Close()

	c := gnmi.NewGNMIClient(conn)
	stream, err := c.Subscribe(gnmiCtx)
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	if err := stream.Send(&gnmi.SubscribeRequest{
		Request: &gnmi.SubscribeRequest_Subscribe{
			Subscribe: &gnmi.SubscriptionList{},
		},
	}); err != nil {
		t.Fatalf("failed to send subscribe req: %v", err)
	}

	if _, err := stream.Recv(); err != nil {
		t.Fatalf("failed to receive sync response: %v", err)
	}

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v", err)
	}
	defer ws.Close()

	payload := []byte{0x3e, 98, 0x00, 75, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	packet := ParsePacket(payload)
	now := time.Now().UTC()

	if packet.Type != PacketReading {
		t.Fatalf("expected PacketReading, got %v", packet.Type)
	}

	if err := db.InsertReading(packet.SpO2, packet.Pulse); err != nil {
		t.Fatalf("InsertReading failed: %v", err)
	}
	gnmiSrv.Broadcast(TelemetryUpdate{Timestamp: now, Type: "spo2", Value: packet.SpO2})
	gnmiSrv.Broadcast(TelemetryUpdate{Timestamp: now, Type: "pulse", Value: packet.Pulse})

	time.Sleep(100 * time.Millisecond)
	var spo2, pulse int
	if err := db.conn.QueryRow("SELECT spo2, pulse FROM readings ORDER BY id DESC LIMIT 1").Scan(&spo2, &pulse); err != nil {
		t.Fatalf("db query failed: %v", err)
	}

	if diff := cmp.Diff(reading{spo2: 98, pulse: 75}, reading{spo2: spo2, pulse: pulse}, cmp.AllowUnexported(reading{})); diff != "" {
		t.Errorf("db values mismatch (-want +got):\n%s", diff)
	}

	resp, err := stream.Recv()
	if err != nil {
		t.Fatalf("gNMI recv failed: %v", err)
	}
	update := resp.GetUpdate()
	if update == nil || len(update.Update) == 0 {
		t.Fatalf("gNMI update was empty")
	}
	pathName := update.Update[0].Path.Elem[len(update.Update[0].Path.Elem)-1].Name
	val := update.Update[0].Val.GetIntVal()

	if diff := cmp.Diff("spo2", pathName); diff != "" {
		t.Errorf("gNMI path mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(int64(98), val); diff != "" {
		t.Errorf("gNMI val mismatch (-want +got):\n%s", diff)
	}

	var wsMsg TelemetryUpdate
	if err := ws.ReadJSON(&wsMsg); err != nil {
		t.Fatalf("websocket ReadJSON failed: %v", err)
	}

	wantWs := TelemetryUpdate{Timestamp: now, Type: "spo2", Value: 98}
	if diff := cmp.Diff(wantWs, wsMsg, cmpopts.IgnoreFields(TelemetryUpdate{}, "Timestamp")); diff != "" {
		t.Errorf("websocket msg mismatch (-want +got):\n%s", diff)
	}
}
