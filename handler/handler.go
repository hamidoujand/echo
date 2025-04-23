package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/hamidoujand/echo/errs"
	"github.com/hamidoujand/echo/mid"
	"github.com/hamidoujand/echo/web"
)

type Config struct {
	Logger *slog.Logger
}

func Register(cfg Config) *web.App {
	app := web.NewApp(cfg.Logger,
		mid.Logger(cfg.Logger),
		mid.Error(cfg.Logger),
		mid.Panics(),
	)
	const version = "v1"

	app.HandleFunc(http.MethodGet, version, "/foo", foo)

	return app
}

func foo(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	msg := map[string]string{
		"msg": "Hello Foo",
	}
	if err := web.Respond(ctx, w, http.StatusOK, msg); err != nil {
		return errs.New(http.StatusInternalServerError, err)
	}
	return nil
}
