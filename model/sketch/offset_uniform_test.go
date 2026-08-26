// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	gmath "oblikovati.org/math"
)

// groundedSquare builds a CCW square [0,side]^2, grounds every edge (a fixed source profile), and
// returns its four lines.
func groundedSquare(s *Sketch, side float64) []*Line {
	c := []gmath.Point2{gmath.P2(0, 0), gmath.P2(gmath.Scalar(side), 0), gmath.P2(gmath.Scalar(side), gmath.Scalar(side)), gmath.P2(0, gmath.Scalar(side))}
	lines := make([]*Line, 4)
	for i := range 4 {
		lines[i] = s.Lines().AddByTwoPoints(c[i], c[(i+1)%4])
		s.GeometricConstraints().AddGround(lines[i])
	}
	return lines
}

// assertUniformOffset fails unless every offset line sits at perpendicular distance want (±1e-4) from
// its source — i.e. the offset stayed uniform.
func assertUniformOffset(t *testing.T, sources []*Line, offsets []Entity, want float64) {
	t.Helper()
	for i, off := range offsets {
		ol := wantLine(t, off)
		d := stdmath.Abs(perpDistanceToLine(sources[i].A.Position(), sources[i].B.Position(), ol.A.Position()))
		if stdmath.Abs(d-want) > 1e-4 {
			t.Errorf("offset segment %d at distance %.5f, want %.5f (offset is not uniform)", i, d, want)
		}
	}
}

// TestOffsetLoopUniformlyDriven is the Inventor-parity core: a constrained loop offset carries ONE
// driving dimension, and editing that single dimension moves the WHOLE loop uniformly — every segment
// follows, because the non-dimensioned lines are bound to the same distance by driven offset
// constraints. The sketch stays exactly determined (converges, not over-constrained).
func TestOffsetLoopUniformlyDriven(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	lines := groundedSquare(s, 4)
	path := closedPath(lines[0], lines[1], lines[2], lines[3])

	offsets, err := s.OffsetConnectedLoop(path, 0.5) // inner offset by 0.5
	if err != nil {
		t.Fatalf("OffsetConnectedLoop: %v", err)
	}
	dim := s.ConstrainOffsetLoopUniform(path, offsets)
	if dim == nil {
		t.Fatal("no driving offset dimension was created")
	}
	if r := s.Solve(); !r.Converged || r.Status == OverConstrained {
		t.Fatalf("solve after uniform offset: converged=%v status=%v (want converged, not over-constrained)", r.Converged, r.Status)
	}
	assertUniformOffset(t, lines, offsets, 0.5)

	// Edit the ONE dimension — the whole loop must follow to 1.0 uniformly.
	if err := dim.Drive(1.0); err != nil {
		t.Fatalf("Drive(1.0): %v", err)
	}
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve after driving the dimension to 1.0: converged=%v", r.Converged)
	}
	assertUniformOffset(t, lines, offsets, 1.0)
}

// TestOffsetLoopUniformSurvivesRoundTrip is the serialization guarantee: after a uniform-offset loop
// is saved and reloaded, editing its single driving dimension STILL moves every segment uniformly. The
// driven links are re-bound to the restored dimension parameter (applyPendingOffsetDrives), so the
// loop does not freeze at its last distance on reload (the former closure-only limitation).
func TestOffsetLoopUniformSurvivesRoundTrip(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	lines := groundedSquare(s, 4)
	path := closedPath(lines[0], lines[1], lines[2], lines[3])
	offsets, err := s.OffsetConnectedLoop(path, 0.5)
	if err != nil {
		t.Fatalf("OffsetConnectedLoop: %v", err)
	}
	dim := s.ConstrainOffsetLoopUniform(path, offsets)
	if dim == nil {
		t.Fatal("no driving offset dimension was created")
	}
	name := dim.Parameter().Name()
	s.Solve()

	data, err := sc.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	sc2 := NewSketches()
	if err := sc2.ApplyRecipe(data); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	s2 := sc2.Item(0)

	if got := countDrivenOffsets(s2); got != 3 {
		t.Fatalf("restored driven offset constraints = %d, want 3 (loop of 4 lines, one carries the dimension)", got)
	}
	drv := dimensionNamed(t, s2, name)
	if err := drv.Drive(1.0); err != nil {
		t.Fatalf("Drive(1.0) after reload: %v", err)
	}
	if r := s2.Solve(); !r.Converged {
		t.Fatalf("solve after reload+drive: converged=%v", r.Converged)
	}
	src := []*Line{s2.Lines().Item(0), s2.Lines().Item(1), s2.Lines().Item(2), s2.Lines().Item(3)}
	off := []Entity{s2.Lines().Item(4), s2.Lines().Item(5), s2.Lines().Item(6), s2.Lines().Item(7)}
	assertUniformOffset(t, src, off, 1.0)
}

