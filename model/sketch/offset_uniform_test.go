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
	for i := 0; i < 4; i++ {
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
