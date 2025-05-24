package app

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/websocket"
	"github.com/hamidoujand/echo/signature"
)

type UIWriter func(id, msg string)
type UpdateContact func(id, name string)

type user struct {
	ID   common.Address `json:"id"`
	Name string         `json:"name"`
}

type inMessage struct {
	From user   `json:"from"`
	Text string `json:"text"`
}

type outMessage struct {
	ToID  common.Address `json:"toID"`
	Text  string         `json:"text"`
	Nonce uint64         `json:"nonce"`
	V     *big.Int       `json:"v"`
	R     *big.Int       `json:"r"`
	S     *big.Int       `json:"s"`
}

type Client struct {
	privateKey *ecdsa.PrivateKey
	id         common.Address
	conn       *websocket.Conn
	url        string
	contacts   *Database
	uiWriter   UIWriter
}

func NewClient(id common.Address, private *ecdsa.PrivateKey, url string, contacts *Database) *Client {
	return &Client{
		privateKey: private,
		id:         id,
		url:        url,
		contacts:   contacts,
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
		ID:   c.id.Hex(),
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
			usr, err := c.contacts.LookupContact(inMsg.From.ID)
			switch {
			case err != nil:
				var err error
				usr, err = c.contacts.AddContact(inMsg.From.ID, inMsg.From.Name)
				if err != nil {
					uiWriter("system", fmt.Sprintf("failed to add user into contacts: %s", err))
					return
				}

				updateContact(inMsg.From.ID.Hex(), inMsg.From.Name)
			default:
				inMsg.From.Name = usr.Name
			}

			formattedMsg := formatMessage(usr.Name, inMsg.Text)

			if err := c.contacts.AddMessage(inMsg.From.ID, formattedMsg); err != nil {
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

	dataToSign := struct {
		ToID  common.Address
		Text  string
		Nonce uint64
	}{
		ToID:  to,
		Text:  msg,
		Nonce: 1,
	}

	v, r, s, err := signature.Sign(dataToSign, c.privateKey)
	if err != nil {
		return fmt.Errorf("sign: %w", err)
	}

	outMsg := outMessage{
		ToID:  to,
		Text:  msg,
		Nonce: 1,
		V:     v,
		R:     r,
		S:     s,
	}

	bs, err := json.Marshal(outMsg)
	if err != nil {
		return fmt.Errorf("marshaling inMessage: %w", err)
	}

	if err := c.conn.WriteMessage(websocket.TextMessage, bs); err != nil {
		return fmt.Errorf("writing message to the conn: %w", err)
	}

	msg = formatMessage("You", msg)
	if err := c.contacts.AddMessage(to, msg); err != nil {
		return fmt.Errorf("addMessage: %w", err)
	}

	c.uiWriter(to.String(), msg)

	return nil
}
