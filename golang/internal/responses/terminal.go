package responses

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fiscalismia/fiscalismia-monitoring/internal/requests"
)

const (
	CURL_DIVIDER_COUNT int    = 50
	CURL_DIVIDER_CHAR  string = "="
	NAME_LENGTH               = 15
	TYPE_LENGTH               = 6
	STATUS_LENGTH             = 5
	PING_LENGTH               = 8
	X509_LENGTH               = 4
	DETAIL_LENGTH             = 0
)

func CURL(results []requests.Result) string {
	var builder strings.Builder
	header := fmt.Sprintf("%-15s %-6s %-5s %-8s %-4s %s", "NAME", "TYPE", "STATI", "PING", "X509", "DETAIL")
	divider := paint(strings.Repeat(CURL_DIVIDER_CHAR, CURL_DIVIDER_COUNT), fgBrBlack)

	builder.WriteString(divider + "\n")
	builder.WriteString(header + "\n")
	for _, r := range results {
		var status string
		var typeOverride string
		var formattedName string
		if r.Type == "DIVIDER_CONTROL_SEQUENCE" {
			headerLength := utf8.RuneCountInString(r.Name)
			if r.Name == " EXTERNAL " {
				formattedName = paint(r.Name, bold, fgBlack, bgBrYellow)
			} else {
				formattedName = paint(r.Name, bold, fgBlack, bgBrCyan)
			}
			halfDivider := paint(strings.Repeat(CURL_DIVIDER_CHAR, CURL_DIVIDER_COUNT/2-headerLength/2), fgBrBlack)
			if CURL_DIVIDER_COUNT%2 == 1 {
				// add an extra character for uneven divider count
				endDivider := paint(strings.Repeat(CURL_DIVIDER_CHAR, CURL_DIVIDER_COUNT/2-headerLength/2+1), fgBrBlack)
				builder.WriteString(halfDivider + formattedName + endDivider + "\n")
				continue
			}
			builder.WriteString(halfDivider + formattedName + halfDivider + "\n")
			continue
		}

		detail := r.Body
		if r.Err != nil {
			status = paint(padWhitespace("Error", STATUS_LENGTH), fgBrRed)
			detail = r.Err.Error()
		}
		// color status column conditionally
		if r.Type == "http" && r.Err == nil {
			switch r.StatusCode {
			case 200:
				status = paint(padWhitespace(fmt.Sprintf("%d", r.StatusCode), STATUS_LENGTH), bold, fgBrGreen)
			case 400:
				status = paint(padWhitespace(fmt.Sprintf("%d", r.StatusCode), STATUS_LENGTH), bold, fgBrRed)
			case 404:
				status = paint(padWhitespace(fmt.Sprintf("%d", r.StatusCode), STATUS_LENGTH), bold, fgRed)
			case 500:
				status = paint(padWhitespace(fmt.Sprintf("%d", r.StatusCode), STATUS_LENGTH), bold, fgRed)
			case 429:
				status = paint(padWhitespace(fmt.Sprintf("%d", r.StatusCode), STATUS_LENGTH), bold, fgBrMagenta)
			}
		} else if r.Type == "tcp" && r.StatusCode == 1 && r.Err == nil {
			status = paint(padWhitespace("UP", STATUS_LENGTH), bold, fgBrGreen)
		} else if r.Type == "icmp" && r.StatusCode == 1 && r.Err == nil {
			status = paint(padWhitespace("OK", STATUS_LENGTH), bold, fgBrGreen)
		} else {
			status = paint(padWhitespace("n/a", STATUS_LENGTH), bold, fgBrYellow)
		}
		// color type column conditionally
		switch r.Type {
		case "http":
			if !strings.HasPrefix(r.URL, "https://") {
				typeOverride = paint(padWhitespace("https", TYPE_LENGTH), bold, fgCyan)
			} else {
				typeOverride = paint(padWhitespace(r.Type, TYPE_LENGTH), fgCyan)
			}
		case "icmp":
			typeOverride = paint(padWhitespace(r.Type, TYPE_LENGTH), fgCyan)
		case "tcp":
			typeOverride = paint(padWhitespace(r.Type, TYPE_LENGTH), fgCyan)
		}

		// color X509 validity days conditionally
		var daysUntilCertificateExpiration string
		if r.X509Info.DaysUntilExpiry != -1 {
			if r.X509Info.DaysUntilExpiry <= 7 {
				var X509Builder strings.Builder
				X509Builder.WriteString(strconv.Itoa(r.X509Info.DaysUntilExpiry))
				X509Builder.WriteString("d")
				daysUntilCertificateExpiration = paint(padWhitespace(X509Builder.String(), X509_LENGTH), bold, fgBrYellow)
			} else if r.X509Info.DaysUntilExpiry > 7 {
				var X509Builder strings.Builder
				X509Builder.WriteString(strconv.Itoa(r.X509Info.DaysUntilExpiry))
				X509Builder.WriteString("d")
				daysUntilCertificateExpiration = paint(padWhitespace(X509Builder.String(), X509_LENGTH), fgBrGreen)
			}
		} else {
			slog.Debug("TLS Certificate fallback value detected in result. DaysUntilExpiry can be discarded")
			daysUntilCertificateExpiration = paint(padWhitespace("n/a", X509_LENGTH), fgBrBlack)
		}
		line := fmt.Sprintf("%-15s %-6s %-5s %-8s %-4s %s",
			padWhitespace(r.Name, NAME_LENGTH),
			typeOverride,
			status,
			r.Latency.Round(time.Millisecond).String(),
			daysUntilCertificateExpiration,
			truncate(detail, 40),
		)
		builder.WriteString(line + "\n")
	}
	builder.WriteString(divider + "\n")

	slog.Debug("Finished constructing ANSI escape sequence with", "lines", len(results))
	return builder.String()
}

func padWhitespace(s string, padSize int) string {
	curLength := utf8.RuneCountInString(s)
	slog.Debug("Length of", "string", s, "length", curLength)
	var builder strings.Builder
	if curLength < padSize {
		builder.WriteString(s)
		builder.WriteString(strings.Repeat(" ", padSize-curLength))
		return builder.String()
	} else {
		return s
	}

}
