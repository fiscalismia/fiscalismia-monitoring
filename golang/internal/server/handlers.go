package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/fiscalismia/fiscalismia-monitoring/internal/version"
)

type healthResponse struct {
	Status       string      `json:"status"`
	Version      versionInfo `json:"version"`
	ServerUptime string      `json:"server_uptime"`
	HostUptime   string      `json:"host_uptime"`
	Timestamp    time.Time   `json:"timestamp"`
}

type versionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
}

func (s *Server) handleHealthcheck(w http.ResponseWriter, r *http.Request) {
	hostUptimeStr := "unavailable"
	if d, err := readHostUptime(); err == nil {
		hostUptimeStr = d.Round(time.Second).String()
	} else {
		slog.Warn("failed to read host uptime", "err", err)
	}

	resp := healthResponse{
		Status: "ok",
		Version: versionInfo{
			Version:   version.Version,
			Commit:    version.Commit,
			BuildTime: version.BuildTime,
		},
		ServerUptime: time.Since(s.startTime).Round(time.Second).String(),
		HostUptime:   hostUptimeStr,
		Timestamp:    time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("encode healthcheck response failed", "err", err)
	}
}
