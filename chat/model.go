package chat

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type User struct {
	ID       common.Address  `json:"id"`
	Name     string          `json:"name"`
	LastPing time.Time       `json:"lastPing"`
	LastPong time.Time       `json:"lastPong"`
	Conn     *websocket.Conn `json:"-"`
}

type outgoingUser struct {
	ID    common.Address `json:"id"`
	Name  string         `json:"name"`
	Nonce uint64         `json:"nonce"`
}

type inMessage struct {
	ToID      common.Address `json:"toID"`
	Text      string         `json:"text"`
	FromNonce uint64         `json:"fromNonce"`
	V         *big.Int       `json:"v"`
	R         *big.Int       `json:"r"`
	S         *big.Int       `json:"s"`
}

type outMessage struct {
	From outgoingUser `json:"from"`
	Text string       `json:"text"`
}

type busMessage struct {
	CapID     uuid.UUID      `json:"capID"`
	FromID    common.Address `json:"fromID"`
	FromName  string         `json:"fromName"`
	ToID      common.Address `json:"toID"`
	Text      string         `json:"text"`
	FromNonce uint64         `json:"fromNonce"`
	V         *big.Int       `json:"v"`
	R         *big.Int       `json:"r"`
	S         *big.Int       `json:"s"`
}

type Connection struct {
	Conn     *websocket.Conn
	LastPong time.Time
	LastPing time.Time
}
