package mid

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/hamidoujand/echo/errs"
	"github.com/hamidoujand/echo/web"
)

func Panics() web.Middleware {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) (err error) {
			defer func() {
				if r := recover(); r != nil {
					trace := debug.Stack()
					appErr := errs.New(http.StatusInternalServerError, fmt.Errorf("stackTrace: %s", string(trace)))
					err = appErr
				}
			}()

			return next(ctx, w, r)
		}
	}
}
