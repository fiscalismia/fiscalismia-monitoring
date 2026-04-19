package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lmittmann/tint"

	"github.com/fiscalismia/fiscalismia-monitoring/internal/config"
	"github.com/fiscalismia/fiscalismia-monitoring/internal/requests"
	"github.com/fiscalismia/fiscalismia-monitoring/internal/server"
	"github.com/fiscalismia/fiscalismia-monitoring/internal/version"
)

func main() {
	///// GLOBAL LOGGING
	level := slog.LevelDebug
	if env := os.Getenv("ENVIRONMENT"); env == "production" {
		level = slog.LevelInfo
	}
	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      level,
		TimeFormat: time.Kitchen,
	})))

	slog.Info("starting fiscalismia-healthcheck",
		"version", version.Version,
		"commit", version.Commit,
		"buildTime", version.BuildTime,
	)

	///// CONFIG — loaded once at startup, not on every request.
	configPath := os.Getenv("HEALTHCHECK_CONFIG")
	if env := os.Getenv("ENVIRONMENT"); env == "production" {
		slog.Debug("[prod] Environment target.yml override")
		configPath = "./targets-prod.yml"
	} else {
		configPath = "./targets-demo.yml"
	}
	conf, err := config.Load(configPath)
	if err != nil {
		slog.Error("could not load config", "path", configPath, "err", err)
		os.Exit(1)
	}

	///// HTTP CLIENT — constructed once, shared across handlers.
	client := requests.CreateClient(conf.GlobalTimeout)

	///// SERVER
	addr := os.Getenv("HEALTHCHECK_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8445"
	}
	srv := server.New(addr, conf, client)

	serverErr := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		slog.Error("server failed", "err", err)
		os.Exit(1)
	case sig := <-stop:
		slog.Warn("received signal, shutting down", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "err", err)
		os.Exit(1)
	}
	slog.Info("server exited cleanly")
}
