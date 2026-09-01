// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// dimensionedSketchSession returns a session editing a sketch with one fully-dimensioned
// horizontal line a–b (length d0), centred so sketch point (x,0) is at screen centre. The
// line is not MoveableFree, so a normal drag refuses it — the Relax Mode fixture.
func dimensionedSketchSession(t *testing.T, length float64) (*Session, *sketch.Sketch, *sketch.Point) {
	t.Helper()
	s, _ := emptyPartSession(t)
	sk, err := s.CreateSketch(sketch.XYPlane())
	if err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	s.Grid().SnapToGrid = false
	a := sk.Points().Add(math.P2(0, 0))
	b := sk.Points().Add(math.P2(length, 0))
	sk.GeometricConstraints().AddFix(a) // ground one end so b is fully determined, not free
	sk.GeometricConstraints().AddHorizontal(a, b)
	if _, err := sk.DimensionConstraints().AddDistance(a, b, "3 cm"); err != nil {
		t.Fatalf("AddDistance: %v", err)
	}
	return s, sk, b
}

func TestRelaxModeTogglePersistsAndReports(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if s.RelaxMode() {
		t.Fatal("Relax Mode should be off by default")
	}
	if got := s.ToggleRelaxMode(); !got || !s.RelaxMode() {
		t.Fatalf("ToggleRelaxMode → %v, RelaxMode %v, want both true", got, s.RelaxMode())
	}
	s.SetRelaxMode(false)
	if s.RelaxMode() {
		t.Error("SetRelaxMode(false) did not turn it off")
	}
}

// TestRelaxModeRefusedDragWithoutMode confirms the baseline: a fully-dimensioned line cannot
// be direct-dragged when Relax Mode is off (it click-selects instead).
func TestRelaxModeRefusedDragWithoutMode(t *testing.T) {
	t.Parallel()
	s, _, b := dimensionedSketchSession(t, 3)
	centerOnSketchPoint(s, 3) // endpoint b (3,0) at screen centre
	if s.BeginEntityDrag(100, 100) {
		t.Error("a fully-dimensioned endpoint must not drag with Relax Mode off")
	}
	_ = b
}

// TestRelaxModeDragsDimensionedLine confirms that with Relax Mode on the same fully-dimensioned
// endpoint drags, the geometry follows the cursor, and the driving dimension relaxes to it.
func TestRelaxModeDragsDimensionedLine(t *testing.T) {
	t.Parallel()
	s, sk, b := dimensionedSketchSession(t, 3)
	s.SetRelaxMode(true)
	centerOnSketchPoint(s, 3) // endpoint b (3,0) at screen centre

	if !s.BeginEntityDrag(100, 100) {
		t.Fatal("Relax Mode should allow dragging the fully-dimensioned endpoint")
	}
	s.UpdateEntityDrag(140, 100) // drag +x in the sketch
	s.CommitEntityDrag()

	if b.Position().X <= 3.0 {
		t.Errorf("relaxed endpoint did not follow the cursor in +x: %v", b.Position())
	}
	// The driving dimension relaxed to the new length (its residual is ~0).
	d := sk.DimensionConstraints().Item(0)
	for _, r := range d.Residuals() {
		if r > 1e-6 || r < -1e-6 {
			t.Errorf("dimension not relaxed after drag: residual %v", r)
		}
	}
}
