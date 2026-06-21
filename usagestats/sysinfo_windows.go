// SPDX-License-Identifier: GPL-2.0-only

//go:build windows

package usagestats

import (
	"os"
	"syscall"
	"unsafe"
)

// memoryStatusEx mirrors the Win32 MEMORYSTATUSEX struct passed to GlobalMemoryStatusEx.
type memoryStatusEx struct {
	length               uint32
	memoryLoad           uint32
	totalPhys            uint64
	availPhys            uint64
	totalPageFile        uint64
	availPageFile        uint64
	totalVirtual         uint64
	availVirtual         uint64
	availExtendedVirtual uint64
}

var kernel32 = syscall.NewLazyDLL("kernel32.dll")

// totalRAMBytes reads ullTotalPhys via GlobalMemoryStatusEx. Returns 0 on error.
func totalRAMBytes() int64 {
	var m memoryStatusEx
	m.length = uint32(unsafe.Sizeof(m))
	ret, _, _ := kernel32.NewProc("GlobalMemoryStatusEx").Call(uintptr(unsafe.Pointer(&m)))
	if ret == 0 {
		return 0
	}
	return int64(m.totalPhys)
}

// cpuModel returns the processor description from the environment. Windows exposes
// PROCESSOR_IDENTIFIER (e.g. "Intel64 Family 6 Model 142 Stepping 10, GenuineIntel"); there
// is no admin-free brand string, so this identifier stands in. Returns "" if unset.
func cpuModel() string {
	return os.Getenv("PROCESSOR_IDENTIFIER")
}

// homeVolumeCapacityBytes returns the total size of the volume holding the probe dir via
// GetDiskFreeSpaceExW. Returns 0 on error.
func homeVolumeCapacityBytes() int64 {
	dir, err := syscall.UTF16PtrFromString(probeDir())
	if err != nil {
		return 0
	}
	var freeAvail, totalBytes, totalFree uint64
	ret, _, _ := kernel32.NewProc("GetDiskFreeSpaceExW").Call(
		uintptr(unsafe.Pointer(dir)),
		uintptr(unsafe.Pointer(&freeAvail)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFree)),
	)
	if ret == 0 {
		return 0
	}
	return int64(totalBytes)
}
