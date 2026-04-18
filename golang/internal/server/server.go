package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

const (
	ROUTE_ROOT_INFO          string = "/"
	ROUTE_GOLANG_HEALTH      string = "/goapi/fiscalismia/hc"
	ROUTE_FISCALISMIA_HEALTH string = "/goapi/fiscalismia/infrastructure/health"
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
		MaxHeaderBytes:    1 << 14, // binary 1 shifted 14x to left so 2^14 = 16384 bytes
		ErrorLog:          slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
	}
	return s
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET "+ROUTE_ROOT_INFO, s.handleRootPath)
	mux.HandleFunc("GET "+ROUTE_GOLANG_HEALTH, s.handleHealthcheck)
	mux.HandleFunc("GET "+ROUTE_FISCALISMIA_HEALTH, s.handleHealthcheck)
}

func (s *Server) Start() error {
	slog.Info("http server listening at", "addr", s.httpServer.Addr, "hc", ROUTE_GOLANG_HEALTH, "bytes", s.httpServer.MaxHeaderBytes)
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
