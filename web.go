package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all for local dev
	},
}

func StartWebServer(port int, gnmiSrv *GNMIServer) {
	fs := http.FileServer(http.Dir("./web"))
	http.Handle("/", fs)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsHandler(w, r, gnmiSrv)
	})

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("🌐 Web UI listening on http://localhost%s\n", addr)
	go func() {
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			log.Fatalf("Web server failed: %v", err)
		}
	}()
}

func wsHandler(w http.ResponseWriter, r *http.Request, gnmiSrv *GNMIServer) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer c.Close()

	// Subscribe to internal telemetry stream
	ch := gnmiSrv.SubscribeChan()
	defer gnmiSrv.UnsubscribeChan(ch)

	for update := range ch {
		// Send JSON update to the browser
		err := c.WriteJSON(update)
		if err != nil {
			break // Client disconnected
		}
	}
}
