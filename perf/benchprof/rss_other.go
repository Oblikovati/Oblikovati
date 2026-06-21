// SPDX-License-Identifier: GPL-2.0-only

//go:build !linux

package benchprof

// readPeakRSS returns 0 on platforms without a /proc-style peak-RSS source; the memory
// summary still reports Go heap/GC stats, which are portable.
func readPeakRSS() uint64 { return 0 }
