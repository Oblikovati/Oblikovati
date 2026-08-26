// SPDX-License-Identifier: GPL-2.0-only

//go:build linux

package usagestats

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

// totalRAMBytes reads MemTotal from /proc/meminfo. Returns 0 if unreadable or unparsable.
func totalRAMBytes() int64 {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	return parseMemTotalKB(b) * 1024
}

// parseMemTotalKB extracts the MemTotal value (in kB) from /proc/meminfo contents, or 0 if the
// line is absent or malformed. Kept pure so the parsing is unit-tested without /proc.
func parseMemTotalKB(meminfo []byte) int64 {
	for line := range strings.SplitSeq(string(meminfo), "\n") {
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line) // "MemTotal:", "16327084", "kB"
		if len(fields) < 2 {
			return 0
		}
		kb, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return 0
		}
		return kb
	}
	return 0
}

// cpuModel returns the first CPU "model name" from /proc/cpuinfo. Returns "" if unreadable.
func cpuModel() string {
	b, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	return parseCPUModelName(b)
}

// parseCPUModelName extracts the first "model name" value from /proc/cpuinfo contents, or "".
// Kept pure so the parsing is unit-tested without /proc.
func parseCPUModelName(cpuinfo []byte) string {
	for line := range strings.SplitSeq(string(cpuinfo), "\n") {
		key, val, ok := strings.Cut(line, ":")
		if ok && strings.TrimSpace(key) == "model name" {
			return strings.TrimSpace(val)
		}
	}
	return ""
}

// homeVolumeCapacityBytes returns the total size of the filesystem holding the probe dir via
// statfs (blocks × block size). On Linux, Bsize is already int64. Returns 0 on error.
func homeVolumeCapacityBytes() int64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(probeDir(), &stat); err != nil {
		return 0
	}
	return int64(stat.Blocks) * stat.Bsize
}
