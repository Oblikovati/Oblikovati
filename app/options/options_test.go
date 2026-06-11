// SPDX-License-Identifier: GPL-2.0-only

package options

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/api/types"
)

func TestLoadMissingFileIsDefaults(t *testing.T) {
	s := NewFileStore(filepath.Join(t.TempDir(), "options.yaml"))
	all, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if all != Defaults() {
		t.Errorf("Load(missing) = %+v, want defaults %+v", all, Defaults())
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "options.yaml")
	s := NewFileStore(path)
	in := Defaults()
	in.General.StartupAction = types.StartupEmptyWorkspace
	in.Sketch.GridSpacingCm = 0.5
	in.Sketch.SnapToGrid = false
	in.Part.ChamferFlatCorners = false
	if err := s.Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	out, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out != in {
		t.Errorf("round trip = %+v, want %+v", out, in)
	}
}

func TestLoadKeepsDefaultsForAbsentKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.yaml")
	// An old file knowing only the sketch group: everything else keeps its default.
	yaml := "sketch:\n  gridSpacingCm: 2\n  gridVisible: true\n  gridMajorEvery: 5\n  snapToPoints: true\n  snapToGrid: true\n"
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := NewFileStore(path).Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.Sketch.GridSpacingCm != 2 {
		t.Errorf("sketch spacing = %v, want 2 (from file)", out.Sketch.GridSpacingCm)
	}
	if !out.Part.ChamferFlatCorners || out.General.StartupAction != types.StartupNewPart {
		t.Errorf("absent groups = %+v, want defaults kept", out)
	}
}
