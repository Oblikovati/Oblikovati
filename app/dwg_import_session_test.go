// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"os"
	"path/filepath"
	"testing"
)

func corpusDWG(t *testing.T, name string) string {
	t.Helper()
	dir := os.Getenv("DWG_TESTFILES_DIR")
	if dir == "" {
		dir = filepath.Join("..", "..", "experiments", "dwg-reverse-engineering")
	}
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err != nil {
		t.Skipf("corpus file %s unavailable: %v", name, err)
	}
	return path
}

// TestDWGPlaneChoices lists the planes a 2D DWG import can target: at least the
// three origin planes of the active part.
func TestDWGPlaneChoices(t *testing.T) {
	s := sessionWithPart(t)
	choices, err := s.DWGPlaneChoices()
	if err != nil {
		t.Fatalf("DWGPlaneChoices: %v", err)
	}
	if len(choices) < 3 {
		t.Fatalf("got %d plane choices, want at least the 3 origin planes", len(choices))
	}
	for _, c := range choices {
		if c.Name == "" {
			t.Errorf("plane choice has empty name")
		}
	}
}

// TestSessionImportDWGPlanar imports a real planar drawing onto the first plane and
// checks it lands as a populated 2D sketch (undoable edit).
func TestSessionImportDWGPlanar(t *testing.T) {
	s := sessionWithPart(t)
	choices, err := s.DWGPlaneChoices()
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.ImportDWGFile(corpusDWG(t, "testfile-7.dwg"), choices[0].Plane)
	if err != nil {
		t.Fatalf("ImportDWGFile: %v", err)
	}
	if res.Is3D {
		t.Errorf("planar drawing imported as 3D")
	}
	if res.EntityCount < 1000 {
		t.Errorf("imported only %d entities", res.EntityCount)
	}
}
