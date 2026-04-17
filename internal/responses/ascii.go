package responses

import (
	"fmt"
	"strings"
	"time"

	"github.com/fiscalismia/fiscalismia-monitoring/internal/requests"
)

func ASCII(results []requests.Result) string {
	var builder strings.Builder
	header := fmt.Sprintf("%-20s %-6s %-6s %-10s %-14s %s", "NAME", "TYPE", "STATUS", "LATENCY", "TLS_VALID_DAYS", "DETAIL")
	divider := strings.Repeat("─", len(header)*2)

	builder.WriteString(divider + "\n")
	builder.WriteString(header + "\n")
	builder.WriteString(divider + "\n")

	for _, r := range results {
		var status string
		detail := r.Body
		if r.Err != nil {
			detail = r.Err.Error()
		}
		if r.Type == "http" && r.Err == nil {
			status = fmt.Sprintf("%d", r.StatusCode)
		}
		if r.Type == "tcp" && r.StatusCode == 1 && r.Err == nil {
			status = "UP"
		}
		if r.Type == "icmp" && r.StatusCode == 1 && r.Err == nil {
			status = "OK"
		}

		line := fmt.Sprintf("%-20s %-6s %-6s %-10s %-14d %s",
			r.Name,
			r.Type,
			status,
			r.Latency.Round(time.Millisecond),
			r.X509Info.DaysUntilExpiry,
			truncate(detail, 60),
		)
		builder.WriteString(line + "\n")
	}
	builder.WriteString(divider + "\n")

	return builder.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
