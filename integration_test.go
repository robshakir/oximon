package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestIntegrationPipeline(t *testing.T) {
	// 1. Setup DB
	db, err := InitDB("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("DB Init failed: %v", err)
	}
	defer db.Close()

	// 2. Setup gNMI Server
	gnmiSrv := NewGNMIServer()
	
	lis, err := net.Listen("tcp", "127.0.0.1:0") // Random free port
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	gnmi.RegisterGNMIServer(grpcServer, gnmiSrv)
	go grpcServer.Serve(lis)
	defer grpcServer.Stop()

	// 3. Setup WebSocket Server (using httptest)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsHandler(w, r, gnmiSrv)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// 4. Connect gNMI Client
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Did not connect: %v", err)
	}
	defer conn.Close()
	c := gnmi.NewGNMIClient(conn)
	
	stream, err := c.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	err = stream.Send(&gnmi.SubscribeRequest{
		Request: &gnmi.SubscribeRequest_Subscribe{
			Subscribe: &gnmi.SubscriptionList{},
		},
	})
	if err != nil {
		t.Fatalf("Failed to send subscribe req: %v", err)
	}

	// Consume the initial SyncResponse
	_, err = stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive sync response: %v", err)
	}

	// 5. Connect WebSocket Client
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer ws.Close()

	// 6. INJECT PACKET (Fake BLE Notification)
	payload := []byte{0x3e, 98, 0x00, 75, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	packet := ParsePacket(payload)
	now := time.Now().UTC()

	// Route it like ble.go does
	if packet.Type == PacketReading {
		db.InsertReading(packet.SpO2, packet.Pulse)
		gnmiSrv.Broadcast(TelemetryUpdate{Timestamp: now, Type: "spo2", Value: packet.SpO2})
		gnmiSrv.Broadcast(TelemetryUpdate{Timestamp: now, Type: "pulse", Value: packet.Pulse})
	} else {
		t.Fatalf("Expected PacketReading, got %v", packet.Type)
	}

	// 7. Verify DB
	time.Sleep(100 * time.Millisecond) // Let worker process
	var spo2, pulse int
	err = db.conn.QueryRow("SELECT spo2, pulse FROM readings ORDER BY id DESC LIMIT 1").Scan(&spo2, &pulse)
	if err != nil {
		t.Fatalf("DB query failed: %v", err)
	}
	if spo2 != 98 || pulse != 75 {
		t.Errorf("DB expected 98/75, got %d/%d", spo2, pulse)
	}

	// 8. Verify gNMI Client
	resp, err := stream.Recv() // We expect spo2 first
	if err != nil {
		t.Fatalf("gNMI Recv failed: %v", err)
	}
	update := resp.GetUpdate()
	if update == nil || len(update.Update) == 0 {
		t.Fatalf("gNMI update was empty")
	}
	path := update.Update[0].Path.Elem
	if path[len(path)-1].Name != "spo2" {
		t.Errorf("Expected gNMI path 'spo2', got '%s'", path[len(path)-1].Name)
	}
	val := update.Update[0].Val.GetIntVal()
	if val != 98 {
		t.Errorf("Expected gNMI val 98, got %d", val)
	}

	// 9. Verify WebSocket Client
	var wsMsg TelemetryUpdate
	err = ws.ReadJSON(&wsMsg) // Should get spo2 first
	if err != nil {
		t.Fatalf("WebSocket ReadJSON failed: %v", err)
	}
	if wsMsg.Type != "spo2" || wsMsg.Value != 98 {
		t.Errorf("Expected WS JSON spo2=98, got %s=%d", wsMsg.Type, wsMsg.Value)
	}
}
