// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Guards for the exact face-loop corner ring (M48/C3, Oblikovati/Oblikovati#3476): the boundary the
// loop validity detectors now run on, derived from topology alone rather than from a tessellation.

// TestLoopCornerRingIsOnePointPerEdgeUse: a four-edge loop yields exactly its four corners, in
// traversal order — not the chord polyline the tessellator would have produced.
func TestLoopCornerRingIsOnePointPerEdgeUse(t *testing.T) {
	t.Parallel()
	corners := []math.Point3{math.P3(0, 0, 0), math.P3(4, 0, 0), math.P3(4, 0, 3), math.P3(0, 0, 3)}
	f := planarLoopBody(t, math.P3(0, 0, 0), math.V3(0, 1, 0), corners).Faces()[0]
	ring := faceOuterCornerRing(f).pts
	if len(ring) != len(corners) {
		t.Fatalf("corner ring has %d points, want one per edge use (%d)", len(ring), len(corners))
	}
	for i, want := range corners {
		if ring[i].DistanceTo(want) > 1e-12 { // tol:numeric — the ring reports the vertex points verbatim
			t.Errorf("corner %d is %v, want the loop vertex %v", i, ring[i], want)
		}
	}
}

// TestCornerRingIsTheSameAtEveryQuality is the #3476 property in its plainest form: the ring carries no
// Quality at all, so a curved boundary that the tessellator would sample differently at display and
// property quality still yields the same ring. Falsify by routing the ring back through loopBoundary.
func TestCornerRingIsTheSameAtEveryQuality(t *testing.T) {
	t.Parallel()
	f := lobedBandBody(t, 24, 30, 100, 8, 6).Faces()[0]
	ring := faceOuterCornerRing(f).pts
	if len(ring) != 5 {
		t.Fatalf("the lobed band's loop has 5 edge uses, its corner ring has %d points", len(ring))
	}
	for _, q := range []Quality{DefaultQuality(), PropertyQuality()} {
		if got := len(loopBoundary(f.Loops()[0], q)); got == len(ring) {
			t.Errorf("the discretized boundary at chord tol %g has %d points too, so this fixture "+
				"cannot tell the two apart", q.tol(), got)
		}
	}
}

// TestFaceCornerRingsPutTheOuterLoopFirst pins the ORDER the developed loops are paired against: outer
// first, then the holes, the same order developedFaceLoops develops them in.
func TestFaceCornerRingsPutTheOuterLoopFirst(t *testing.T) {
	t.Parallel()
	f := planarLoopBody(t, math.P3(0, 0, 0), math.V3(0, 1, 0),
		[]math.Point3{math.P3(0, 0, 0), math.P3(9, 0, 0), math.P3(9, 0, 9), math.P3(0, 0, 9)}).Faces()[0]
	rings := faceCornerRings(f)
	if len(rings) != 1 || len(rings[0].pts) != 4 {
		t.Fatalf("a one-loop face yields one 4-corner ring, got %d ring(s) %v", len(rings), rings)
	}
	if rings[0].pts[0].DistanceTo(math.P3(0, 0, 0)) > 1e-12 { // tol:numeric — verbatim vertex
		t.Errorf("the first ring must be the outer loop, starting at its first vertex, got %v", rings[0].pts[0])
	}
	if len(rings[0].owners) != 4 || rings[0].owners[0] != f.Loops()[0].EdgeUses()[0].Edge() {
		t.Errorf("each ring point must carry the edge its outgoing segment lies on, got %v", rings[0].owners)
	}
}

// TestEdgeUseStartPointFollowsTheUseOrientation: a REVERSED use enters its edge at the edge's END
// vertex, so the ring must read that one. Getting this wrong reverses one segment of the polygon and
// invents a crossing that is not there.
func TestEdgeUseStartPointFollowsTheUseOrientation(t *testing.T) {
	t.Parallel()
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("use", "body", 0)))
	lin := topo.NewLineage(topo.Tok("use", "x", 0))
	v0 := bld.AddVertex(math.P3(0, 0, 0), lin)
	v1 := bld.AddVertex(math.P3(1, 0, 0), lin)
	e := bld.AddEdge(geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0)), v0, v1, lin)
	pl, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 1, 0))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	bld.AddFace(pl, lin, topo.OuterLoop(topo.Use{Edge: e, Reversed: true}))
	use := bld.Build().Faces()[0].Loops()[0].EdgeUses()[0]
	if got := edgeUseStartPoint(use); got.DistanceTo(math.P3(1, 0, 0)) > 1e-12 { // tol:numeric — verbatim vertex
		t.Errorf("a reversed use starts at the edge's END vertex (1,0,0), got %v", got)
	}
}

