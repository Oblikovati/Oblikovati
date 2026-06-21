// SPDX-License-Identifier: GPL-2.0-only

package benchprof

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStartStopDisabledWritesNothing(t *testing.T) {
	t.Setenv(envDir, "")
	run, err := Start("disabled")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	// Allocate so the summary has something to report even with profiling off.
	sink := make([]byte, 1<<20)
	_ = sink
	summary, err := run.Stop()
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if summary.Label != "disabled" {
		t.Errorf("label = %q, want disabled", summary.Label)
	}
	if Enabled() {
		t.Error("Enabled should be false when OBK_PPROF_DIR is empty")
	}
}

func TestStartStopWritesProfiles(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envDir, dir)
	if !Enabled() {
		t.Fatal("Enabled should be true when OBK_PPROF_DIR is set")
	}
	run, err := Start("flatten")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	summary, err := run.Stop()
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	for _, name := range []string{"flatten.cpu.pprof", "flatten.heap.pprof", "flatten.mem.txt"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("expected %s: %v", name, err)
		}
	}
	if !strings.Contains(summary.String(), "[flatten]") {
		t.Errorf("summary missing label: %q", summary.String())
	}
}

func TestStartRejectsEmptyLabel(t *testing.T) {
	if _, err := Start(""); err == nil {
		t.Error("expected error for empty label")
	}
}
