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

// TestMachineUUIDResolvesUnderHome exercises the default (non-env) path: with no
// OBK_USER_GLOBALS_FILE override, the globals file resolves under the user's home/AppData via
// userpaths, and the directory is created on first write.
func TestMachineUUIDResolvesUnderHome(t *testing.T) {
	t.Setenv("OBK_USER_GLOBALS_FILE", "")
	home := t.TempDir()
	t.Setenv("HOME", home)            // Unix
	t.Setenv("AppData", home)         // Windows (os.UserConfigDir)
	t.Setenv("XDG_CONFIG_HOME", home) // some Linux UserConfigDir paths

	id, err := MachineUUID()
	if err != nil {
		t.Fatalf("MachineUUID: %v", err)
	}
	if id == "" {
		t.Fatal("MachineUUID returned empty under home resolution")
	}
	// The globals file must have been created under <home>/oblikovati.
	path, err := globalsPath()
	if err != nil {
		t.Fatalf("globalsPath: %v", err)
	}
	if !strings.Contains(path, "oblikovati") {
		t.Errorf("globals path %q is not under the oblikovati home", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("globals file not created at %q: %v", path, err)
	}
}

func TestReadGlobalsValueAbsentFileAndKey(t *testing.T) {
	if v := readGlobalsValue(filepath.Join(t.TempDir(), "nope"), "machineUUID"); v != "" {
		t.Errorf("missing file gave %q, want empty", v)
	}
	path := filepath.Join(t.TempDir(), "globals")
	if err := writeGlobalsValue(path, "other", "123"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if v := readGlobalsValue(path, "machineUUID"); v != "" {
		t.Errorf("absent key gave %q, want empty", v)
	}
	if v := readGlobalsValue(path, "other"); v != "123" {
		t.Errorf("present key gave %q, want 123", v)
	}
}
