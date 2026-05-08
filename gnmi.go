package main

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

// TelemetryUpdate represents a single reading from the oximeter device,
// mapping a value to a specific measurement type at a given timestamp.
type TelemetryUpdate struct {
	// Timestamp is the UTC time the reading was processed.
	Timestamp time.Time `json:"timestamp"`
	// Type indicates the metric name (e.g., "spo2", "pulse", or "waveform").
	Type string `json:"type"`
	// Value is the measured integer data point.
	Value int `json:"value"`
}

// GNMIServer implements the standard OpenConfig gNMI service, providing
// a publish-subscribe mechanism for streaming telemetry data to clients.
type GNMIServer struct {
	gnmi.UnimplementedGNMIServer

	// mu protects the subscribers map.
	mu sync.RWMutex
	// subscribers holds all active channels listening for updates.
	subscribers map[chan TelemetryUpdate]struct{}
}

// NewGNMIServer initializes and returns a new GNMIServer instance.
func NewGNMIServer() *GNMIServer {
	return &GNMIServer{
		subscribers: make(map[chan TelemetryUpdate]struct{}),
	}
}

// UnsubscribeChan gracefully closes a subscriber channel and removes it from the server.
func (s *GNMIServer) UnsubscribeChan(ch chan TelemetryUpdate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.subscribers, ch)
	close(ch)
}

// SubscribeChan registers a new channel to receive broadcasted telemetry updates.
func (s *GNMIServer) SubscribeChan() chan TelemetryUpdate {
	ch := make(chan TelemetryUpdate, 100)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscribers[ch] = struct{}{}
	return ch
}

// Broadcast sends the provided telemetry update to all active gNMI subscriptions.
// It drops the update for any subscriber whose channel is full to prevent deadlocking.
func (s *GNMIServer) Broadcast(update TelemetryUpdate) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for ch := range s.subscribers {
		select {
		case ch <- update:
		default:
		}
	}
}

// Subscribe implements the gNMI Subscribe RPC, streaming telemetry updates back to the client.
func (s *GNMIServer) Subscribe(stream gnmi.GNMI_SubscribeServer) error {
	req, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to read subscribe request: %w", err)
	}

	if req.GetSubscribe() == nil {
		return status.Error(codes.InvalidArgument, "request must be a subscription")
	}

	ch := make(chan TelemetryUpdate, 100)
	s.mu.Lock()
	s.subscribers[ch] = struct{}{}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.subscribers, ch)
		s.mu.Unlock()
		close(ch)
	}()

	err = stream.Send(&gnmi.SubscribeResponse{
		Response: &gnmi.SubscribeResponse_SyncResponse{
			SyncResponse: true,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to send sync response: %w", err)
	}

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case update := <-ch:
			val := &gnmi.TypedValue{
				Value: &gnmi.TypedValue_IntVal{IntVal: int64(update.Value)},
			}
			path := &gnmi.Path{
				Elem: []*gnmi.PathElem{
					{Name: "oximeter"},
					{Name: "state"},
					{Name: update.Type},
				},
			}

			resp := &gnmi.SubscribeResponse{
				Response: &gnmi.SubscribeResponse_Update{
					Update: &gnmi.Notification{
						Timestamp: update.Timestamp.UnixNano(),
						Update: []*gnmi.Update{
							{
								Path: path,
								Val:  val,
							},
						},
					},
				},
			}

			if err := stream.Send(resp); err != nil {
				return fmt.Errorf("failed to send update: %w", err)
			}
		}
	}
}

// Capabilities implements the gNMI Capabilities RPC, returning supported models and encodings.
func (s *GNMIServer) Capabilities(ctx context.Context, req *gnmi.CapabilityRequest) (*gnmi.CapabilityResponse, error) {
	return &gnmi.CapabilityResponse{
		GNMIVersion:        "0.8.0",
		SupportedModels:    []*gnmi.ModelData{},
		SupportedEncodings: []gnmi.Encoding{gnmi.Encoding_PROTO},
	}, nil
}

// StartGRPCServer initializes a TCP listener on the given port and begins serving gNMI requests.
func StartGRPCServer(port int, srv *GNMIServer) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	g := grpc.NewServer()
	gnmi.RegisterGNMIServer(g, srv)

	go func() {
		if err := g.Serve(lis); err != nil {
			klog.Errorf("grpc server failed: %v", err)
		}
	}()
	return nil
}
