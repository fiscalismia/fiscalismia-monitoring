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
	CURL_DIVIDER_COUNT int    = 50
	CURL_DIVIDER_CHAR  string = "-"
)

func CURL(results []requests.Result) string {
	var builder strings.Builder
	header := fmt.Sprintf("%-20s %-6s %-6s %-10s %-14s %s", "NAME", "TYPE", "STATUS", "PING", "CERT", "DETAIL")
	divider := strings.Repeat(CURL_DIVIDER_CHAR, CURL_DIVIDER_COUNT)

	builder.WriteString(divider + "\n")
	builder.WriteString(header + "\n")
	for _, r := range results {
		var status string
		if r.Type == "DIVIDER_CONTROL_SEQUENCE" {
			headerLength := utf8.RuneCountInString(r.Name)
			halfDivider := strings.Repeat(CURL_DIVIDER_CHAR, CURL_DIVIDER_COUNT/2-headerLength/2)
			if CURL_DIVIDER_COUNT%2 == 1 {
				// add an extra character for uneven divider count
				endDivider := strings.Repeat(CURL_DIVIDER_CHAR, CURL_DIVIDER_COUNT/2-headerLength/2+1)
				builder.WriteString(halfDivider + r.Name + endDivider + "\n")
				continue
			}
			builder.WriteString(halfDivider + r.Name + halfDivider + "\n")
			continue
		}

		detail := r.Body
		if r.Err != nil {
			status = paint("Error", fgBrRed)
			detail = r.Err.Error()
		}
		if r.Type == "http" && r.Err == nil {
			switch r.StatusCode {
			case 200:
				status = paint(fmt.Sprintf("%d", r.StatusCode), bold, fgBrGreen)
			case 400:
				status = paint(fmt.Sprintf("%d", r.StatusCode), bold, fgBrRed)
			case 404:
				status = paint(fmt.Sprintf("%d", r.StatusCode), bold, fgRed)
			case 500:
				status = paint(fmt.Sprintf("%d", r.StatusCode), bold, fgRed)
			case 429:
				status = paint(fmt.Sprintf("%d", r.StatusCode), bold, fgBrMagenta)
			}
		} else if r.Type == "tcp" && r.StatusCode == 1 && r.Err == nil {
			status = paint("UP", bold, fgBrGreen)
		} else if r.Type == "icmp" && r.StatusCode == 1 && r.Err == nil {
			status = paint("OK", bold, fgBrGreen)
		} else {
			status = paint("n/a", bold, fgBrYellow)
		}

		line := fmt.Sprintf("%-20s %-6s %-6s %-10s %-14d %s",
			r.Name,
			r.Type,
			status,
			r.Latency.Round(time.Millisecond),
			r.X509Info.DaysUntilExpiry,
			truncate(detail, 40),
		)
		builder.WriteString(line + "\n")
	}
	builder.WriteString(divider + "\n")

	slog.Debug("Finished constructing ANSI escape sequence with", "lines", len(results))
	return builder.String()
}
