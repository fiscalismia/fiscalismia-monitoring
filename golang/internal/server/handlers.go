package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/fiscalismia/fiscalismia-monitoring/internal/config"
	"github.com/fiscalismia/fiscalismia-monitoring/internal/requests"
	"github.com/fiscalismia/fiscalismia-monitoring/internal/responses"
	"github.com/fiscalismia/fiscalismia-monitoring/internal/version"
)

type healthResponse struct {
	Status       string `json:"status"`
	Version      string `json:"version"`
	Commit       string `json:"commit"`
	BuildTime    string `json:"build_time"`
	Environment  string `json:"environment"`
	ServerUptime string `json:"server_uptime"`
	HostUptime   string `json:"host_uptime"`
}

type rootInfo struct {
	Info     string `json:"info"`
	Endpoint string `json:"endpoint"`
	Health   string `json:"health"`
}

// handleHealthcheck is a cheap liveness probe — no outbound I/O.
func (s *Server) handleHealthcheck(w http.ResponseWriter, r *http.Request) {
	slog.Debug("GET request received to", "route", ROUTE_GOLANG_HEALTH)

	hostUptimeStr := "unavailable"
	if d, err := readHostUptime(); err == nil {
		hostUptimeStr = d.Round(time.Second).String()
	} else {
		slog.Warn("failed to read host uptime", "err", err)
	}

	resp := healthResponse{
		Status:       "ok",
		Version:      version.Version,
		Commit:       version.Commit,
		BuildTime:    version.BuildTime,
		Environment:  os.Getenv("ENVIRONMENT"),
		ServerUptime: time.Since(s.startTime).Round(time.Second).String(),
		HostUptime:   hostUptimeStr,
	}
	writeJSON(w, r.URL.Path, resp)
}

// handleInfrastructureHealth is the expensive endpoint — it probes
// every target from targets.yml and returns an ASCII status table.
func (s *Server) handleInfrastructureHealth(w http.ResponseWriter, r *http.Request) {
	slog.Debug("GET request received to", "route", ROUTE_FISCALISMIA_HEALTH)
	results := s.probeTargets(r.Context(), s.config.Targets.External)
	if s.isRemote {
		slog.Info("Remote Deployment detected. Running internal network probes...")
		internal := s.probeTargets(r.Context(), s.config.Targets.Internal)
		results = append(results, internal...)
	}
	// r.Context() is cancelled if the client disconnects. Propagating
	// it downstream means in-flight probes terminate instead of leaking
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	slog.Info("Sending ASCII response on invocation of", "path", r.URL.Path)
	if _, err := io.WriteString(w, responses.ASCII(results)); err != nil {
		slog.Error("write infrastructure response failed", "err", err)
	}
}

func (s *Server) handleRootPath(w http.ResponseWriter, r *http.Request) {
	slog.Debug("root path request", "route", ROUTE_ROOT_INFO)
	writeJSON(w, r.URL.Path, rootInfo{
		Info:     "Fiscalismia Go HTTP monitoring server.",
		Endpoint: "/goapi/fiscalismia/",
		Health:   ROUTE_GOLANG_HEALTH,
	})
}

// probeTargets runs one probe per target and collects results.
//
// SEQUENTIAL TODAY — CONCURRENCY-READY BY DESIGN.
//
// The shape here is the whole point. Each target maps to exactly one
// Result with no shared mutable state between iterations, and probeOne
// has no ordering dependency. The only aggregation is the final append.
//
// When this goes concurrent, only the body of this function changes:
// spawn a goroutine per target, each writes its Result onto a channel,
// a receiver loop collects them, bounded by either sync.WaitGroup +
// channel close or golang.org/x/sync/errgroup. The handler above and
// probeOne below both stay untouched. That's the boundary.
func (s *Server) probeTargets(ctx context.Context, targets []config.Target) []requests.Result {
	results := make([]requests.Result, 0, len(targets))
	for _, t := range targets {
		results = append(results, s.probeOne(ctx, t))
	}
	return results
}

// probeOne dispatches a single probe by target type. Can be called concurrently
func (s *Server) probeOne(ctx context.Context, t config.Target) requests.Result {
	switch t.Type {
	case "http":
		slog.Debug("probe http", "name", t.Name, "url", t.URL)
		result := s.client.QueryHTTP(ctx, &t)
		if t.X509Verify {
			result.X509Info = s.client.VerifyTLSCertificate(ctx, &t, s.config.RootDomain)
		}
		return result
	case "tcp":
		slog.Debug("probe tcp", "name", t.Name, "host", t.Host)
		return s.client.QueryTCP(ctx, &t)
	case "icmp":
		slog.Debug("probe icmp", "name", t.Name, "host", t.Host)
		return s.client.QueryICMP(ctx, &t)
	default:
		return requests.Result{
			Name: t.Name,
			Type: t.Type,
			Err:  fmt.Errorf("unknown target type %q", t.Type),
		}
	}
}

// writeJSON is a small helper function for responses
func writeJSON(w http.ResponseWriter, path string, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	slog.Debug("Sending JSON response on invocation of", "path", path)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("json encode failed", "err", err)
	}
}
