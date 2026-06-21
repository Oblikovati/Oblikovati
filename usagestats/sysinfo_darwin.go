// SPDX-License-Identifier: GPL-2.0-only

//go:build darwin

package usagestats

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// sysctlTimeout bounds each sysctl probe so a wedged process never stalls startup telemetry.
const sysctlTimeout = 2 * time.Second

// totalRAMBytes reads hw.memsize via sysctl. CGO is disabled in the unit-test build, so we
// shell out to sysctl rather than call the C API. Returns 0 on error.
func totalRAMBytes() int64 {
	out := sysctl("hw.memsize")
	if out == "" {
		return 0
	}
	n, err := strconv.ParseInt(out, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// cpuModel reads machdep.cpu.brand_string via sysctl (e.g. "Apple M1 Pro"). Returns "" on error.
func cpuModel() string {
	return sysctl("machdep.cpu.brand_string")
}

// sysctl runs `sysctl -n <key>` and returns the trimmed output, or "" on any failure.
func sysctl(key string) string {
	ctx, cancel := context.WithTimeout(context.Background(), sysctlTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "sysctl", "-n", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
