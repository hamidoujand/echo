package web

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

type HandlerFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

type App struct {
	mux    *http.ServeMux
	logger *slog.Logger
	mids   []Middleware
}

func NewApp(logger *slog.Logger, mids ...Middleware) *App {
	return &App{
		mux:    http.NewServeMux(),
		logger: logger,
		mids:   mids,
	}
}

func (a *App) HandleFunc(method string, version string, path string, handlerFunc HandlerFunc, mids ...Middleware) {
	//apply handler level mids
	handlerFunc = applyMiddleware(handlerFunc, mids...)
	//apply global mids
	handlerFunc = applyMiddleware(handlerFunc, a.mids...)

	h := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		//set traceID
		traceID := uuid.New()
		ctx = setTraceID(ctx, traceID)

		if err := handlerFunc(ctx, w, r); err != nil {
			a.logger.Error("web application", "status", "failed to handle request", "err", err)
			return
		}
	}

	p := path
	if version != "" {
		p = "/" + version + path
	}
	pattern := fmt.Sprintf("%s %s", method, p)
	a.mux.HandleFunc(pattern, h)
}
