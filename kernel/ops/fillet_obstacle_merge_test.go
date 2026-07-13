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
	assertT6Notch(t, notch, flat)
}

// assertT6Notch is the shared invariant for a correctly-merged T6 notch, whatever orientation the
// splice took: a single simple loop whose front edge stays at the receded boundary y≈-7, whose 2D
// area is 760 (rect) − 426.914 (ellipse-above-boundary) = 333.086, and which reaches up into the
// rectangle along the absorbed UPPER ellipse arc (a point near its apex (0,10,0)).
func assertT6Notch(t *testing.T, notch filletLoop, flat func(m.Point3) m.Point2) {
	t.Helper()
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

// TestMergeHoleIntoNotchNativeOrientation exercises orientedHostArc's NATIVE (non-reversed) branch,
// which T6 never hits (T6 always reverses). Traversing the receded front edge in the opposite
// direction — a clockwise rectangle — flips nearerNode's verdict so the host arc is spliced in its
// native P+→P- order. The merged loop must still satisfy every T6 invariant (single simple loop,
// y-min≈-7, area≈333.086, upper arc absorbed) — proving the orientation branch, not just the case
// T6 happens to take, is correct.
func TestMergeHoleIntoNotchNativeOrientation(t *testing.T) {
	outer := recededTopOuterT6CW()
	hole := ellipseHoleT6()
	nodes := nodesForT6(t)
	flat, back := zPlaneProjector()
	if nearerNode(outer, 0, nodes, flat) != 1 {
		t.Fatalf("fixture must select the NATIVE branch (nearerNode==1); got %d", nearerNode(outer, 0, nodes, flat))
	}
	notch, ok := mergeHoleIntoNotch(outer, hole, nodes, flat, back)
	if !ok {
		t.Fatal("merge failed on the native-orientation fixture")
	}
	assertT6Notch(t, notch, flat)
}

// TestMergeHoleIntoNotchBoundarySegmentFidelity is the regression guard for the stale-curve fix: the
// two host-arc segments that touch the truncation crossings P± must carry NO domain-mismatched curve.
// It samples each boundary segment exactly as the mesher would (nil ⇒ straight chord; non-nil ⇒
// curve.PointAt over [0,1]) and requires every sample within the model weld of the segment's OWN
// (truncated) chord. The stale original-segment curve — whose small-t samples fall in the DISCARDED
// span beyond the crossing — lands a full sample-spacing off that chord, so this fails loudly if the
// curve is ever reintroduced.
func TestMergeHoleIntoNotchBoundarySegmentFidelity(t *testing.T) {
	hole := ellipseHoleT6()
	nodes := nodesForT6(t)
	_, back := zPlaneProjector()
	arc := hostSideSubArc(hole, nodes, back)
	weld := ResolutionForPoints(arc.pts).Weld()
	for _, seg := range []int{0, len(arc.pts) - 2} { // P+ leg and P- leg
		if arc.curves[seg] != nil {
			t.Errorf("boundary segment %d must not carry the stale original-segment curve", seg)
		}
		dev := maxSegmentChordDeviation(arc.pts[seg], arc.pts[seg+1], arc.curves[seg])
		if dev > weld {
			t.Errorf("boundary segment %d samples %.3g off its truncated chord (weld %.3g) — stale span", seg, dev, weld)
		}
	}
}

// TestMergeHoleIntoNotchAmbiguousEdgeRejected covers frontEdgeSegment's hits>1 honest-reject: an
// outer loop with TWO edges each spanning both Nodes on the receded boundary is ambiguous (the
// splice cannot pick a cut edge), so the merge must reject rather than mis-splice.
func TestMergeHoleIntoNotchAmbiguousEdgeRejected(t *testing.T) {
	var outer filletLoop
	outer.add(m.P3(-20, -7, 0), nil) // edge 0: (-20,-7)->(20,-7) spans both Nodes
	outer.add(m.P3(20, -7, 0), nil)  // edge 1: (20,-7)->(-20,-7) ALSO spans both Nodes (back-and-forth)
	outer.add(m.P3(-20, -7, 0), nil)
	outer.add(m.P3(20, 12, 0), nil)
	outer.add(m.P3(-20, 12, 0), nil)
	hole := ellipseHoleT6()
	nodes := nodesForT6(t)
	flat, back := zPlaneProjector()
	if _, ok := mergeHoleIntoNotch(outer, hole, nodes, flat, back); ok {
		t.Error("an outer loop with two front-edge candidates must be rejected as ambiguous")
	}
}

// recededTopOuterT6CW is recededTopOuterT6 with the front edge traversed right-to-left (a clockwise
// rectangle): same geometry, opposite winding — the fixture that drives orientedHostArc's native
// (non-reversed) branch.
func recededTopOuterT6CW() filletLoop {
	var loop filletLoop
	loop.add(m.P3(20, -7, 0), nil)
	loop.add(m.P3(-20, -7, 0), nil)
	loop.add(m.P3(-20, 12, 0), nil)
	loop.add(m.P3(20, 12, 0), nil)
	return loop
}

// maxSegmentChordDeviation samples a segment as the mesher would — the straight chord p0→p1 when
// curve is nil, else curve.PointAt(t) over its [0,1] domain — and returns the largest perpendicular
// distance of an interior sample from the chord p0→p1. Small-t samples are included on purpose: a
// stale full-segment curve places them in the discarded span, off the truncated chord.
func maxSegmentChordDeviation(p0, p1 m.Point3, curve geom.Curve3) float64 {
	var maxDev float64
	for _, u := range []float64{0.05, 0.15, 0.3, 0.5, 0.7, 0.9} {
		s := p0.Lerp(p1, u)
		if curve != nil {
			s = curve.PointAt(u)
		}
		if d := pointToSegmentDist3(s, p0, p1); d > maxDev {
			maxDev = d
		}
	}
	return maxDev
}

// pointToSegmentDist3 is the distance from p to the segment a→b (clamped at the endpoints, so a
// sample projecting BEYOND an endpoint — into the discarded span — reports the true off-chord gap).
func pointToSegmentDist3(p, a, b m.Point3) float64 {
	ab := a.VectorTo(b)
	lenSq := ab.LengthSquared()
	if lenSq == 0 {
		return p.DistanceTo(a)
	}
	u := a.VectorTo(p).Dot(ab) / lenSq
	u = stdmath.Max(0, stdmath.Min(1, u))
	return p.DistanceTo(a.TranslateBy(ab.Scale(u)))
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
