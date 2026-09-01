// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Guards for the EXACT retrace detector (M48/C3, Oblikovati/Oblikovati#3475). The old detector
// developed a DISCRETIZED boundary into the surface's metric chart, so the retraced length it reported
// was a sum of chords and moved with the caller's Quality. These tests pin the replacement on a CURVED
// back-track, where the difference is visible: the reported length is the arc's own closed form, at
// every quality.

// arcSpikeCentre, arcSpikeRadius and the sweep below define the fixture's spike: a quarter arc run out
// from the box corner and a HALF of it run straight back, so the retraced stretch is exactly
// arcSpikeRadius × π/4.
var (
	arcSpikeCentre = math.P3(100, 0, 10)
	arcSpikeNormal = math.V3(0, 1, 0)
	arcSpikeRef    = math.V3(1, 0, 0)
)

const arcSpikeRadius = 10.0

// arcSpikeFaceBody builds a planar face in y = 0 whose boundary runs out along a quarter arc and back
// along half of that same arc — a retrace whose length no chord sum can report exactly.
func arcSpikeFaceBody(t *testing.T) *topo.Body {
	t.Helper()
	out := arcSpikeArc(t, stdmath.Pi/2, -stdmath.Pi/2) // corner → arc end
	back := arcSpikeArc(t, 0, stdmath.Pi/4)            // arc end → half way back
	pl, err := geom.NewPlane(math.P3(0, 0, 0), arcSpikeNormal)
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("spike", "body", 0)))
	uses := arcSpikeLoopUses(bld, out, back)
	bld.AddFace(pl, topo.NewLineage(topo.Tok("spike", "f", 0)), topo.OuterLoop(uses...))
	return bld.Build()
}

// arcSpikeArc is one arc of the spike, on the fixture's shared circle.
func arcSpikeArc(t *testing.T, start, sweep float64) geom.Arc3d {
	t.Helper()
	arc, err := geom.NewArc3d(arcSpikeCentre, arcSpikeNormal, arcSpikeRef, arcSpikeRadius, start, sweep)
	if err != nil {
		t.Fatalf("NewArc3d(%g, %g): %v", start, sweep, err)
	}
	return arc
}

// arcSpikeLoopUses wires the five-edge loop: base line, arc out, arc back, and two closing lines.
func arcSpikeLoopUses(bld *topo.Builder, out, back geom.Arc3d) []topo.Use {
	lin := topo.NewLineage(topo.Tok("spike", "e", 0))
	corners := []math.Point3{math.P3(0, 0, 0), out.PointAt(0), out.PointAt(1), back.PointAt(1), math.P3(0, 0, 60)}
	verts := make([]*topo.Vertex, len(corners))
	for i, p := range corners {
		verts[i] = bld.AddVertex(p, lin)
	}
	curves := []geom.Curve3{
		geom.NewLineSegment(corners[0], corners[1]), out, back,
		geom.NewLineSegment(corners[3], corners[4]), geom.NewLineSegment(corners[4], corners[0]),
	}
	uses := make([]topo.Use, len(curves))
	for i, c := range curves {
		uses[i] = topo.Fwd(bld.AddEdge(c, verts[i], verts[(i+1)%len(corners)], lin))
	}
	return uses
}

// TestRetraceLengthIsTheExactArcLengthAtEveryQuality is the #3475 acceptance criterion, on a fixture
// whose answer a chord sum cannot reach: the spike's two arcs share exactly a quarter of the circle's
// quarter, so the closed form is 10 × π/4. Falsify by routing the detector back through a discretized
// boundary — the two qualities then disagree, and neither matches the closed form.
func TestRetraceLengthIsTheExactArcLengthAtEveryQuality(t *testing.T) {
	t.Parallel()
	body := arcSpikeFaceBody(t)
	want := arcSpikeRadius * stdmath.Pi / 4
	for _, q := range []Quality{DefaultQuality(), PropertyQuality()} {
		bad := RetracingFaceLoops(body, q)
		if len(bad) != 1 {
			t.Fatalf("at chord tol %g the arc spike reports %d retracing loop(s), want 1", q.tol(), len(bad))
		}
		if rel := stdmath.Abs(bad[0].Overlap-want) / want; rel > 1e-12 { // tol:numeric (relative length)
			t.Errorf("at chord tol %g the retrace measures %.12g, closed form %.12g (rel %.4g)",
				q.tol(), bad[0].Overlap, want, rel)
		}
	}
}

// TestVertexNeighbourhoodIsNotARetrace is the regression the OCCT blend-parity corpus produced the
// moment the detector moved onto the exact curves: every pair of edges meeting at a shared vertex
// coincides over a ball of the weld radius about it, so without a floor the detector reports the vertex
// itself — 34 loops across 20 corpus cases, all between 1.4e-16 and 1.9e-10 long. Here the spike is a
// REAL back-track, but 1e-11 of one on a 117-long face.
func TestVertexNeighbourhoodIsNotARetrace(t *testing.T) {
	t.Parallel()
	loop := []math.Point3{
		math.P3(0, 0, 0), math.P3(100, 0, 0), math.P3(100, 0, 40), math.P3(100, 0, 40-1e-11),
		math.P3(100, 0, 60), math.P3(0, 0, 60),
	}
	body := planarLoopBody(t, math.P3(0, 0, 0), math.V3(0, 1, 0), loop)
	if bad := RetracingFaceLoops(body, PropertyQuality()); len(bad) != 0 {
		t.Errorf("a 1e-11 back-track is the shared vertex, not a retrace, got %d: %+v", len(bad), bad)
	}
}

