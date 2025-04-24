package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hamidoujand/echo/errs"
	"github.com/hamidoujand/echo/mid"
	"github.com/hamidoujand/echo/web"
)

type Config struct {
	Logger *slog.Logger
	WS     websocket.Upgrader
}

func Register(cfg Config) *web.App {
	app := web.NewApp(cfg.Logger,
		mid.Logger(cfg.Logger),
		mid.Error(cfg.Logger),
		mid.Panics(),
	)
	const version = "v1"

	h := Handler{
		Logger: cfg.Logger,
		WS:     cfg.WS,
	}

	app.HandleFunc(http.MethodGet, version, "/connect", h.connect)

	return app
}

type Handler struct {
	WS     websocket.Upgrader
	Logger *slog.Logger
}

func (h Handler) connect(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	//client
	c, err := h.WS.Upgrade(w, r, nil)
	if err != nil {
		return errs.New(http.StatusBadRequest, fmt.Errorf("upgrade failed: %w", err))
	}

	defer c.Close()

	usr, err := h.handshake(c)
	if err != nil {
		return errs.New(http.StatusBadRequest, fmt.Errorf("failed to handshake: %w", err))
	}
	h.Logger.Info("handshake completed", "user", usr.Name)
	// var wg sync.WaitGroup
	// wg.Add(3)

	// //ping goroutine
	// go func() {
	// 	defer wg.Done()
	// 	//ticker used to send a ping over websocket
	// 	ticker := time.NewTicker(time.Second)

	// 	for {
	// 		select {
	// 		case <-ticker.C:
	// 			if err := c.WriteMessage(websocket.TextMessage, []byte("PING")); err != nil {
	// 				return
	// 			}
	// 		}
	// 	}
	// }()

	// //reader goroutine
	// go func() {
	// 	defer wg.Done()

	// 	for {
	// 		_, msg, err := c.ReadMessage()
	// 		if err != nil {
	// 			return
	// 		}

	// 	}
	// }()

	// //writer goroutine
	// go func() {
	// 	defer wg.Done()
	// 	//read messages from another channel

	// }()

	// wg.Wait()
	//no content
	if err := web.Respond(ctx, w, http.StatusNoContent, nil); err != nil {
		return errs.New(http.StatusInternalServerError, errors.New("failed to respond"))
	}
	return nil
}

func (h Handler) handshake(c *websocket.Conn) (user, error) {
	//write to the conn
	if err := c.WriteMessage(websocket.TextMessage, []byte("Hello")); err != nil {
		return user{}, fmt.Errorf("writing message to conn: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	msg, err := h.readMessage(ctx, c)
	if err != nil {
		return user{}, fmt.Errorf("reading message: %w", err)
	}

	var usr user
	if err := json.Unmarshal(msg, &usr); err != nil {
		return user{}, fmt.Errorf("unmarshal msg: %w", err)
	}

	//send an ack
	ack := fmt.Sprintf("Welcome, %s", usr.Name)
	if err := c.WriteMessage(websocket.TextMessage, []byte(ack)); err != nil {
		return user{}, fmt.Errorf("writing message: %w", err)
	}

	return usr, nil
}

func (h Handler) readMessage(ctx context.Context, c *websocket.Conn) ([]byte, error) {
	type response struct {
		msg []byte
		err error
	}

	ch := make(chan response, 1)
	go func() {
		h.Logger.Info("started read message")
		defer h.Logger.Info("completed read message")

		_, msg, err := c.ReadMessage()
		if err != nil {
			ch <- response{msg: nil, err: err}
		}

		ch <- response{msg: msg, err: nil}
	}()

	select {
	case <-ctx.Done():
		//close the conn
		_ = c.Close()
		return nil, ctx.Err()
	case resp := <-ch:
		if resp.err != nil {
			return nil, fmt.Errorf("readMessage: %w", resp.err)
		}

		return resp.msg, nil
	}
}
