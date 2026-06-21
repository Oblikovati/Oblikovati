// SPDX-License-Identifier: GPL-2.0-only

package usagestats

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// useTempGlobals points the globals file at a fresh temp path for the test.
func useTempGlobals(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "globals")
	t.Setenv("OBK_USER_GLOBALS_FILE", path)
	return path
}

func TestMachineUUIDGeneratesThenReuses(t *testing.T) {
	path := useTempGlobals(t)
	first, err := MachineUUID()
	if err != nil {
		t.Fatalf("first MachineUUID: %v", err)
	}
	if first == "" {
		t.Fatal("MachineUUID returned empty")
	}
	// The file must now persist the id under the machineUUID key.
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read globals: %v", err)
	}
	if !strings.Contains(string(b), "machineUUID="+first) {
		t.Errorf("globals file %q does not persist the id", b)
	}
	// A second call returns the SAME id (read back, not regenerated).
	second, err := MachineUUID()
	if err != nil {
		t.Fatalf("second MachineUUID: %v", err)
	}
	if second != first {
		t.Errorf("MachineUUID not stable: %q then %q", first, second)
	}
}

func TestMachineUUIDIsV4Shaped(t *testing.T) {
	useTempGlobals(t)
	id, err := MachineUUID()
	if err != nil {
		t.Fatalf("MachineUUID: %v", err)
	}
	// 8-4-4-4-12 hex with version nibble 4.
	parts := strings.Split(id, "-")
	if len(parts) != 5 || len(parts[0]) != 8 || len(parts[2]) != 4 || parts[2][0] != '4' {
		t.Errorf("id %q is not a v4-shaped UUID", id)
	}
}

func TestMachineUUIDDistinctAcrossFiles(t *testing.T) {
	useTempGlobals(t)
	a, _ := MachineUUID()
	useTempGlobals(t) // a different temp file
	b, _ := MachineUUID()
	if a == b {
		t.Errorf("two fresh globals files produced the same id %q", a)
	}
}