// wideCylBandBody builds a cylindrical band of radius r spanning `sweep` radians about the axis and
// `height` along it — the fixture for the periodicity refinement, because one rim arc of it spans more
// than half a period.
func wideCylBandBody(t *testing.T, r, sweep, height float64) *topo.Body {
	t.Helper()
	cyl, err := geom.NewCylinderWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), r)
	if err != nil {
		t.Fatalf("NewCylinderWithRef: %v", err)
	}
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("band", "body", 0)))
	lin := topo.NewLineage(topo.Tok("band", "x", 0))
	corners := [][2]float64{{0, 0}, {sweep, 0}, {sweep, height}, {0, height}}
	verts := make([]*topo.Vertex, len(corners))
	for i, c := range corners {
		verts[i] = bld.AddVertex(cyl.PointAt(c[0], c[1]), lin)
	}
	uses := wideCylBandUses(t, bld, cyl, corners, verts, lin)
	bld.AddFace(cyl, lin, topo.OuterLoop(uses...))
	return bld.Build()
}

// wideCylBandUses wires the band's four edges: an iso-v rim arc, an axial ruling, the far rim arc back,
// and the closing ruling.
func wideCylBandUses(t *testing.T, bld *topo.Builder, cyl geom.Cylinder, corners [][2]float64,
	verts []*topo.Vertex, lin topo.Lineage) []topo.Use {
	t.Helper()
	uses := make([]topo.Use, len(corners))
	for i, c := range corners {
		j := (i + 1) % len(corners)
		var curve geom.Curve3 = geom.NewLineSegment(cyl.PointAt(c[0], c[1]), cyl.PointAt(corners[j][0], corners[j][1]))
		if c[1] == corners[j][1] {
			arc, err := geom.NewArc3d(math.P3(0, 0, math.Scalar(c[1])), math.V3(0, 0, 1), math.V3(1, 0, 0),
				cyl.Radius, c[0], corners[j][0]-c[0])
			if err != nil {
				t.Fatalf("NewArc3d: %v", err)
			}
			curve = arc
		}
		uses[i] = topo.Fwd(bld.AddEdge(curve, verts[i], verts[j], lin))
	}
	return uses
}

// TestPeriodicRefinementDevelopsAWideBandAtItsTrueWidth is the guard on the one refinement the corner
// ring makes. A 270° band's rim arc is a SINGLE edge, and one step across it unwraps the short way —
// as −90° — so the development would render a 270° band as a 90° one. Falsify by returning 1 from
// chartStepsFor: the measured width collapses to r × π/2.
func TestPeriodicRefinementDevelopsAWideBandAtItsTrueWidth(t *testing.T) {
	t.Parallel()
	const r, height = 24.0, 10.0
	sweep := 3 * stdmath.Pi / 2
	band := wideCylBandBody(t, r, sweep, height).Faces()[0]
	loops, ok := developedFaceLoops(band, faceCornerRings(band))
	if !ok || len(loops) != 1 {
		t.Fatalf("the wide band must develop into exactly one loop (ok=%v, %d loops)", ok, len(loops))
	}
	lo, hi := stdmath.Inf(1), stdmath.Inf(-1)
	for _, p := range loops[0].pts {
		lo, hi = stdmath.Min(lo, float64(p.X)), stdmath.Max(hi, float64(p.X))
	}
	want := r * sweep
	if rel := stdmath.Abs((hi-lo)-want) / want; rel > 1e-9 { // tol:numeric (relative length)
		t.Errorf("developed width %.10g, true band width %.10g (rel %.4g)", hi-lo, want, rel)
	}
}

// TestHalvesAgreeCatchesAWrongWayUnwrap covers the refinement's predicate directly: a step of 3π/2 is
// unwrapped as −π/2 while its halves each unwrap as +3π/4, so the two disagree and the step splits. A
// step of 3π/4 agrees with its halves and does not.
func TestHalvesAgreeCatchesAWrongWayUnwrap(t *testing.T) {
	t.Parallel()
	if halvesAgree(0, 3*stdmath.Pi/4, 3*stdmath.Pi/2) {
		t.Error("a 3π/2 step unwraps the wrong way and must disagree with its halves")
	}
	if !halvesAgree(0, 3*stdmath.Pi/8, 3*stdmath.Pi/4) {
		t.Error("a 3π/4 step unwraps correctly and must agree with its halves")
	}
}

