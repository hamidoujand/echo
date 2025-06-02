package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/websocket"
	"github.com/hamidoujand/echo/signature"
)

type UIWriter func(id, msg string)
type UpdateContact func(id, name string)

type user struct {
	ID    common.Address `json:"id"`
	Name  string         `json:"name"`
	Nonce uint64         `json:"nonce"`
}

type inMessage struct {
	From user   `json:"from"`
	Text string `json:"text"`
}

type outMessage struct {
	ToID      common.Address `json:"toID"`
	Text      string         `json:"text"`
	FromNonce uint64         `json:"fromNonce"`
	V         *big.Int       `json:"v"`
	R         *big.Int       `json:"r"`
	S         *big.Int       `json:"s"`
}

type Client struct {
	id       ID
	conn     *websocket.Conn
	url      string
	db       *Database
	uiWriter UIWriter
}

func NewClient(id ID, url string, db *Database) *Client {
	return &Client{
		id:  id,
		url: url,
		db:  db,
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
	c.uiWriter = uiWriter

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
		ID:   c.id.Address.Hex(),
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

			var inMsg inMessage
			if err := json.Unmarshal(msg, &inMsg); err != nil {
				uiWriter("system", fmt.Sprintf("unmarshaling message failed: %s", err))
				return
			}
			//find the username
			usr, err := c.db.LookupContact(inMsg.From.ID)
			switch {
			case err != nil:
				var err error
				usr, err = c.db.AddContact(inMsg.From.ID, inMsg.From.Name)
				if err != nil {
					uiWriter("system", fmt.Sprintf("failed to add user into contacts: %s", err))
					return
				}

				updateContact(inMsg.From.ID.Hex(), inMsg.From.Name)
			default:
				inMsg.From.Name = usr.Name
			}

			//check nonce
			expectedNonce := usr.IncomingNonce + 1
			if expectedNonce != inMsg.From.Nonce {
				uiWriter("system", fmt.Sprintf("invalid nonce: got %d, expected %d", inMsg.From.Nonce, expectedNonce))
				return
			}

			//update nonce to the new value
			if err := c.db.UpdateIncomingNonce(inMsg.From.ID, expectedNonce); err != nil {
				uiWriter("system", fmt.Sprintf("failed to update contact nonce: %s", err))
				return
			}

			inMsg, err = c.processReceivedMessages(inMsg)
			if err != nil {
				uiWriter("system", fmt.Sprintf("failed to process received messages: %s", err))
				return
			}

			formattedMsg := formatMessage(usr.Name, inMsg.Text)

			if err := c.db.AddMessage(inMsg.From.ID, formattedMsg); err != nil {
				uiWriter("system", fmt.Sprintf("failed to add message: %s", err))
				return
			}

			uiWriter(inMsg.From.ID.Hex(), formattedMsg)
		}
	}()

	return nil
}

func (c *Client) Send(to common.Address, msg string) error {
	if c.conn == nil {
		return fmt.Errorf("no connection")
	}

	if len(msg) == 0 {
		return errors.New("message can not be empty")
	}

	usr, err := c.db.LookupContact(to)
	if err != nil {
		return fmt.Errorf("lookup contact: %w", err)
	}

	nonce := usr.OutgoingNonce + 1

	msg, err = c.processSendMessages(msg)
	if err != nil {
		return fmt.Errorf("processSendMessages: %w", err)
	}

	dataToSign := struct {
		ToID      common.Address
		Text      string
		FromNonce uint64
	}{
		ToID:      to,
		Text:      msg,
		FromNonce: nonce,
	}

	v, r, s, err := signature.Sign(dataToSign, c.id.ECDSAKey)
	if err != nil {
		return fmt.Errorf("sign: %w", err)
	}

	outMsg := outMessage{
		ToID:      to,
		Text:      msg,
		FromNonce: nonce,
		V:         v,
		R:         r,
		S:         s,
	}

	bs, err := json.Marshal(outMsg)
	if err != nil {
		return fmt.Errorf("marshaling inMessage: %w", err)
	}

	if err := c.conn.WriteMessage(websocket.TextMessage, bs); err != nil {
		return fmt.Errorf("writing message to the conn: %w", err)
	}

	if err := c.db.UpdateOutgoingNonce(to, nonce); err != nil {
		return fmt.Errorf("updateAppNonce: %w", err)
	}

	msg = formatMessage("You", msg)
	if err := c.db.AddMessage(to, msg); err != nil {
		return fmt.Errorf("addMessage: %w", err)
	}

	c.uiWriter(to.String(), msg)

	return nil
}

func (c *Client) processSendMessages(msg string) (string, error) {
	//not a command
	if !strings.HasPrefix(msg, "/") {
		return msg, nil
	}

	parts := strings.Split(msg, " ")
	if len(parts) != 2 {
		return "", fmt.Errorf("%s: invalid command formant: command must be in [/<cmd> <args>]", msg)
	}

	switch parts[0] {
	case "/share":
		switch parts[1] {
		case "key":
			if c.id.RSAPublicKey == "" {
				return "", errors.New("no key to share")
			}
			return fmt.Sprintf("/key %s", c.id.RSAPublicKey), nil
		}
	}

	return "", fmt.Errorf("invalid command %s", msg)
}

func (c *Client) processReceivedMessages(msg inMessage) (inMessage, error) {
	text := msg.Text
	//not a command
	if !strings.HasPrefix(text, "/") {
		return msg, nil
	}

	parts := strings.SplitN(text, " ", 2)
	if len(parts) != 2 {
		return inMessage{}, fmt.Errorf("%s: invalid command formant: command must be in [/key <RSA_Public>]", text)
	}

	switch parts[0] {
	case "/key":
		key := parts[1]
		if err := c.db.UpdateContactKey(msg.From.ID, key); err != nil {
			return inMessage{}, fmt.Errorf("updating contact key: %w", err)
		}
	}

	return inMessage{}, fmt.Errorf("invalid command %s", text)
}
