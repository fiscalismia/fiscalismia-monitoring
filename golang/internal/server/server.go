package server

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/fiscalismia/fiscalismia-monitoring/internal/config"
	"github.com/fiscalismia/fiscalismia-monitoring/internal/requests"
)

const (
	ROUTE_ROOT_INFO          string = "/"
	ROUTE_GOLANG_HEALTH      string = "/goapi/fiscalismia/hc"
	ROUTE_FISCALISMIA_HEALTH string = "/goapi/fiscalismia/infrastructure/health"
)

type Server struct {
	httpServer *http.Server
	startTime  time.Time

	// Injected dependencies. Unexported so only methods inside this
	// package can reach them — external callers construct via New().
	config *config.Config
	client *requests.Client
}

// New wires the HTTP server with its dependencies. Note we accept the
// config and client from outside rather than constructing them here
func New(addr string, conf *config.Config, client *requests.Client) *Server {
	s := &Server{
		startTime: time.Now(),
		config:    conf,
		client:    client,
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	// initialize empty TLS config
	tlsConf := &tls.Config{}
	// Seed TLS Config only on deployed environments
	if env := os.Getenv("ENVIRONMENT"); env == "demo" || env == "prod" || env == "tls_test" {
		slog.Debug("X509 serving Environment detected: Initializing TLS Config.")
		cer, err := tls.LoadX509KeyPair("/etc/ssl/certs/fullchain.pem", "/etc/ssl/certs/privkey.pem")
		if err != nil {
			slog.Error("X509 certificate pair not loaded.", "err", err)
		}
		tlsConf.MinVersion = tls.VersionTLS13
		tlsConf.Certificates = []tls.Certificate{cer}
		tlsConf.CurvePreferences = []tls.CurveID{tls.CurveP521, tls.CurveP384}
	}

	s.httpServer = &http.Server{
		TLSConfig:         tlsConf,
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 14,
		ErrorLog:          slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
	}
	return s
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET "+ROUTE_ROOT_INFO, s.handleRootPath)
	mux.HandleFunc("GET "+ROUTE_GOLANG_HEALTH, s.handleHealthcheck)
	mux.HandleFunc("GET "+ROUTE_FISCALISMIA_HEALTH, s.handleInfrastructureHealth)
}

func (s *Server) Start() error {
	slog.Info("http server listening",
		"addr", s.httpServer.Addr,
		"health", ROUTE_GOLANG_HEALTH,
		"infra", ROUTE_FISCALISMIA_HEALTH,
	)
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
