// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// stubCustomSolver is a named fake CustomConstraintSolver: it pulls the two world points
// together (residual = pA − pB) so a custom constraint behaves like a point coincidence,
// letting the test prove dispatch reaches the add-in solver.
type stubCustomSolver struct{ calls int }

func (s *stubCustomSolver) Residuals(_ string, _ []float64, a, b math.Point3) []float64 {
	s.calls++
	d := b.VectorTo(a)
	return []float64{d.X, d.Y, d.Z}
}

// TestMotionConstraintsHoldNoStaticDOF checks the gear/rack/translate couplings contribute
// no static residual: the free component keeps all six DOF (they couple rates at drive
// time, F03), while their parameters round-trip.
func TestMotionConstraintsHoldNoStaticDOF(t *testing.T) {
	occs := occurrence.NewOccurrences()
	a := place(occs, "a:1", math.Identity4())
	a.SetGrounded(true)
	b := place(occs, "b:1", math.Identity4())
	set := NewConstraintSet(occs, nil)

	axis := LinePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))
	rr := set.AddRotateRotate(a, axis, b, axis, 2.5)
	rtc := set.AddRotateTranslate(a, axis, b, axis, 6.28)
	tt := set.AddTranslateTranslate(a, axis, b, axis, 0.5)
	rep := set.Solve()

	if got := dofOf(rep, b.ID()); got != 6 {
		t.Errorf("free component DOF with motion couplings = %d, want 6", got)
	}
	if rr.Ratio() != 2.5 || rtc.Distance() != 6.28 || tt.Ratio() != 0.5 {
		t.Errorf("motion params = %v/%v/%v, want 2.5/6.28/0.5", rr.Ratio(), rtc.Distance(), tt.Ratio())
	}
}

// TestTransitionalKeepsContact checks a transitional constraint pulls the moving point onto
// the transition plane (one contact residual, removing one DOF).
func TestTransitionalKeepsContact(t *testing.T) {
	occs := occurrence.NewOccurrences()
	base := place(occs, "base:1", math.Identity4())
	base.SetGrounded(true)
	moving := place(occs, "moving:1", math.Translation4(math.V3(0, 0, 8)))

	set := NewConstraintSet(occs, nil)
	plane := PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1)) // z = 0 transition plane
	rider := PointPrimitive(math.P3(0, 0, 0))
	set.AddTransitional(moving, rider, base, plane)
	rep := set.Solve()

	if z := moving.Transform().Translation().Z; z > 1e-6 || z < -1e-6 {
		t.Errorf("rider z = %v, want 0 (on the plane)", z)
	}
	if got := dofOf(rep, moving.ID()); got != 5 {
		t.Errorf("transitional DOF = %d, want 5", got)
	}
}

// TestCustomConstraintDispatches checks a custom constraint reaches the installed add-in
// solver and applies its residual (here a point coincidence).
func TestCustomConstraintDispatches(t *testing.T) {
	occs := occurrence.NewOccurrences()
	base := place(occs, "base:1", math.Identity4())
	base.SetGrounded(true)
	moving := place(occs, "moving:1", math.Translation4(math.V3(4, 4, 4)))

	stub := &stubCustomSolver{}
	set := NewConstraintSet(occs, nil)
	set.UseCustomSolver(stub)
	set.AddCustom(base, PointPrimitive(math.P3(1, 1, 1)), moving, PointPrimitive(math.P3(0, 0, 0)), "weld", []float64{1})
	set.Solve()

	if stub.calls == 0 {
		t.Fatal("custom solver was never called")
	}
	if got := moving.Transform().TransformPoint(math.P3(0, 0, 0)); !got.IsEqualTo(math.P3(1, 1, 1), 1e-6) {
		t.Errorf("custom-welded point = %+v, want (1,1,1)", got)
	}
}

// TestCustomConstraintInertWithoutSolver checks a custom constraint with no solver
// installed contributes no residual (held inert, not an error).
func TestCustomConstraintInertWithoutSolver(t *testing.T) {
	occs := occurrence.NewOccurrences()
	base := place(occs, "base:1", math.Identity4())
	base.SetGrounded(true)
	moving := place(occs, "moving:1", math.Identity4())

	set := NewConstraintSet(occs, nil)
	c := set.AddCustom(base, PointPrimitive(math.P3(0, 0, 0)), moving, PointPrimitive(math.P3(0, 0, 0)), "weld", nil)
	rep := set.Solve()

	if got := dofOf(rep, moving.ID()); got != 6 {
		t.Errorf("DOF with inert custom constraint = %d, want 6", got)
	}
	if c.Kind() != "weld" {
		t.Errorf("Kind() = %q, want weld", c.Kind())
	}
}
