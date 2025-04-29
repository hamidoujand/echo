package chat

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// type connection struct {
// 	id   uuid.UUID
// 	name string
// 	conn *websocket.Conn
// }

type User struct {
	ID   uuid.UUID       `json:"id"`
	Name string          `json:"name"`
	Conn *websocket.Conn `json:"-"`
}

type inMessage struct {
	FromID uuid.UUID `json:"fromID"`
	ToID   uuid.UUID `json:"toID"`
	Text   string    `json:"text"`
}

type outMessage struct {
	From User   `json:"from"`
	To   User   `json:"to"`
	Text string `json:"text"`
}
