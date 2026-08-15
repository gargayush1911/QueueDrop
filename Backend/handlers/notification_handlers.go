package handlers

import (
	"queuedrop/notify"

	"github.com/gofiber/websocket/v2"
)

func HandleNotifications(c *websocket.Conn) {
	username := c.Locals("username").(string)

	notify.Register(username, c)
	defer notify.Unregister(username)

	for {
		if _, _, err := c.ReadMessage(); err != nil {
			break
		}
	}
}
