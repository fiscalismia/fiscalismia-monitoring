package responses

import (
	"fmt"
	"strings"
	"time"

	"github.com/fiscalismia/fiscalismia-monitoring/internal/requests"
)

func ASCII(results []requests.Result) string {
	var builder strings.Builder
	header := fmt.Sprintf("%-20s %-6s %-6s %-10s %s", "NAME", "TYPE", "STATUS", "LATENCY", "DETAIL")
	divider := strings.Repeat("─", len(header))

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
			status = fmt.Sprintf("Status %d", r.StatusCode)
		}
		if r.Type == "tcp" && r.Err == nil {
			status = "TCP UP"
		}

		line := fmt.Sprintf("%-20s %-6s %-6s %-10s %s",
			r.Name,
			r.Type,
			status,
			r.Latency.Round(time.Millisecond),
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