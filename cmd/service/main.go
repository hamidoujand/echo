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
	"strings"
	"syscall"
	"time"

	"github.com/ardanlabs/conf/v3"
	"github.com/google/uuid"
	"github.com/hamidoujand/echo/chat"
	"github.com/hamidoujand/echo/handler"
	"github.com/hamidoujand/echo/users"
	"github.com/nats-io/nats.go"
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
		NATS struct {
			Host    string `conf:"default:demo.nats.io"`
			Name    string `conf:"default:cap"`
			Subject string `conf:"default:cap"`
			CapID   string `conf:"default:infra"`
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
	//------------------------------------------------------------------------------
	//NATS
	if !strings.HasSuffix(cfg.NATS.CapID, "/") {
		cfg.NATS.CapID += "/"
	}

	capIDFilename := cfg.NATS.CapID + "id.txt"

	_, err = os.Stat(capIDFilename)
	if err != nil {
		if err := os.MkdirAll(cfg.NATS.CapID, 0755); err != nil {
			return fmt.Errorf("mkdirAll: %w", err)
		}

		//find not found , create one
		f, err := os.Create(capIDFilename)
		if err != nil {
			return fmt.Errorf("creating new capID file: %w", err)
		}
		capID := uuid.NewString()
		if _, err := f.WriteString(capID); err != nil {
			return fmt.Errorf("writing capID file: %w", err)
		}
		f.Close()
	}

	f, err := os.Open(cfg.NATS.CapID)
	if err != nil {
		return fmt.Errorf("open capID file: %s: %w", cfg.NATS.CapID, err)
	}

	bs, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("readAll capID file: %w", err)
	}

	capID, err := uuid.ParseBytes(bs)
	if err != nil {
		return fmt.Errorf("capID must be a uuid: %w", err)
	}

	f.Close()

	log.Info("startup", "capID", capID)

	nc, err := nats.Connect(cfg.NATS.Host)
	if err != nil {
		return fmt.Errorf("nats connect: %w", err)
	}
	defer nc.Close()
	users := users.New(log)

	chat, err := chat.New(log, users, nc, cfg.NATS.Subject, capID.String())
	if err != nil {
		return fmt.Errorf("creating chat obj: %w", err)
	}

	//---------------------------------------------------------------------------
	//Mux
	mux := handler.Register(handler.Config{
		Logger:  log,
		Chat:    chat,
		Subject: cfg.NATS.Subject,
	})

	errCh := make(chan error)
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	server := &http.Server{
		Handler:     mux,
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
