package notify

import (
	"sync"

	"github.com/gofiber/websocket/v2"
)

var clients = make(map[string]*websocket.Conn)
var mu sync.Mutex

func Register(username string, conn *websocket.Conn) {
	mu.Lock()
	defer mu.Unlock()
	clients[username] = conn
}

func Unregister(username string) {
	mu.Lock()
	defer mu.Unlock()
	delete(clients, username)
}

func SendToUser(username string, message []byte) {
	mu.Lock()
	conn, ok := clients[username]
	mu.Unlock()

	if !ok {
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
		Unregister(username)
	}
}
