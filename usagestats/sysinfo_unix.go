// SPDX-License-Identifier: GPL-2.0-only

//go:build linux || darwin

package usagestats

import "syscall"

// homeVolumeCapacityBytes returns the total size of the filesystem holding the probe dir via
// statfs (blocks × block size). Both Linux and Darwin expose Blocks/Bsize on Statfs_t; an
// int64 conversion covers the differing field widths. Returns 0 on error.
func homeVolumeCapacityBytes() int64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(probeDir(), &stat); err != nil {
		return 0
	}
	return int64(stat.Blocks) * int64(stat.Bsize)
}
