package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fiscalismia/fiscalismia-monitoring/internal/config"
	"github.com/fiscalismia/fiscalismia-monitoring/internal/requests"
	"github.com/fiscalismia/fiscalismia-monitoring/internal/responses"
	"github.com/fiscalismia/fiscalismia-monitoring/internal/version"
)

type healthResponse struct {
	Status       string `json:"status"`
	Version      string `json:"version"`
	Service      string `json:"service"`
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

const (
	TERMINAL_CLIENTS string = "curl wget go-http-client fetch python-requests powershell"
)

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
		Status:       "OK",
		Version:      version.Version,
		Service:      "fiscalismia-monitoring",
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
	results := []requests.Result{{Name: " EXTERNAL ", Type: requests.TYPE_DIVIDER}}
	results = append(results, s.probeTargets(r.Context(), s.config.Targets.External)...)
	if s.isRemote {
		slog.Info("Remote Deployment detected. Running internal network probes...")
		internal := s.probeTargets(r.Context(), s.config.Targets.Internal)
		// adds a special type to check for in results to add a divider between External and Internal Targets
		results = append(results, requests.Result{Name: " INTERNAL ", Type: requests.TYPE_DIVIDER})
		results = append(results, internal...)
	}
	// r.Context() is cancelled if the client disconnects. Propagating
	// it downstream means in-flight probes terminate instead of leaking
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// extract User Agent from request Header
	headerMap := r.Header
	userAgent := headerMap.Get("User-Agent")
	if userAgent != "" {
		var terminalClients []string = strings.Fields(TERMINAL_CLIENTS)
		for _, client := range terminalClients {
			if strings.Contains(strings.ToLower(userAgent), strings.ToLower(client)) {
				slog.Info("User-agent is terminal client, formatting ANSI escape sequence...", "user-agent", userAgent, "path", r.URL.Path)
				var showDetail bool
				if strings.Contains(r.URL.Path, "/detail") {
					showDetail = true
				}
				if _, err := io.WriteString(w, responses.CURL(results, showDetail)); err != nil {
					slog.Error("write terminal infrastructure response failed", "err", err)
				}
				return
			}
		}
		slog.Debug("User-Agent not matched as any terminal client", "user-agent", userAgent)
	} else {
		slog.Warn("Failed to extract User-Agent from http Header")
	}
	slog.Info("Sending default ASCII response on invocation of", "path", r.URL.Path)
	if _, err := io.WriteString(w, responses.ASCII(results)); err != nil {
		slog.Error("write terminal infrastructure response failed", "err", err)
	}

}

// rootpath handler signaling correct endpoints to user
func (s *Server) handleRootPath(w http.ResponseWriter, r *http.Request) {
	slog.Debug("root path request", "route", ROUTE_ROOT_INFO)
	writeJSON(w, r.URL.Path, rootInfo{
		Info:     "Fiscalismia Go HTTP monitoring server.",
		Endpoint: "/goapi/",
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
	start := time.Now()
	for _, t := range targets {
		results = append(results, s.probeOne(ctx, t))
	}
	results = append(results, requests.Result{Latency: time.Since(start), Type: requests.TYPE_QUERY_DURATION})
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
			Name:       t.Name,
			Type:       t.Type,
			ExpectFail: t.ExpectFail,
			X509Info:   requests.X509CertificateValidity{IsValid: false, DaysUntilExpiry: -1, Err: nil},
			Err:        fmt.Errorf("unknown target type %q", t.Type),
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
