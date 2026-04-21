package responses

import "strings"

// 4-bit 16 Color ANSI - SGR (Select Graphic Rendition) parameters from ECMA-48 / ISO 6429
const (
	// Escape sequence (33 in octal 1b in hex)
	esc   = "\x1b["
	reset = esc + "0m"

	// Text attributes
	bold      = esc + "1m"
	underline = esc + "4m"
	reverse   = esc + "7m"

	// Standard foreground (30–37)
	fgBlack   = esc + "30m"
	fgRed     = esc + "31m"
	fgGreen   = esc + "32m"
	fgYellow  = esc + "33m"
	fgBlue    = esc + "34m"
	fgMagenta = esc + "35m"
	fgCyan    = esc + "36m"
	fgWhite   = esc + "37m"

	// Bright foreground (90–97)
	fgBrBlack   = esc + "90m" // often rendered as gray
	fgBrRed     = esc + "91m"
	fgBrGreen   = esc + "92m"
	fgBrYellow  = esc + "93m"
	fgBrBlue    = esc + "94m"
	fgBrMagenta = esc + "95m"
	fgBrCyan    = esc + "96m"
	fgBrWhite   = esc + "97m"

	// Standard background (40–47)
	bgBlack   = esc + "40m"
	bgRed     = esc + "41m"
	bgGreen   = esc + "42m"
	bgYellow  = esc + "43m"
	bgBlue    = esc + "44m"
	bgMagenta = esc + "45m"
	bgCyan    = esc + "46m"
	bgWhite   = esc + "47m"

	// Bright background (100–107)
	bgBrBlack   = esc + "100m"
	bgBrRed     = esc + "101m"
	bgBrGreen   = esc + "102m"
	bgBrYellow  = esc + "103m"
	bgBrBlue    = esc + "104m"
	bgBrMagenta = esc + "105m"
	bgBrCyan    = esc + "106m"
	bgBrWhite   = esc + "107m"
)

func paint(s string, sgr ...string) string {
	if len(sgr) == 0 {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + len(reset) + 5*len(sgr))
	for _, c := range sgr {
		b.WriteString(c)
	}
	b.WriteString(s)
	b.WriteString(reset)
	return b.String()
}
