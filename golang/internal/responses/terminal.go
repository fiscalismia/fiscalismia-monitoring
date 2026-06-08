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
	CURL_DIVIDER_CHAR           string = "="
	NAME_LENGTH                 int    = 15
	TYPE_LENGTH                 int    = 6
	STATUS_LENGTH               int    = 5
	PING_LENGTH                 int    = 8
	X509_LENGTH                 int    = 4
	DETAIL_DEFAULT_LENGTH       int    = 17
	DETAIL_TRUNCATION_NUM       int    = 50
	DETAIL_ERROR_TRUNCATION_NUM int    = 90
)

func CURL(results []requests.Result, showDetail bool) string {
	var builder strings.Builder
	header := fmt.Sprintf("%-15s %-6s %-5s %-8s %-4s %s", "NAME", "TYPE", "STATI", "PING", "X509", "DETAIL")
	// dynamically compute header length (+ add padding for whitespace between Columns)
	dividerCount := NAME_LENGTH + TYPE_LENGTH + STATUS_LENGTH + PING_LENGTH + X509_LENGTH + DETAIL_DEFAULT_LENGTH + 5
	if showDetail {
		dividerCount = dividerCount + DETAIL_TRUNCATION_NUM - DETAIL_DEFAULT_LENGTH
	}
	divider := paint(strings.Repeat(CURL_DIVIDER_CHAR, dividerCount), fgBrBlack)

	builder.WriteString(divider + "\n")
	builder.WriteString(header + "\n")
	for _, r := range results {
		var status string
		var typeOverride string
		var formattedName string
		if r.Type == requests.TYPE_DIVIDER {
			headerLength := utf8.RuneCountInString(r.Name)
			if r.Name == " EXTERNAL " {
				formattedName = paint(r.Name, bold, fgBlack, bgBrYellow)
			} else {
				formattedName = paint(r.Name, bold, fgBlack, bgBrCyan)
			}
			halfDivider := paint(strings.Repeat(CURL_DIVIDER_CHAR, dividerCount/2-headerLength/2), fgBrBlack)
			if dividerCount%2 == 1 {
				// add an extra character for uneven divider count
				endDivider := paint(strings.Repeat(CURL_DIVIDER_CHAR, dividerCount/2-headerLength/2+1), fgBrBlack)
				builder.WriteString(halfDivider + formattedName + endDivider + "\n")
				continue
			}
			builder.WriteString(halfDivider + formattedName + halfDivider + "\n")
			continue
		} else if r.Type == requests.TYPE_QUERY_DURATION {
			queryDurationStr := ("query duration: " + r.Latency.Round(time.Millisecond).String())
			durationStrLen := utf8.RuneCountInString(queryDurationStr)
			// paint after length count to avoid adding escape sequence runes
			queryDurationStr = paint(strings.Repeat(" ", dividerCount-durationStrLen)+queryDurationStr+"\n", fgBlue)
			builder.WriteString(queryDurationStr)
			continue
		}

		detail := r.Body

		// conditionally render detail string based on route
		var detailStr string
		if showDetail {
			detailStr = truncate(detail, DETAIL_TRUNCATION_NUM)
		} else {
			detailStr = paint("see /detail route", fgBrBlack)
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
			if strings.HasPrefix(r.URL, "https://") {
				typeOverride = paint(padWhitespace("https", TYPE_LENGTH), fgCyan)
			} else {
				typeOverride = paint(padWhitespace(r.Type, TYPE_LENGTH), fgBrCyan)
			}
		case "icmp":
			typeOverride = paint(padWhitespace(r.Type, TYPE_LENGTH), fgBrCyan)
		case "tcp":
			typeOverride = paint(padWhitespace(r.Type, TYPE_LENGTH), fgBrCyan)
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
			slog.Debug("TLS Certificate fallback value detected in result. DaysUntilExpiry can be discarded", "type", r.Type)
			daysUntilCertificateExpiration = paint(padWhitespace("-", X509_LENGTH), fgBrBlack)
		}

		// Final centralized Error exclusion handling to avoid having to handle errors for each column
		if r.Err != nil {
			if r.ExpectFail {
				slog.Debug("Expected Query Error encountered", "code", r.StatusCode, "url", r.URL, "host", r.Host, "type", r.Type)
				status = paint(padWhitespace("DROP", STATUS_LENGTH), fgBrMagenta)
				daysUntilCertificateExpiration = paint(padWhitespace("n/a", X509_LENGTH), fgBrBlack)
				if showDetail {
					detailStr = truncate(r.Err.Error(), DETAIL_TRUNCATION_NUM)
				} else {
					detailStr = paint("Failure expected.", fgBrMagenta)
				}
			} else {
				slog.Warn("Unexpected Query Error encountered", "code", r.StatusCode, "url", r.URL, "host", r.Host, "type", r.Type)
				status = paint(padWhitespace("ERR", STATUS_LENGTH), bold, fgRed)
				if showDetail {
					detailStr = truncate(r.Err.Error(), DETAIL_ERROR_TRUNCATION_NUM)
				} else {
					detailStr = paint("/detail for error", fgBrBlack)
				}
			}
		}

		line := fmt.Sprintf("%-15s %-6s %-5s %-8s %-4s %s",
			padWhitespace(r.Name, NAME_LENGTH),
			typeOverride,
			status,
			padWhitespace(r.Latency.Round(time.Millisecond).String(), PING_LENGTH),
			daysUntilCertificateExpiration,
			detailStr,
		)
		builder.WriteString(line + "\n")
	}
	builder.WriteString(divider + "\n")

	slog.Debug("Finished constructing ANSI escape sequence with", "lines", len(results))
	return builder.String()
}

func padWhitespace(s string, padSize int) string {
	curLength := utf8.RuneCountInString(s)
	var builder strings.Builder
	if curLength < padSize {
		builder.WriteString(s)
		builder.WriteString(strings.Repeat(" ", padSize-curLength))
		return builder.String()
	} else {
		return s
	}

}
