package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

type Server struct {
	httpServer *http.Server
	startTime  time.Time
}

func New(addr string) *Server {
	s := &Server{startTime: time.Now()}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 14,
		ErrorLog:          slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
	}
	return s
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /goapi/fiscalismia/hc", s.handleHealthcheck)
}

func (s *Server) Start() error {
	slog.Info("http server starting", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("http server shutting down")
	return s.httpServer.Shutdown(ctx)
}
