package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/ardanlabs/conf/v3"
	"github.com/hamidoujand/echo/handler"
)

var build = "development"

func main() {
	h := logHandler(os.Stdout, build, slog.LevelDebug,
		slog.String("build", build),
		slog.String("service", "echo"),
	)

	logger := slog.New(h)
	if err := run(logger); err != nil {
		logger.Error("failed to run", "ERROR", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	//---------------------------------------------------------------------------
	//configs

	cfg := struct {
		Web struct {
			ReadTimeout     time.Duration `conf:"default:5s"`
			WriteTimeout    time.Duration `conf:"default:10s"`
			IdleTimeout     time.Duration `conf:"default:120s"`
			ShutdownTimeout time.Duration `conf:"default:20s"`
			APIHost         string        `conf:"default:0.0.0.0:8000"`
		}
	}{}

	const prefix = "ECHO"
	help, err := conf.Parse(prefix, &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			return nil
		}
		return fmt.Errorf("parsing configs: %w", err)
	}

	cfgString, err := conf.String(&cfg)
	if err != nil {
		return fmt.Errorf("conf to string: %w", err)
	}

	log.Info("app configurations", "configs", cfgString)

	log.Info("service starting", "GOMAXPROCS", runtime.GOMAXPROCS(0))
	//---------------------------------------------------------------------------
	//Mux
	mux := handler.Register(handler.Config{
		Logger: log,
	})

	errCh := make(chan error)
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	server := &http.Server{
		Handler:     http.TimeoutHandler(mux, cfg.Web.WriteTimeout, "timeout"),
		Addr:        cfg.Web.APIHost,
		ReadTimeout: cfg.Web.ReadTimeout,
		IdleTimeout: cfg.Web.IdleTimeout,
	}

	go func() {
		log.Info("server starting", "host", cfg.Web.APIHost)

		if err := server.ListenAndServe(); err != nil {
			errCh <- fmt.Errorf("listenAndServe: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case sig := <-shutdownCh:
		log.Info(fmt.Sprintf("received %s signal", sig), "status", "shutting down server")

		ctx, cancel := context.WithTimeout(context.Background(), cfg.Web.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Error("failed to gracefully shutdown the server, start force shutdown", "err", err)
			if err := server.Close(); err != nil {
				return fmt.Errorf("closing server: %w", err)
			}
		}
	}

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
