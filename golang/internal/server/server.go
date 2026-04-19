package server

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/fiscalismia/fiscalismia-monitoring/internal/config"
	"github.com/fiscalismia/fiscalismia-monitoring/internal/requests"
	"github.com/pires/go-proxyproto"
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

	// initialized conditionally based on deployed environment
	tlsEnabled bool
	hostname   string
	protocol   string
}

// New wires the HTTP server with its dependencies. Note we accept the
// config and client from outside rather than constructing them here
func New(addr string, conf *config.Config, client *requests.Client) *Server {

	///// CONDITIONAL LOGIC BASED ON ENVIRONMENT
	env := os.Getenv("ENVIRONMENT")
	var tlsCert bool
	var hostname string
	var protocol string
	switch env {
	case "production":
		protocol = "https"
		hostname = "golang.monitoring.fiscalismia.com"
		tlsCert = true
	case "demo":
		protocol = "https"
		hostname = "golang.demo.fiscalismia.com"
		tlsCert = true
	default:
		protocol = "http"
		hostname = addr
	}

	s := &Server{
		startTime:  time.Now(),
		config:     conf,
		client:     client,
		tlsEnabled: tlsCert,
		hostname:   hostname,
		protocol:   protocol,
	}

	// initialize empty TLS config
	tlsConf := &tls.Config{}
	// Seed TLS Config only on deployed environments
	if s.tlsEnabled {
		slog.Debug("X509 serving Environment detected: Initializing TLS Config.")
		cer, err := tls.LoadX509KeyPair("/etc/ssl/certs/fullchain.pem", "/etc/ssl/certs/privkey.pem")
		if err != nil {
			slog.Error("X509 certificate pair not loaded.", "err", err)
		}
		tlsConf.MinVersion = tls.VersionTLS13
		tlsConf.Certificates = []tls.Certificate{cer}
		tlsConf.CurvePreferences = []tls.CurveID{tls.CurveP521, tls.CurveP384}
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	s.httpServer = &http.Server{
		TLSConfig:         tlsConf,
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 14,
		ErrorLog:          slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
	}
	return s
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET "+ROUTE_ROOT_INFO, func(w http.ResponseWriter, r *http.Request) {
		// sinkhols erroneous paths, root path is NOT a fallback, only exact matches get routed.
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		s.handleRootPath(w, r)
	})
	mux.HandleFunc("GET "+ROUTE_GOLANG_HEALTH, s.handleHealthcheck)
	mux.HandleFunc("GET "+ROUTE_FISCALISMIA_HEALTH, s.handleInfrastructureHealth)
}

func (s *Server) Start() error {
	slog.Info("["+s.protocol+"] server listening",
		"addr", s.hostname,
		"env", os.Getenv("ENVIRONMENT"),
		"health", ROUTE_GOLANG_HEALTH,
		"infra", ROUTE_FISCALISMIA_HEALTH,
	)

	var startupError error
	if s.tlsEnabled {
		// DEPLOYED TLS HTTPS PROXY PROTOCOL V2 SERVER
		ln, err := net.Listen("tcp", s.httpServer.Addr)
		if err != nil {
			slog.Error("Tcp listener startup failed:", "err", err)
			os.Exit(1)
		}

		proxyListener := &proxyproto.Listener{
			Listener:          ln,
			ReadHeaderTimeout: 3 * time.Second,
		}
		defer proxyListener.Close()

		startupError = s.httpServer.ServeTLS(proxyListener, "", "")
	} else {
		// HTTP SERVER FOR LOCAL DEVELOPMENT
		startupError = s.httpServer.ListenAndServe()
	}
	if startupError != nil &&
		!errors.Is(startupError, http.ErrServerClosed) {
		return startupError
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("http server shutting down")
	return s.httpServer.Shutdown(ctx)
}