// TestPeriodicSeamIsNotARetrace is the guard on the one named exclusion: a closed cylindrical face
// bounds itself with ONE seam edge used twice, forward and back. Those two uses genuinely cover the
// same ground in opposite senses, and reporting them would condemn every seamed face in the kernel.
func TestPeriodicSeamIsNotARetrace(t *testing.T) {
	t.Parallel()
	body, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	if bad := RetracingFaceLoops(body, PropertyQuality()); len(bad) != 0 {
		t.Errorf("a plain cylinder reports %d retracing loop(s): %+v", len(bad), bad)
	}
}

// TestSpanCandidatesAreTheTwoTrimsOwnEnds pins WHY the reported length is exact: the only parameters a
// coincident stretch can begin or end at are the two trims' ends, so they are the only cuts considered.
// Here the other curve's far end falls at the middle of this one.
func TestSpanCandidatesAreTheTwoTrimsOwnEnds(t *testing.T) {
	t.Parallel()
	a := geom.NewLineSegment(math.P3(0, 0, 0), math.P3(10, 0, 0))
	b := geom.NewLineSegment(math.P3(5, 0, 0), math.P3(20, 0, 0))
	cuts := spanCandidates(a, b)
	if len(cuts) != 3 {
		t.Fatalf("candidates %v, want a's two ends plus b's interior end", cuts)
	}
	if stdmath.Abs(cuts[1]-0.5) > 1e-12 { // tol:numeric (parameter)
		t.Errorf("the interior candidate is %g, want b's start projected onto a (0.5)", cuts[1])
	}
}

// TestPointOfCurveLiesOnRespectsTheOtherTrim: a point on the other curve's INFINITE support but past
// its trim is not on it, which is what stops a segment from "covering" its own extension.
func TestPointOfCurveLiesOnRespectsTheOtherTrim(t *testing.T) {
	t.Parallel()
	a := geom.NewLineSegment(math.P3(0, 0, 0), math.P3(10, 0, 0))
	b := geom.NewLineSegment(math.P3(6, 0, 0), math.P3(10, 0, 0))
	if !pointOfCurveLiesOn(a, b, 0.8, 1e-9) {
		t.Error("a's point at x = 8 lies inside b's trim and must read as on it")
	}
	if pointOfCurveLiesOn(a, b, 0.2, 1e-9) {
		t.Error("a's point at x = 2 is past b's trim and must not read as on it")
	}
}

// TestLongestCoveredRunMergesConsecutiveCoveredIntervals: two adjacent covered sub-intervals are ONE
// stretch, and the longest run wins over an earlier shorter one.
func TestLongestCoveredRunMergesConsecutiveCoveredIntervals(t *testing.T) {
	t.Parallel()
	c := geom.NewLineSegment(math.P3(0, 0, 0), math.P3(10, 0, 0))
	lo, hi, ok := longestCoveredRun(c, []float64{0, 0.1, 0.4, 0.5, 0.9}, []bool{true, false, true, true})
	if !ok {
		t.Fatal("a run of covered sub-intervals must produce an interval")
	}
	if lo != 0.4 || hi != 0.9 {
		t.Errorf("longest run is [%g, %g], want the merged [0.4, 0.9]", lo, hi)
	}
	if _, _, ok := longestCoveredRun(c, []float64{0, 1}, []bool{false}); ok {
		t.Error("no covered sub-interval must produce no interval")
	}
}

// TestStartOfRunKeepsTheFirstIndex covers the run bookkeeping directly: the start is claimed once and
// then held, so a run that continues does not restart at each covered sub-interval.
func TestStartOfRunKeepsTheFirstIndex(t *testing.T) {
	t.Parallel()
	if got := startOfRun(-1, 3); got != 3 {
		t.Errorf("an unclaimed run must start at the current index, got %d", got)
	}
	if got := startOfRun(3, 4); got != 3 {
		t.Errorf("a claimed run must keep its first index 3, got %d", got)
	}
}

// TestSpanHoldsThroughoutRejectsAMidpointCoincidence is the guard on the one thing exact endpoints
// cannot certify: two curves that meet only where the sub-interval midpoint happened to probe. A
// segment crossing another at its centre is on it there and nowhere else.
func TestSpanHoldsThroughoutRejectsAMidpointCoincidence(t *testing.T) {
	t.Parallel()
	a := geom.NewLineSegment(math.P3(0, -5, 0), math.P3(0, 5, 0))
	b := geom.NewLineSegment(math.P3(-5, 0, 0), math.P3(5, 0, 0))
	if spanHoldsThroughout(a, b, 0, 1, 1e-9) {
		t.Error("two segments crossing at one point do not coincide over a stretch")
	}
}

// TestOppositeTraversalReadsTheLoopsOwnDirection: the same two coincident edges are a back-track only
// when the loop runs them in opposite senses, and reversing one use flips the verdict.
func TestOppositeTraversalReadsTheLoopsOwnDirection(t *testing.T) {
	t.Parallel()
	body := arcSpikeFaceBody(t)
	uses := body.Faces()[0].Loops()[0].EdgeUses()
	out, back := uses[1], uses[2] // the arc out and the arc back
	ca, cb := out.Edge().Geometry(), back.Edge().Geometry()
	lo, hi, ok := coincidentSpanOn(ca, cb, 1e-9)
	if !ok {
		t.Fatal("the two arcs of the spike must share a stretch")
	}
	if !oppositeTraversal(out, back, ca, cb, lo, hi) {
		t.Error("the spike's two arcs are traversed in opposite senses and must read as a back-track")
	}
	if oppositeTraversal(out, out, ca, ca, lo, hi) {
		t.Error("an edge use compared with itself runs the SAME way and is not a back-track")
	}
}
