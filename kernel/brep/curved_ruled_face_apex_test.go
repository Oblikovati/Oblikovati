// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A cone face bounded by its own APEX is a wall the loop-framed chart carries (ADR-0062). The
// recogniser used to refuse it, which sent every apex cone to the mixed boolean's pass bucket — whose
// gate cannot prove a cone clear of a crossing plane — so the whole boolean declined and the
// half-space cut kept a pipeline of its own to serve them.

// apexConeSideFace is the side face of a full cone: apex at z=10, base circle of radius 3 at z=0.
func apexConeSideFace(t *testing.T) curvedFace {
	t.Helper()
	body, err := SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 0, "apexcone")
	if err != nil {
		t.Fatalf("SolidCylinderCone: %v", err)
	}
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(geom.Cone); ok {
			return curvedFaceOf(f)
		}
	}
	t.Fatal("the cone body has no cone face")
	return curvedFace{}
}

// TestRuledFaceTakesAConeBoundedByItsApex: the recogniser admits it, and the window reaches the apex
// through the face's own ruling — the apex is no frame edge, but the ruling out to it is.
func TestRuledFaceTakesAConeBoundedByItsApex(t *testing.T) {
	t.Parallel()
	rs, ok := ruledFaceOf(apexConeSideFace(t))
	if !ok {
		t.Fatal("a cone bounded by its apex was refused as a ruled wall")
	}
	if rs.band.vMin != 0 {
		t.Errorf("the window starts at v=%g, want the apex at v=0", rs.band.vMin)
	}
	if stdmath.Abs(rs.band.vMax-10) > 1e-9 {
		t.Errorf("the window ends at v=%g, want the base rim at v=10", rs.band.vMax)
	}
}

// TestRuledFaceRefusesAConeStraddlingItsApex: the radius changes sign across the apex, so the two
// nappes are two surfaces and one chart cannot carry both.
func TestRuledFaceRefusesAConeStraddlingItsApex(t *testing.T) {
	t.Parallel()
	frame, ok := geom.RuledFrameOf(apexConeSideFace(t).surface)
	if !ok {
		t.Fatal("the cone face has no ruled frame")
	}
	if _, _, ok := apexBoundedWindow(frame, -4, 6); ok {
		t.Error("a window straddling the apex was admitted; the two nappes are two surfaces")
	}
	if _, _, ok := apexBoundedWindow(frame, 1, 6); !ok {
		t.Error("a window clear of the apex was refused")
	}
}

// TestApexChartClosesAtTheApexAndStopsThere: the chart closes its parameter rectangle at the apex with
// a DEGENERATE segment — the whole azimuth there is one point — and its artificial seam gets no
// overrun past it, because past the apex the surface folds onto its other nappe and a seam crossing
// found there would be a crossing on geometry the face does not own.
func TestApexChartClosesAtTheApexAndStopsThere(t *testing.T) {
	t.Parallel()
	f := apexConeSideFace(t)
	rs, ok := ruledFaceOf(f)
	if !ok {
		t.Fatal("apex cone not recognised")
	}
	c := newRuledFaceUV(f, rs, Difference, false, func(math.Point3) bool { return false })
	if _, atApex := c.boundedByApex(); !atApex {
		t.Fatal("the chart does not know its window ends at the apex")
	}
	if got := c.seamOverrun(); got != 0 {
		t.Errorf("seam overrun = %g past an apex, want 0: the overrun lands on the other nappe", got)
	}
	segs := c.apexSegments()
	if len(segs) != 1 {
		t.Fatalf("apexSegments returned %d segments, want exactly one closing the rectangle", len(segs))
	}
	s := segs[0]
	apexV, _ := c.boundedByApex()
	if float64(s.a.Y) != apexV || float64(s.b.Y) != apexV || float64(s.b.X-s.a.X) != 2*stdmath.Pi {
		t.Errorf("the apex segment spans (%v → %v), want the whole azimuth at v=%g", s.a, s.b, apexV)
	}
	lo, hi := s.curve.Domain()
	if d := float64(s.curve.PointAt(lo).DistanceTo(s.curve.PointAt(hi))); d > 1e-12 {
		t.Errorf("the apex segment's curve is %g long, want a degenerate point", d)
	}
}

// TestApexChartDropsTheDegenerateApexEdge: an apex edge is a boundary in parameter space and a point in
// space. Left in a face's loop it is a zero-length edge with one use, and the body reads as open — the
// same thing OCCT's degenerate edges are, and skipped for the same reason.
func TestApexChartDropsTheDegenerateApexEdge(t *testing.T) {
	t.Parallel()
	f := apexConeSideFace(t)
	rs, ok := ruledFaceOf(f)
	if !ok {
		t.Fatal("apex cone not recognised")
	}
	c := newRuledFaceUV(f, rs, Difference, false, func(math.Point3) bool { return false })
	apex := c.point3(0, 0)
	rim := c.point3(0, 10)
	loops := []curvedLoop{{edges: []loopEdge{
		{curve: geom.NewLineSegment(apex, rim), t0: 0, t1: 1},
		{curve: geom.NewLineSegment(apex, apex), t0: 0, t1: 1}, // the apex
	}}}
	// The drop is one rule for every chart, applied where the faces are built (curved_uv_side.go), so
	// this exercises it directly rather than through a chart that no longer carries a copy.
	out := dropDegenerateEdges(c.finalizeLoops(loops), geom.ResolutionForBox(faceLoopBox(f)))
	if len(out) != 1 || len(out[0].edges) != 1 {
		t.Fatalf("the degenerate drop kept %v, want the ruling alone", out)
	}
	if d := float64(out[0].edges[0].start().DistanceTo(out[0].edges[0].end())); d < 1 {
		t.Errorf("the surviving edge is %g long: the ruling was dropped instead of the apex", d)
	}
}
