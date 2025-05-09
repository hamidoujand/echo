package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/hamidoujand/echo/chat"
	"github.com/hamidoujand/echo/errs"
	"github.com/hamidoujand/echo/mid"
	"github.com/hamidoujand/echo/web"
)

type Config struct {
	Logger  *slog.Logger
	Chat    *chat.Chat
	Subject string
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
		chat:   cfg.Chat,
	}

	app.HandleFunc(http.MethodGet, version, "/connect", h.connect)

	return app
}

type Handler struct {
	Logger *slog.Logger
	chat   *chat.Chat
}

func (h Handler) connect(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	usr, err := h.chat.Handshake(ctx, w, r)
	if err != nil {
		return errs.New(http.StatusBadRequest, fmt.Errorf("handshake failed: %w", err))
	}

	defer func() { _ = usr.Conn.Close() }()

	h.chat.Listen(ctx, usr)

	return nil
}
