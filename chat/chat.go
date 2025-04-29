package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hamidoujand/echo/errs"
)

var ErrUserNotFound = errors.New("user not found")

type Chat struct {
	logger *slog.Logger
	users  map[uuid.UUID]User
	mu     sync.RWMutex
}

func New(logger *slog.Logger) *Chat {
	c := Chat{
		logger: logger,
		users:  make(map[uuid.UUID]User),
	}

	c.ping()

	return &c
}

func (c *Chat) Handshake(ctx context.Context, w http.ResponseWriter, r *http.Request) (User, error) {
	var ws websocket.Upgrader
	conn, err := ws.Upgrade(w, r, nil)
	if err != nil {
		return User{}, errs.New(http.StatusBadRequest, fmt.Errorf("upgrade failed: %w", err))
	}

	//write to the connection
	if err := conn.WriteMessage(websocket.TextMessage, []byte("Hello")); err != nil {
		return User{}, fmt.Errorf("writing message to conn: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	usr := User{
		Conn: conn,
	}

	msg, err := c.readMessage(ctx, usr)
	if err != nil {
		return User{}, fmt.Errorf("reading message: %w", err)
	}

	if err := json.Unmarshal(msg, &usr); err != nil {
		return User{}, fmt.Errorf("unmarshal msg: %w", err)
	}

	//add user
	if err := c.addUser(usr); err != nil {
		defer func() { _ = conn.Close() }()

		//user already exists,close the new connection
		if err := conn.WriteMessage(websocket.TextMessage, []byte("Already connected")); err != nil {
			return User{}, fmt.Errorf("writing message to conn: %w", err)
		}

		return User{}, fmt.Errorf("adding user: %w", err)
	}

	//send an ack
	ack := fmt.Sprintf("Welcome, %s", usr.Name)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(ack)); err != nil {
		return User{}, fmt.Errorf("writing message: %w", err)
	}

	c.logger.Info("handshake completed", "id", usr.ID, "user", usr.Name)

	return usr, nil
}

func (c *Chat) sendMessage(msg inMessage) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	from, ok := c.users[msg.FromID]
	if !ok {
		return ErrUserNotFound
	}

	to, ok := c.users[msg.ToID]
	if !ok {
		return ErrUserNotFound
	}

	m := outMessage{
		From: User{ID: from.ID, Name: from.Name},
		To:   User{ID: to.ID, Name: to.Name},
		Text: msg.Text,
	}

	if err := to.Conn.WriteJSON(m); err != nil {
		return fmt.Errorf("writing message: %w", err)
	}

	return nil
}

func (c *Chat) connections() map[uuid.UUID]*websocket.Conn {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[uuid.UUID]*websocket.Conn)
	for id, usr := range c.users {
		result[id] = usr.Conn
	}

	return result
}

func (c *Chat) ping() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			//block for the tick, then ping all connections.
			<-ticker.C
			connections := c.connections()
			for id, conn := range connections {
				if err := conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
					c.logger.Error("ping failed", "id", id, "err", err)
				}
			}
		}
	}()
}

func (c *Chat) Listen(ctx context.Context, usr User) {
	for {
		msg, err := c.readMessage(ctx, usr)
		if err != nil {
			var closedErr *websocket.CloseError
			if errors.As(err, &closedErr) {
				c.logger.Error("client disconnected", "status", "reading message", "err", err)
				return
			} else if errors.Is(err, context.Canceled) {
				c.logger.Error("client context is cancelled", "status", "reading message", "err", err)
				return
			} else {
				//another kind of err we continue
				c.logger.Error("error while reading message", "err", err)
				continue
			}
		}

		//create the inMessage
		var in inMessage
		if err := json.Unmarshal(msg, &in); err != nil {
			c.logger.Error("unmarshaling inMessage failed", "err", err)
			continue
		}

		if err := c.sendMessage(in); err != nil {
			c.logger.Error("sending message failed", "err", err)
		}
	}
}

func (c *Chat) readMessage(ctx context.Context, usr User) ([]byte, error) {
	type response struct {
		msg []byte
		err error
	}

	ch := make(chan response, 1)
	go func() {
		c.logger.Info("started read message")
		defer c.logger.Info("completed read message")

		_, msg, err := usr.Conn.ReadMessage()
		if err != nil {
			ch <- response{msg: nil, err: err}
		}

		ch <- response{msg: msg, err: nil}
	}()

	select {
	case <-ctx.Done():
		c.removeUser(usr.ID)
		return nil, ctx.Err()
	case resp := <-ch:
		if resp.err != nil {
			c.removeUser(usr.ID)
			return nil, resp.err
		}

		return resp.msg, nil
	}
}

func (c *Chat) addUser(usr User) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.users[usr.ID]; ok {
		return fmt.Errorf("user %s already exists", usr.ID)
	}

	c.users[usr.ID] = usr

	c.logger.Info("adding user to the connection map", "id", usr.ID, "name", usr.Name)
	return nil
}

func (c *Chat) removeUser(userID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	usr, ok := c.users[userID]
	if !ok {
		c.logger.Info("removing user failed, user not found", "id", usr.ID, "name", usr.Name)
		return
	}

	delete(c.users, userID)
	c.logger.Info("removing user", "id", usr.ID, "name", usr.Name)

	_ = usr.Conn.Close()
}
