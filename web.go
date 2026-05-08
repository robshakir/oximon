package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"k8s.io/klog/v2"
)

// wsUpgrader configures the Gorilla WebSocket upgrader for incoming UI connections.
var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// StartWebServer initializes an HTTP server on the specified port. It serves
// static assets for the UI and sets up a WebSocket endpoint that streams gNMI telemetry.
func StartWebServer(port int, gnmiSrv *GNMIServer) {
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("/", fs)

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsHandler(w, r, gnmiSrv)
	})

	addr := fmt.Sprintf(":%d", port)
	klog.Infof("Web UI listening on http://localhost%s", addr)

	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			klog.Fatalf("web server failed: %v", err)
		}
	}()
}

// wsHandler upgrades an incoming HTTP request to a WebSocket and streams telemetry updates.
func wsHandler(w http.ResponseWriter, r *http.Request, gnmiSrv *GNMIServer) {
	c, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		klog.Errorf("websocket upgrade failed: %v", err)
		return
	}
	defer c.Close()

	ch := gnmiSrv.SubscribeChan()
	defer gnmiSrv.UnsubscribeChan(ch)

	for update := range ch {
		if err := c.WriteJSON(update); err != nil {
			klog.Infof("client disconnected: %v", err)
			return
		}
	}
}
