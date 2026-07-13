// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	m "oblikovati.org/math"
)

// ellipseHoleSamplesT6 is the T6 hole-loop discretization used by both nodesForT6 (crossing
// detection) and ellipseHoleT6 (the filletLoop) — they MUST share sample count/phase so a
// crossing.I computed on one indexes the other's pts directly.
const ellipseHoleSamplesT6 = 720

// recededTopOuterT6 is the T6 fillet's receded top-face outer loop: rectangle x[-20,20]
// y[-7,12], wound CCW (+Z outward normal, shoelace area +760) with the front edge
// (pts[0]->pts[1]) along the receded boundary y=-7 — the OCCT result_8 precondition.
func recededTopOuterT6() filletLoop {
	var loop filletLoop
	loop.add(m.P3(-20, -7, 0), nil)
	loop.add(m.P3(20, -7, 0), nil)
	loop.add(m.P3(20, 12, 0), nil)
	loop.add(m.P3(-20, 12, 0), nil)
	return loop
}

// ellipseHoleT6 is the T6 obstacle base rim: the full a=15,b=10 ellipse in the z=0 host plane,
// sampled into ellipseHoleSamplesT6 segments each carrying its OWN geom.EllipticalArc (not a
// shared full-period curve) so a segment's curve endpoints match its own two polyline points —
// the same "per-segment arc" shape the fillet rebuild uses elsewhere (fillet_orient.go).
func ellipseHoleT6() filletLoop {
	pts2 := sampleEllipse(ellipseHoleSamplesT6)
	var loop filletLoop
	for i, p := range pts2 {
		loop.add(m.P3(p.X, p.Y, 0), ellipseArcSegmentT6(i, ellipseHoleSamplesT6))
	}
	return loop
}

// ellipseArcSegmentT6 is the exact elliptical arc for sample segment i of n around the T6
// ellipse (center origin, major axis +X, a=15, b=10) — PointAt(0)/PointAt(1) land on sample i
// and i+1 exactly, matching geom.EllipseFull's t->angle convention (kernel/geom/ellipse.go).
func ellipseArcSegmentT6(i, n int) geom.Curve3 {
	start := 2 * stdmath.Pi * float64(i) / float64(n)
	sweep := 2 * stdmath.Pi / float64(n)
	arc, _ := geom.NewEllipticalArc(m.P3(0, 0, 0), m.V3(0, 0, 1), m.V3(1, 0, 0), 15, 10, start, sweep)
	return arc
}

// nodesForT6 returns T6's two boundary crossings (~(-10.712142,-7) and ~(10.712142,-7)) via the
// Task 2 detector, sampled on the SAME phase/count as ellipseHoleT6 so crossing.I indexes it.
func nodesForT6(t *testing.T) [2]crossing {
	t.Helper()
	boundary := boundaryLine2{origin: m.P2(0, -7), dir: m.V2(1, 0)}
	nodes, ok := obstacleNodes(sampleEllipse(ellipseHoleSamplesT6), boundary, ResolutionForSize(50))
	if !ok {
		t.Fatal("expected two T6 boundary crossings")
	}
	return nodes
}

// zPlaneProjector is the flat/back pair for the T6 host plane z=0 — planeProjector's own
// inverse for a +Z-normal plane (earclip.go: a +Z normal drops Z, keeps (X,Y) in order).
func zPlaneProjector() (func(m.Point3) m.Point2, func(m.Point2) m.Point3) {
	flat := planeProjector(m.V3(0, 0, 1))
	back := func(p m.Point2) m.Point3 { return m.P3(p.X, p.Y, 0) }
	return flat, back
}

// loopMinY returns the smallest Y among a filletLoop's points.
func loopMinY(loop filletLoop) float64 {
	lowest := loop.pts[0].Y
	for _, p := range loop.pts[1:] {
		if p.Y < lowest {
			lowest = p.Y
		}
	}
	return lowest
}

