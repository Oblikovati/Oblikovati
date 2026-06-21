// SPDX-License-Identifier: GPL-2.0-only

//go:build darwin

package usagestats

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// totalRAMBytes reads hw.memsize via sysctl. Returns 0 on error.
func totalRAMBytes() int64 {
	n, err := unix.SysctlUint64("hw.memsize")
	if err != nil {
		return 0
	}
	return int64(n)
}

// cpuModel reads machdep.cpu.brand_string via sysctl (e.g. "Apple M1 Pro"). Returns "" on error.
func cpuModel() string {
	s, err := unix.Sysctl("machdep.cpu.brand_string")
	if err != nil {
		return ""
	}
	return s
}

// homeVolumeCapacityBytes returns the total size of the filesystem holding the probe dir via
// statfs (blocks × block size). On Darwin, Bsize is uint32, so both fields convert. Returns 0
// on error.
func homeVolumeCapacityBytes() int64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(probeDir(), &stat); err != nil {
		return 0
	}
	return int64(stat.Blocks) * int64(stat.Bsize)
}
