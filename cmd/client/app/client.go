package app

import (
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
)

type UIWriter func(name, msg string)
type UpdateContact func(id, name string)

type user struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type outMessage struct {
	From user   `json:"from"`
	Text string `json:"text"`
}

type inMessage struct {
	ToID string `json:"toID"`
	Text string `json:"text"`
}

type Client struct {
	id   string
	conn *websocket.Conn
	url  string
	cfg  *Config
}

func NewClient(id string, url string, cfg *Config) *Client {
	return &Client{
		id:  id,
		url: url,
		cfg: cfg,
	}
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) Handshake(name string, uiWriter UIWriter, updateContact UpdateContact) error {
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
		ID   string
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
	uiWriter("system", string(msg))

	//=========================================================================
	// listener goroutine
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				uiWriter("system", fmt.Sprintf("read message failed: %s", err))
				return
			}

			var out outMessage
			if err := json.Unmarshal(msg, &out); err != nil {
				uiWriter("system", fmt.Sprintf("unmarshaling message failed: %s", err))
				return
			}
			//find the username
			usr, err := c.cfg.LookupContact(out.From.ID)
			switch {
			case err != nil:
				if err := c.cfg.AddContact(out.From.ID, out.From.Name); err != nil {
					uiWriter("system", fmt.Sprintf("failed to add user into contacts: %s", err))
					return
				}
				updateContact(out.From.ID, out.From.Name)
			default:
				out.From.Name = usr.Name
			}

			uiWriter(out.From.Name, out.Text)
		}
	}()

	return nil
}

func (c *Client) Send(to string, msg string) error {
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
