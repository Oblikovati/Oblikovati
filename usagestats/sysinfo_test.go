// SPDX-License-Identifier: GPL-2.0-only

package usagestats

import "testing"

func TestGatherHardwareReportsCores(t *testing.T) {
	hw := GatherHardware()
	// CPUCores comes from the Go runtime and is always available.
	if hw.CPUCores < 1 {
		t.Errorf("CPUCores = %d, want >= 1", hw.CPUCores)
	}
	// The probes are best-effort and may legitimately return zero in a sandbox, but they must
	// never be negative.
	if hw.RAMBytes < 0 || hw.StorageBytes < 0 {
		t.Errorf("negative hardware sizes: %+v", hw)
	}
}

func TestProbeDirNonEmpty(t *testing.T) {
	if probeDir() == "" {
		t.Fatal("probeDir returned empty; expected a home or temp path")
	}
}
