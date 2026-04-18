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

	"github.com/fiscalismia/fiscalismia-monitoring/internal/server"
	"github.com/fiscalismia/fiscalismia-monitoring/internal/version"
)

func main() {
	level := slog.LevelDebug
	if env := os.Getenv("ENVIRONMENT"); env == "demo" || env == "prod" {
		level = slog.LevelInfo
	}
	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      level,
		TimeFormat: time.Kitchen,
	})))

	slog.Info("starting healthcheck server",
		"version", version.Version,
		"commit", version.Commit,
		"buildTime", version.BuildTime,
	)

	addr := os.Getenv("HEALTHCHECK_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8445" // loopback-only by default
	}

	srv := server.New(addr)

	// Run server in a goroutine so main can wait on signals.
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Listen for SIGINT (Ctrl+C) and SIGTERM (Podman/K8s stop).
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		slog.Error("server failed", "err", err)
		os.Exit(1)
	case sig := <-stop:
		slog.Info("received signal, shutting down", "signal", sig.String())
	}

	// Give in-flight requests 10s to complete, then force-close.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "err", err)
		os.Exit(1)
	}
	slog.Info("server exited cleanly")
}
