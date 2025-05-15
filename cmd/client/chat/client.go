package chat

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type WriteText func(name, msg string)

type user struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type outMessage struct {
	From user   `json:"from"`
	Text string `json:"text"`
}

type inMessage struct {
	ToID uuid.UUID `json:"toID"`
	Text string    `json:"text"`
}

type Client struct {
	id   uuid.UUID
	conn *websocket.Conn
	url  string
}

func NewClient(id uuid.UUID, url string) *Client {
	return &Client{
		id:  id,
		url: url,
	}
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) Handshake(name string, writeText WriteText) error {
	conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	c.conn = conn

	_, msg, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("readMessage: %w", err)
	}

	if string(msg) != "Hello" {
		return fmt.Errorf("expected message to be Hello, got %s", string(msg))
	}

	user := struct {
		ID   uuid.UUID
		Name string
	}{
		ID:   c.id,
		Name: name,
	}

	bs, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("marshal user: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, bs); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	_, msg, err = conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("readMessage: %w", err)
	}
	writeText("system", string(msg))

	//=========================================================================
	// listener goroutine
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				writeText("system", fmt.Sprintf("read message failed: %s", err))
				return
			}

			var out outMessage
			if err := json.Unmarshal(msg, &out); err != nil {
				writeText("system", fmt.Sprintf("unmarshaling message failed: %s", err))
				return
			}
			writeText(out.From.Name, out.Text)
		}
	}()

	return nil
}

func (c *Client) Send(to uuid.UUID, msg string) error {
	if c.conn == nil {
		return fmt.Errorf("no connection")
	}

	in := inMessage{
		ToID: to,
		Text: msg,
	}

	bs, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("marshaling inMessage: %w", err)
	}

	if err := c.conn.WriteMessage(websocket.TextMessage, bs); err != nil {
		return fmt.Errorf("writing message to the conn: %w", err)
	}
	return nil
}
