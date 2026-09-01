// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/exchange"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// syntheticDWG writes a small DWG (a rectangle sketch exported through the DWG writer) to a temp
// file and returns its path, so an import test runs on CI without the git-ignored corpus.
func syntheticDWG(t *testing.T) string {
	t.Helper()
	src := compdef.NewPartComponentDefinition()
	sk := src.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(0, 0))
	c1 := sk.Points().Add(math.P2(40, 0))
	c2 := sk.Points().Add(math.P2(40, 30))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	path := filepath.Join(t.TempDir(), "synthetic.dwg")
	if err := exchange.ExportDWGFile(sk, path, param.DefaultUnitsOfMeasure()); err != nil {
		t.Fatalf("ExportDWGFile: %v", err)
	}
	return path
}

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
	t.Parallel()
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
	t.Parallel()
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

// TestSessionImportDWGIsOneUndoStep: importing a file must register as a SINGLE operation in the
// undo stream — the thousands of per-entity sketch additions it makes internally are not
// individual undo steps. So one Import records exactly one event, and undo removes the whole
// imported sketch at once (then redo brings it back). This also exercises the fast undo snapshot
// codec on a real large import.
func TestSessionImportDWGIsOneUndoStep(t *testing.T) {
	t.Parallel()
	s := sessionWithPart(t)
	choices, err := s.DWGPlaneChoices()
	if err != nil {
		t.Fatal(err)
	}
	part, err := activePart(s)
	if err != nil {
		t.Fatal(err)
	}
	before := len(s.undoLabels())

	res, err := s.ImportDWGFile(syntheticDWG(t), choices[0].Plane)
	if err != nil {
		t.Fatalf("ImportDWGFile: %v", err)
	}
	labels := s.undoLabels()
	if len(labels) != before+1 {
		t.Fatalf("import recorded %d undo steps, want exactly 1 (labels: %v)", len(labels)-before, labels[before:])
	}
	if got := labels[len(labels)-1]; got == "" {
		t.Errorf("import undo step has no label")
	}
	if part.Sketches().Count()+part.Sketches3D().Count() == 0 {
		t.Fatalf("import added no sketch (entityCount=%d)", res.EntityCount)
	}

	// One undo removes the entire import.
	if err := s.Undo(); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if n := part.Sketches().Count() + part.Sketches3D().Count(); n != 0 {
		t.Errorf("after one undo: %d sketches remain, want 0 (import was not one atomic step)", n)
	}
	// Redo restores it.
	if err := s.Redo(); err != nil {
		t.Fatalf("Redo: %v", err)
	}
	if n := part.Sketches().Count() + part.Sketches3D().Count(); n == 0 {
		t.Errorf("after redo: import not restored")
	}
}
