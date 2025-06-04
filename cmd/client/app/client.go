package app

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

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
	Encrypted bool   `json:"encrypted"`
	From      user   `json:"from"`
	Text      []byte `json:"text"`
}

type outMessage struct {
	ToID      common.Address `json:"toID"`
	Text      []byte         `json:"text"`
	FromNonce uint64         `json:"fromNonce"`
	Encrypted bool           `json:"encrypted"`
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

			decryptedMsg, encryptedMsg, err := c.processReceivedMessages(inMsg)
			if err != nil {
				uiWriter("system", fmt.Sprintf("failed to process received messages: %s", err))
				return
			}

			if !bytes.HasPrefix(decryptedMsg, []byte("/")) {
				uiDecryptedMsg := formatMessage(usr.Name, decryptedMsg)
				dbEncryptedMsg := formatMessage(usr.Name, encryptedMsg)

				if err := c.db.AddMessage(inMsg.From.ID, dbEncryptedMsg); err != nil {
					uiWriter("system", fmt.Sprintf("failed to add message: %s", err))
					return
				}

				uiWriter(inMsg.From.ID.Hex(), string(uiDecryptedMsg))
			}
		}
	}()

	return nil
}

func (c *Client) Send(to common.Address, msg []byte) error {
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

	decryptedMsg, encryptedMsg, err := c.processSendMessages(usr, msg)
	if err != nil {
		return fmt.Errorf("processSendMessages: %w", err)
	}

	//default msg is decrypted
	msg = decryptedMsg
	isEncrypted := len(encryptedMsg) != 0
	if isEncrypted {
		msg = encryptedMsg
	}

	dataToSign := struct {
		ToID      common.Address
		Text      []byte
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
		Encrypted: isEncrypted,
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

	if !bytes.HasPrefix(decryptedMsg, []byte("/")) {
		uiDecryptedMsg := formatMessage("You", decryptedMsg)
		dbEncryptedMsg := formatMessage("You", encryptedMsg)

		if err := c.db.AddMessage(to, uiDecryptedMsg); err != nil {
			return fmt.Errorf("addMessage: %w", err)
		}

		c.uiWriter(to.String(), string(dbEncryptedMsg))
	}

	return nil
}

func (c *Client) processSendMessages(usr User, msg []byte) ([]byte, []byte, error) {
	//not a command, normal messages
	if !bytes.HasPrefix(msg, []byte("/")) {
		//usr does not have a key for encryption
		if len(usr.Key) == 0 {
			return msg, nil, nil
		}

		//usr does have a key, encrypt messages
		pk, err := parseRSAPublicKey(usr.Key)
		if err != nil {
			return nil, nil, fmt.Errorf("parseRSAPublicKey: %w", err)
		}

		encryptedData, err := rsa.EncryptPKCS1v15(rand.Reader, pk, msg)
		if err != nil {
			return nil, nil, fmt.Errorf("encrypt messages: %w", err)
		}

		return msg, encryptedData, nil
	}

	//its a command
	msg = bytes.ToLower(msg)
	msg = bytes.TrimSpace(msg)

	parts := bytes.Split(msg, []byte(" "))
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("%s: invalid command formant: command must be in [/<cmd> <args>]", msg)
	}

	switch {
	case bytes.Equal(parts[0], []byte("/share")):
		switch {
		case bytes.Equal(parts[1], []byte("key")):
			if c.id.RSAPublicKey == "" {
				return nil, nil, errors.New("no key to share")
			}

			return fmt.Appendf(nil, "/key %s", c.id.RSAPublicKey), nil, nil
		}
	}

	return nil, nil, fmt.Errorf("invalid command %s", msg)
}

func (c *Client) processReceivedMessages(msg inMessage) ([]byte, []byte, error) {
	text := msg.Text
	//not a command, normal message
	if !bytes.HasPrefix(text, []byte("/")) {
		//not encrypted
		if !msg.Encrypted {
			return text, nil, nil
		}

		//encrypted
		decryptedData, err := rsa.DecryptPKCS1v15(rand.Reader, c.id.RSAKey, []byte(msg.Text))
		if err != nil {
			return nil, nil, fmt.Errorf("message decryption: %w", err)
		}

		return decryptedData, text, nil
	}

	//command
	parts := bytes.SplitN(text, []byte(" "), 2)
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("%s: invalid command formant: command must be in [/key <RSA_Public>]", text)
	}

	switch {
	case bytes.Equal(parts[0], []byte("/key")):
		key := parts[1]
		if err := c.db.UpdateContactKey(msg.From.ID, key); err != nil {
			return nil, nil, fmt.Errorf("updating contact key: %w", err)
		}
		return []byte("*** Updated the contact's key ***"), nil, nil
	}

	return nil, nil, fmt.Errorf("invalid command %s", text)
}