// loop2DArea is the shoelace area of loop's 2D projection (unsigned — winding-independent).
func loop2DArea(loop filletLoop, flat func(m.Point3) m.Point2) float64 {
	pts := project2D(loop.pts, flat)
	n := len(pts)
	var sum2 float64
	for i := 0; i < n; i++ {
		a, b := pts[i], pts[(i+1)%n]
		sum2 += a.X*b.Y - b.X*a.Y
	}
	return stdmath.Abs(sum2) / 2
}

// loopHasPointNear reports whether some point of loop lies within tol of target — used to prove
// the merge absorbed the ellipse's UPPER (host-side) arc specifically, not the lower dip.
func loopHasPointNear(loop filletLoop, target m.Point3, tol float64) bool {
	for _, p := range loop.pts {
		if p.DistanceTo(target) <= tol {
			return true
		}
	}
	return false
}

// TestMergeHoleIntoNotchT6 pins the corrected T6 geometry (verified against the OCCT oracle,
// see task-5-report.md): the notch is the ellipse's UPPER (host-side) part, not the lower dip
// the patch rim (Task 3/4) uses. The brief's original test asserted y-min≈-10 (the dip side) —
// that was the wrong arc; the notched face's front edge stays at the receded boundary y=-7 and
// the removed area is the ellipse-above-y=-7 segment (760 rect - 426.914 ellipse-part = 333.086).
func TestMergeHoleIntoNotchT6(t *testing.T) {
	outer := recededTopOuterT6()
	hole := ellipseHoleT6()
	nodes := nodesForT6(t)
	flat, back := zPlaneProjector()

	notch, ok := mergeHoleIntoNotch(outer, hole, nodes, flat, back)
	if !ok {
		t.Fatal("merge failed")
	}
	if ymin := loopMinY(notch); stdmath.Abs(ymin-(-7)) > 0.05 {
		t.Errorf("notch front edge should stay at the receded boundary y=-7, got y-min %.4f", ymin)
	}
	if area := loop2DArea(notch, flat); stdmath.Abs(area-333.086)/333.086 > 0.01 {
		t.Errorf("notch area = %.4f, want 333.086 (760 rect - 426.914 ellipse-part) within 1%%", area)
	}
	if selfCrosses(notch, flat) {
		t.Errorf("notched loop must be a simple polygon")
	}
	if !loopHasPointNear(notch, m.P3(0, 10, 0), 0.05) {
		t.Errorf("notch must absorb the ellipse's UPPER arc (expected a point near (0,10,0))")
	}
}

// TestMergeHoleIntoNotchEmptyLoopsRejected covers mergeHoleIntoNotch's honest-reject path: an
// empty outer or hole loop is malformed input, not a zero-size notch.
func TestMergeHoleIntoNotchEmptyLoopsRejected(t *testing.T) {
	outer, hole := recededTopOuterT6(), ellipseHoleT6()
	nodes := nodesForT6(t)
	flat, back := zPlaneProjector()
	if _, ok := mergeHoleIntoNotch(filletLoop{}, hole, nodes, flat, back); ok {
		t.Error("empty outer loop must be rejected")
	}
	if _, ok := mergeHoleIntoNotch(outer, filletLoop{}, nodes, flat, back); ok {
		t.Error("empty hole loop must be rejected")
	}
}

// TestMergeHoleIntoNotchNoFrontEdgeRejected covers the case where the outer loop has no single
// edge spanning both Nodes (e.g. a rectangle whose front edge is not the receded boundary) —
// frontEdgeSegment must find zero candidates and honest-reject rather than mis-splice.
func TestMergeHoleIntoNotchNoFrontEdgeRejected(t *testing.T) {
	var outer filletLoop
	outer.add(m.P3(-20, -1, 0), nil) // shifted off y=-7: the Nodes no longer lie on this edge
	outer.add(m.P3(20, -1, 0), nil)
	outer.add(m.P3(20, 12, 0), nil)
	outer.add(m.P3(-20, 12, 0), nil)
	hole := ellipseHoleT6()
	nodes := nodesForT6(t)
	flat, back := zPlaneProjector()
	if _, ok := mergeHoleIntoNotch(outer, hole, nodes, flat, back); ok {
		t.Error("an outer loop with no edge spanning both Nodes must be rejected")
	}
}
