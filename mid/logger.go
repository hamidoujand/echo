package mid

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/hamidoujand/echo/web"
)

func Logger(log *slog.Logger) web.Middleware {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			//pre
			traceID := web.GetTraceID(ctx)
			start := time.Now()
			p := r.URL.Path
			if r.URL.RawQuery != "" {
				p += "?" + r.URL.RawQuery
			}

			log.Info("request started", "traceID", traceID.String(), "method", r.Method, "remoteAddr", r.RemoteAddr, "path", p)
			err := next(ctx, w, r)
			//post
			log.Info("request completed", "traceID", traceID.String(), "status", web.GetResponseStatus(ctx), "method", r.Method, "remoteAddr", r.RemoteAddr, "path", p, "took", time.Since(start))
			return err
		}
	}
}
