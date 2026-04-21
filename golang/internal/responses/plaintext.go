package responses

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fiscalismia/fiscalismia-monitoring/internal/requests"
)

const (
	PLAINTEXT_DIVIDER_COUNT     int    = 161
	PLAINTEXT_DIVIDER_CHARACTER string = "─"
)

func ASCII(results []requests.Result) string {
	var builder strings.Builder
	header := fmt.Sprintf("%-20s %-6s %-6s %-10s %-14s %s", "NAME", "TYPE", "STATUS", "LATENCY", "CERT_VALIDITY", "DETAIL")
	divider := strings.Repeat(PLAINTEXT_DIVIDER_CHARACTER, PLAINTEXT_DIVIDER_COUNT)

	builder.WriteString(divider + "\n")
	builder.WriteString(header + "\n")

	for _, r := range results {
		var status string
		if r.Type == "DIVIDER_CONTROL_SEQUENCE" {
			headerLength := utf8.RuneCountInString(r.Name)
			halfDivider := strings.Repeat(PLAINTEXT_DIVIDER_CHARACTER, PLAINTEXT_DIVIDER_COUNT/2-headerLength/2)
			if PLAINTEXT_DIVIDER_COUNT%2 == 1 {
				// add an extra character for uneven divider count
				endDivider := strings.Repeat(PLAINTEXT_DIVIDER_CHARACTER, PLAINTEXT_DIVIDER_COUNT/2-headerLength/2+1)
				builder.WriteString(halfDivider + r.Name + endDivider + "\n")
				continue
			}
			builder.WriteString(halfDivider + r.Name + halfDivider + "\n")
			continue
		}
		detail := r.Body
		if r.Err != nil {
			status = "DOWN"
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
			truncate(detail, 100),
		)
		builder.WriteString(line + "\n")
	}
	builder.WriteString(divider + "\n")

	slog.Debug("Finished constructing ASCII response with", "lines", len(results))
	return builder.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-4] + " ..."
}
