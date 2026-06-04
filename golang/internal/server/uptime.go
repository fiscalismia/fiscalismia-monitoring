package server

import (
	"errors"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// readHostUptime parses /proc/uptime. In a container, this reflects the host
func readHostUptime() (time.Duration, error) {
	if runtime.GOOS != "linux" {
		return 0, errors.New("host uptime only implemented for linux")
	}

	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}
	// Format: "<uptime_seconds> <idle_seconds accumulated per cpu core>\n"
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0, errors.New("unexpected /proc/uptime format")
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(seconds * float64(time.Second)), nil
}
