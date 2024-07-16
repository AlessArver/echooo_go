package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	X    int    `json:"x,omitempty"`
	Y    int    `json:"y,omitempty"`
}

var (
	clients      = make(map[*websocket.Conn]bool)
	clientsMutex sync.Mutex
	upgrader     = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	currentMessage = Message{}
)



// Middleware to add CORS headers
func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// WebSocketHandler handles incoming WebSocket connections
func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer ws.Close()

	fmt.Println("Connected:", ws.RemoteAddr().String())
	addClient(ws)
	defer removeClient(ws)

	// Send the current message to the new client
	sendMessage(ws, currentMessage)

	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				fmt.Println("Client disconnected:", ws.RemoteAddr().String())
			} else {
				log.Println("Error receiving message:", err)
			}
			break
		}

		switch msg.Type {
		case "message":
			fmt.Println("Received message:", msg.Text)
			currentMessage = msg
			broadcastMessage(msg)
		case "user-place":
			fmt.Printf("User %s moved mouse to (%d, %d)\n", ws.RemoteAddr().String(), msg.X, msg.Y)
			broadcastMessage(msg)
		default:
			log.Println("Unknown message type:", msg.Type)
		}
	}
}

func addClient(ws *websocket.Conn) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	clients[ws] = true
}

func removeClient(ws *websocket.Conn) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	delete(clients, ws)
}

func sendMessage(client *websocket.Conn, msg Message) {
	err := client.WriteJSON(msg)
	if err != nil {
		log.Printf("Error sending message to %s: %v", client.RemoteAddr().String(), err)
		client.Close()
		removeClient(client)
	}
}

func broadcastMessage(msg Message) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	for client := range clients {
		sendMessage(client, msg)
	}
}

func main() {
	http.Handle("/ws", withCORS(http.HandlerFunc(WebSocketHandler)))

	log.Println("Serving at localhost:8000...")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
