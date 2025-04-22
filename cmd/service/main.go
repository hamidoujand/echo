package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

var build = "development"

func main() {
	h := logHandler(os.Stdout, build, slog.LevelDebug,
		slog.String("build", build),
		slog.String("service", "echo"),
	)

	logger := slog.New(h)
	if err := run(logger); err != nil {
		logger.Error("run", "ERROR", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	log.Info("startup", "status", "service starting")
	return nil
}

func logHandler(w io.Writer, build string, level slog.Level, attrs ...slog.Attr) slog.Handler {
	var handler slog.Handler
	fn := func(groups []string, attr slog.Attr) slog.Attr {
		if attr.Key == slog.SourceKey {
			source, ok := attr.Value.Any().(*slog.Source)
			if ok {
				filename := fmt.Sprintf("%s:%d", filepath.Base(source.File), source.Line)
				return slog.Attr{Key: slog.SourceKey, Value: slog.StringValue(filename)}
			}
		}
		return attr
	}

	if build == "development" {
		//text handler
		handler = slog.NewTextHandler(w, &slog.HandlerOptions{
			Level:       level,
			AddSource:   true,
			ReplaceAttr: fn,
		})
	} else {
		//json handler
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level:       level,
			AddSource:   true,
			ReplaceAttr: fn,
		})
	}
	handler = handler.WithAttrs(attrs)
	return handler
}
