// SPDX-License-Identifier: GPL-2.0-only

package usagestats

import (
	"os"
	"runtime"
)

// Hardware is the machine's detected hardware profile for a [Snapshot]. Every field is
// best-effort: a value the host cannot read on this platform is left zero/empty rather than
// failing, so telemetry never blocks startup.
type Hardware struct {
	RAMBytes     int64
	CPU          string
	CPUCores     int
	StorageBytes int64
}

// GatherHardware probes this machine's hardware. CPUCores is the one always-available field
// (the Go runtime); RAM, the CPU model, and home-volume capacity come from platform-specific
// probes (see the sysinfo_<os>.go files) and degrade to zero/"" when undetectable.
//
// Example:
//
//	hw := usagestats.GatherHardware()
func GatherHardware() Hardware {
	return Hardware{
		RAMBytes:     totalRAMBytes(),
		CPU:          cpuModel(),
		CPUCores:     runtime.NumCPU(),
		StorageBytes: homeVolumeCapacityBytes(),
	}
}

// probeDir is the path whose containing volume's capacity stands in for the machine's
// storage: the user's home (where ~/oblikovati lives), falling back to the temp dir.
func probeDir() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return home
	}
	return os.TempDir()
}