// archedNotchFaceBody builds a planar face in y = 0 whose top boundary is a SEMICIRCULAR arc and whose
// bottom runs down to a spike. The arc's chord lies along z = 0 and the spike's two sides cross it, so
// the corner ring's chart polygon self-crosses — while the true boundary, which bulges to z = 5 above
// the spike's z = 2, is a simple closed curve.
func archedNotchFaceBody(t *testing.T) *topo.Body {
	t.Helper()
	arc, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 1, 0), math.V3(1, 0, 0), 5, stdmath.Pi, stdmath.Pi)
	if err != nil {
		t.Fatalf("NewArc3d: %v", err)
	}
	pl, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 1, 0))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("arch", "body", 0)))
	corners := []math.Point3{arc.PointAt(0), arc.PointAt(1), math.P3(5, 0, -5), math.P3(0, 0, 2), math.P3(-5, 0, -5)}
	bld.AddFace(pl, topo.NewLineage(topo.Tok("arch", "f", 0)), topo.OuterLoop(archedNotchUses(bld, arc, corners)...))
	return bld.Build()
}

// archedNotchUses wires the arch: the semicircular top, then the four straight sides back to it.
func archedNotchUses(bld *topo.Builder, arc geom.Arc3d, corners []math.Point3) []topo.Use {
	lin := topo.NewLineage(topo.Tok("arch", "e", 0))
	verts := make([]*topo.Vertex, len(corners))
	for i, p := range corners {
		verts[i] = bld.AddVertex(p, lin)
	}
	uses := make([]topo.Use, len(corners))
	for i := range corners {
		j := (i + 1) % len(corners)
		var c geom.Curve3 = geom.NewLineSegment(corners[i], corners[j])
		if i == 0 {
			c = arc
		}
		uses[i] = topo.Fwd(bld.AddEdge(c, verts[i], verts[j], lin))
	}
	return uses
}

// TestArcChordDoesNotInventASelfCrossing is the regression the OCCT blend-parity corpus produced for
// the corner ring's other error direction — six planar faces reported a crossing that their boundary
// does not have. One chart segment per edge renders a curved edge as a straight line, and that line
// cuts across ground the arc keeps clear of. Falsify by letting loopSelfCrossing accept every candidate:
// this fixture then reports a crossing of an arch whose arch is simply not there.
func TestArcChordDoesNotInventASelfCrossing(t *testing.T) {
	t.Parallel()
	if bad := SelfCrossingFaceLoops(archedNotchFaceBody(t), PropertyQuality()); len(bad) != 0 {
		t.Errorf("an arc's chord invented %d self-crossing(s) on a simple boundary: %+v", len(bad), bad)
	}
}

// TestSegmentDevelopsItsEdgeSeparatesAChordFromALine covers the certificate directly on the same
// fixture: the arch's arc segment does NOT develop straight (its mid sits a radius off its chord) while
// every straight side does.
func TestSegmentDevelopsItsEdgeSeparatesAChordFromALine(t *testing.T) {
	t.Parallel()
	f := archedNotchFaceBody(t).Faces()[0]
	rings := faceCornerRings(f)
	loops, ok := developedFaceLoops(f, rings)
	if !ok || len(loops) != 1 {
		t.Fatalf("the arch must develop into exactly one loop (ok=%v, %d loops)", ok, len(loops))
	}
	if segmentDevelopsItsEdge(loops[0], 0) {
		t.Error("the semicircular arc's chord must NOT certify as a straight development of it")
	}
	for i := 1; i < len(loops[0].pts); i++ {
		if !segmentDevelopsItsEdge(loops[0], i) {
			t.Errorf("straight side %d must certify as its own development", i)
		}
	}
}

// TestSelfCrossingVerdictIsIndependentOfQuality is the #3476 acceptance criterion at the API: the same
// body must report the same loops and the same pinched-off area whatever Quality it is asked with.
func TestSelfCrossingVerdictIsIndependentOfQuality(t *testing.T) {
	t.Parallel()
	const r, w, l, h, d = 24.0, 30.0, 100.0, 8.0, 6.0
	body := lobedBandBody(t, r, w, l, h, d)
	base := SelfCrossingFaceLoops(body, DefaultQuality())
	other := SelfCrossingFaceLoops(body, PropertyQuality())
	if len(base) != 1 || len(other) != len(base) {
		t.Fatalf("loop counts differ by quality: %d at display, %d at property", len(base), len(other))
	}
	if stdmath.Abs(base[0].Area-other[0].Area) != 0 {
		t.Errorf("pinched-off area is %.17g at display quality and %.17g at property quality",
			base[0].Area, other[0].Area)
	}
	if stdmath.Abs(base[0].ChartChordRatio-other[0].ChartChordRatio) != 0 {
		t.Errorf("chart/chord ratio is %.17g at display quality and %.17g at property quality",
			base[0].ChartChordRatio, other[0].ChartChordRatio)
	}
}
