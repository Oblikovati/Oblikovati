// SPDX-License-Identifier: GPL-2.0-only

//go:build linux

package benchprof

import (
	"os"
	"strconv"
	"strings"
)

// readPeakRSS returns the process peak resident set size in bytes from /proc/self/status
// (VmHWM, reported in kB). It returns 0 if the field cannot be read — the summary then
// simply omits a meaningful peak rather than failing the run.
func readPeakRSS() uint64 {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		if !strings.HasPrefix(line, "VmHWM:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0
		}
		kb, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0
		}
		return kb * 1024
	}
	return 0
}