// countDrivenOffsets counts the driven (dimension-linked) offset constraints in a sketch.
func countDrivenOffsets(s *Sketch) int {
	n := 0
	for _, c := range s.Constraints() {
		if oc, ok := c.(*OffsetConstraint); ok && oc.driver != nil {
			n++
		}
	}
	return n
}

// dimensionNamed returns the dimension whose parameter has the given name.
func dimensionNamed(t *testing.T, s *Sketch, name string) *DimensionConstraint {
	t.Helper()
	for _, d := range s.DimensionConstraints().All() {
		if d.Parameter() != nil && d.Parameter().Name() == name {
			return d
		}
	}
	t.Fatalf("no dimension with parameter %q after reload", name)
	return nil
}

// TestOffsetLoopRoundedArcsDriveUniformly proves a ROUNDED loop (a stadium: two lines + two semicircle
// caps) offsets uniformly under one driving dimension — every offset arc's radius grows with the line
// distance (concentric arcs are carried by the corner joins), not just the lines. The stadium's arcs
// make the solve redundant (an arc's built-in circularity is redundant with a fully pinned frame — a
// grounded arc alone reports the same), so the assertion is convergence + uniform geometry, not a
// well-constrained verdict.
func TestOffsetLoopRoundedArcsDriveUniformly(t *testing.T) {
	const L, r = 4.0, 1.0
	s := NewSketches().Add(XYPlane())
	bottom := s.Lines().AddByTwoPoints(gmath.P2(0, -r), gmath.P2(L, -r))
	right := s.Arcs().AddByCenterStartEnd(gmath.P2(L, 0), gmath.P2(L, -r), gmath.P2(L, r), true)
	top := s.Lines().AddByTwoPoints(gmath.P2(L, r), gmath.P2(0, r))
	left := s.Arcs().AddByCenterStartEnd(gmath.P2(0, 0), gmath.P2(0, r), gmath.P2(0, -r), true)
	for _, e := range []Entity{bottom, right, top, left} {
		s.GeometricConstraints().AddGround(e)
	}
	path := closedPath(bottom, right, top, left)

	offsets, err := s.OffsetConnectedLoop(path, -0.3) // outward offset by 0.3
	if err != nil {
		t.Fatalf("OffsetConnectedLoop: %v", err)
	}
	dim := s.ConstrainOffsetLoopUniform(path, offsets)
	if dim == nil {
		t.Fatal("no driving offset dimension was created")
	}
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve after uniform offset: converged=%v", r.Converged)
	}
	rightArc := wantArc(t, offsets[1])
	leftArc := wantArc(t, offsets[3])
	assertRadius(t, rightArc, 1.3)
	assertRadius(t, leftArc, 1.3)

	if err := dim.Drive(0.6); err != nil { // grow the offset to 0.6
		t.Fatalf("Drive(0.6): %v", err)
	}
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve after driving the dimension to 0.6: converged=%v", r.Converged)
	}
	assertRadius(t, rightArc, 1.6) // both caps grow uniformly with the one dimension
	assertRadius(t, leftArc, 1.6)
}

// assertRadius fails unless the arc's radius is within 1e-3 of want.
func assertRadius(t *testing.T, a *Arc, want float64) {
	t.Helper()
	if got := float64(a.Radius()); stdmath.Abs(got-want) > 1e-3 {
		t.Errorf("arc radius = %.4f, want %.4f", got, want)
	}
}
