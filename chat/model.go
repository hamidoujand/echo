package chat

import (
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type User struct {
	ID       uuid.UUID       `json:"id"`
	Name     string          `json:"name"`
	LastPing time.Time       `json:"lastPing"`
	LastPong time.Time       `json:"lastPong"`
	Conn     *websocket.Conn `json:"-"`
}

type inMessage struct {
	ToID uuid.UUID `json:"toID"`
	Text string    `json:"text"`
}

type outMessage struct {
	From User   `json:"from"`
	Text string `json:"text"`
}

type busMessage struct {
	CapID    string    `json:"capID"`
	FromID   uuid.UUID `json:"fromID"`
	FromName string    `json:"fromName"`
	ToID     uuid.UUID `json:"toID"`
	Text     string    `json:"text"`
}

type Connection struct {
	Conn     *websocket.Conn
	LastPong time.Time
	LastPing time.Time
}
