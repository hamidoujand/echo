package mid

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/hamidoujand/echo/errs"
	"github.com/hamidoujand/echo/web"
)

func Error(log *slog.Logger) web.Middleware {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			err := next(ctx, w, r)
			//no error
			if err == nil {
				return nil
			}

			//we got an error
			var appError *errs.Error
			if errors.As(err, &appError) {
				log.Error("handling app error during request",
					"err", appError.Message,
					"source_file", filepath.Base(appError.Filename),
					"function", filepath.Base(appError.FuncName),
				)
				return web.Respond(ctx, w, appError.Code, appError)
			} else {
				//log
				log.Error("handling unknown error during request", "err", err)
				//respond with a 500
				err := errs.New(http.StatusInternalServerError, err)
				return web.Respond(ctx, w, http.StatusInternalServerError, err)
			}

		}
	}
}
