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
)

type TelemetryUpdate struct {
	Timestamp time.Time
	Type      string // "spo2", "pulse", or "waveform"
	Value     int
}

type GNMIServer struct {
	gnmi.UnimplementedGNMIServer
	
	mu          sync.RWMutex
	subscribers map[chan TelemetryUpdate]struct{}
}

func NewGNMIServer() *GNMIServer {
	return &GNMIServer{
		subscribers: make(map[chan TelemetryUpdate]struct{}),
	}
}

// Broadcast sends the update to all active gNMI subscriptions
func (s *GNMIServer) Broadcast(update TelemetryUpdate) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for ch := range s.subscribers {
		select {
		case ch <- update:
		default:
			// If channel is full, drop it to avoid blocking the broadcaster
		}
	}
}

func (s *GNMIServer) Subscribe(stream gnmi.GNMI_SubscribeServer) error {
	// Read the initial SubscribeRequest
	req, err := stream.Recv()
	if err != nil {
		return err
	}
	
	// We only support STREAM subscriptions for this simple implementation
	if req.GetSubscribe() == nil {
		return status.Error(codes.InvalidArgument, "request must be a subscription")
	}

	// Register subscriber
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

	// Send initial sync response
	stream.Send(&gnmi.SubscribeResponse{
		Response: &gnmi.SubscribeResponse_SyncResponse{
			SyncResponse: true,
		},
	})

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case update := <-ch:
			// Map update to gNMI path
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
				return err
			}
		}
	}
}

func (s *GNMIServer) Capabilities(ctx context.Context, req *gnmi.CapabilityRequest) (*gnmi.CapabilityResponse, error) {
	return &gnmi.CapabilityResponse{
		GNMIVersion: "0.8.0",
		SupportedModels: []*gnmi.ModelData{},
		SupportedEncodings: []gnmi.Encoding{gnmi.Encoding_PROTO},
	}, nil
}

func StartGRPCServer(port int, srv *GNMIServer) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	
	g := grpc.NewServer()
	gnmi.RegisterGNMIServer(g, srv)
	
	go func() {
		if err := g.Serve(lis); err != nil {
			fmt.Printf("gRPC server failed: %v\n", err)
		}
	}()
	return nil
}
