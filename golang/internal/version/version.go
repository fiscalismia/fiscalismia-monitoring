package version

import (
	"fmt"
	"runtime/debug"
)

// Overwritten by linker at buildtime, wired in Makefile
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// Fills Commit from VCS build info when -ldflags did not provide it,
// so `go run ./cmd/healthcheck` still reports a meaningful commit.
func init() {
	if Commit != "unknown" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && s.Value != "" {
			Commit = s.Value
		}
	}
}

func String() string {
	return fmt.Sprintf("%s (%s, %s)", Version, Commit, BuildTime)
}
