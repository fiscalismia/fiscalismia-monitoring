package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/fiscalismia/fiscalismia-monitoring/internal/version"
)

type healthResponse struct {
	Status       string `json:"status"`
	Version      string `json:"version"`
	Commit       string `json:"commit"`
	BuildTime    string `json:"build_time"`
	ServerUptime string `json:"server_uptime"`
	HostUptime   string `json:"host_uptime"`
}

type rootInfo struct {
	Info     string `json:"info"`
	Endpoint string `json:"endpoint"`
	Health   string `json:"health"`
}

// private Method of the Server struct to handle the GET ROUTE_HC request
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
		ServerUptime: time.Since(s.startTime).Round(time.Second).String(),
		HostUptime:   hostUptimeStr,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("encode healthcheck response failed", "err", err)
	}
}

// private Method of the Server struct to handle the GET ROUTE_ROOT_INFO request
func (s *Server) handleRootPath(w http.ResponseWriter, r *http.Request) {
	slog.Debug("GET request received to", "route", ROUTE_ROOT_INFO)

	resp := rootInfo{
		Info:     "This is a Go HTTP Srv.",
		Endpoint: "/goapi/fiscalismia/",
		Health:   "/goapi/fiscalismia/hc",
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("encode rootInfo response failed", "err", err)
	}
}
