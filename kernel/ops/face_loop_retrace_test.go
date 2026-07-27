// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// y4HostPlaneLoop is the loop simple/Y4 ACTUALLY SHIPPED for its host plane before the band-imprint
// walk corrected it (chain-supplier-report.md §6.1): a 100×75 face carrying the 10×10 slot, whose
// closing segment (100,0,0)→(100,0,80) is re-covered by (100,0,90)→(100,0,75) over z ∈ [75,80].
//
// It is the fixture for BOTH directions of the guard because it is the measured defect itself, not a
// model of one: the retrace length is the closed form 80 − 75 = 5.
func y4HostPlaneLoop() []math.Point3 {
	return []math.Point3{
		math.P3(100, 0, 80), math.P3(90, 0, 80), math.P3(90, 0, 90), math.P3(100, 0, 90),
		math.P3(100, 0, 75), math.P3(0, 0, 75), math.P3(0, 0, 0), math.P3(100, 0, 0),
	}
}

// TestRetracingFaceLoopsCatchesACollinearBackTrack is the falsifiable guard, positive direction: the
// shipped Y4 host loop must be reported, and its overlap must be the closed-form 5.
//
// Falsify by dropping the opposite-direction test in collinearBacktrack (the loop then still reports,
// but so does every legitimate subdivided edge — see the negative guards), or by requiring a
// transversal crossing: nothing is reported and this goes RED.
func TestRetracingFaceLoopsCatchesACollinearBackTrack(t *testing.T) {
	body := planarLoopBody(t, math.P3(0, 0, 0), math.V3(0, 1, 0), y4HostPlaneLoop())
	bad := RetracingFaceLoops(body, PropertyQuality())
	if len(bad) != 1 {
		t.Fatalf("a loop that back-tracks along its own closing segment must report exactly one loop, got %d", len(bad))
	}
	const want = 5.0                                                 // z ∈ [75,80] covered twice, in the plane's own metric
	if rel := stdmath.Abs(bad[0].Overlap-want) / want; rel > 1e-12 { // tol:numeric (relative length)
		t.Errorf("retraced length %.12g, closed form %.12g (rel %.4g)", bad[0].Overlap, want, rel)
	}
}

// TestSelfCrossingFaceLoopsIsBlindToACollinearBackTrack pins the GAP this detector closes, so the two
// predicates can never be confused for one another: the very same shipped Y4 loop scores ZERO
// transversal self-crossings, because two overlapping collinear segments never straddle each other's
// line and every orient2d in segmentsCross is exactly 0.
func TestSelfCrossingFaceLoopsIsBlindToACollinearBackTrack(t *testing.T) {
	body := planarLoopBody(t, math.P3(0, 0, 0), math.V3(0, 1, 0), y4HostPlaneLoop())
	if bad := SelfCrossingFaceLoops(body, PropertyQuality()); len(bad) != 0 {
		t.Errorf("the transversal predicate is expected to be blind here, got %d: %+v", len(bad), bad)
	}
}

// TestRetracingFaceLoopsPassesASubdividedStraightEdge is the falsifiable guard, NEGATIVE direction —
// the one that matters most, because a sloppy "any collinear pair" predicate fails it. A rectangle
// whose bottom edge is split into three collinear pieces is a perfectly legitimate boundary: the
// pieces are collinear and consecutive, but they run the SAME way and cover disjoint ground.
func TestRetracingFaceLoopsPassesASubdividedStraightEdge(t *testing.T) {
	loop := []math.Point3{
		math.P3(0, 0, 0), math.P3(30, 0, 0), math.P3(60, 0, 0), math.P3(100, 0, 0),
		math.P3(100, 0, 60), math.P3(60, 0, 60), math.P3(0, 0, 60),
	}
	body := planarLoopBody(t, math.P3(0, 0, 0), math.V3(0, 1, 0), loop)
	if bad := RetracingFaceLoops(body, PropertyQuality()); len(bad) != 0 {
		t.Errorf("a subdivided straight edge must not be reported, got %d: %+v", len(bad), bad)
	}
}

// TestRetracingFaceLoopsPassesAThinSliver pins the other false-positive direction: two ANTI-parallel
// edges that are nearly — but not degenerately — collinear. A 100 × 1e-5 sliver is a thin face, not a
// zero-area one, and its width sits two decades above retraceCollinearTol × the loop's diagonal.
func TestRetracingFaceLoopsPassesAThinSliver(t *testing.T) {
	const w = 1e-5
	loop := []math.Point3{math.P3(0, 0, 0), math.P3(100, 0, 0), math.P3(100, 0, w), math.P3(0, 0, w)}
	body := planarLoopBody(t, math.P3(0, 0, 0), math.V3(0, 1, 0), loop)
	if bad := RetracingFaceLoops(body, PropertyQuality()); len(bad) != 0 {
		t.Errorf("a thin but two-dimensional sliver must not be reported, got %d: %+v", len(bad), bad)
	}
}

// TestRetracingFaceLoopsCatchesAZeroWidthSpike pins the smallest form of the defect: two CONSECUTIVE
// segments that are collinear and antiparallel — the boundary reversing at a vertex. It is the same
// violation as Y4's at zero separation, and the direction test admits it while still passing the
// subdivided edge above, which shares the identical adjacency.
func TestRetracingFaceLoopsCatchesAZeroWidthSpike(t *testing.T) {
	loop := []math.Point3{
		math.P3(0, 0, 0), math.P3(100, 0, 0), math.P3(100, 0, 40), math.P3(100, 0, 10),
		math.P3(100, 0, 60), math.P3(0, 0, 60),
	}
	body := planarLoopBody(t, math.P3(0, 0, 0), math.V3(0, 1, 0), loop)
	bad := RetracingFaceLoops(body, PropertyQuality())
	if len(bad) != 1 {
		t.Fatalf("a zero-width spike must report exactly one loop, got %d", len(bad))
	}
	const want = 30.0                                                // the spike runs 40 → 10 and back out through 40
	if rel := stdmath.Abs(bad[0].Overlap-want) / want; rel > 1e-12 { // tol:numeric (relative length)
		t.Errorf("spike length %.12g, closed form %.12g (rel %.4g)", bad[0].Overlap, want, rel)
	}
}

// planarLoopBody builds a one-face body from a 3D point loop lying in the plane through org with
// normal n, every side a line edge — the fixture shape the plane-boundary guards share.
func planarLoopBody(t *testing.T, org math.Point3, n math.Vector3, loop []math.Point3) *topo.Body {
	t.Helper()
	pl, err := geom.NewPlane(org, n)
	if err != nil {
		t.Fatalf("NewPlane(%v, %v): %v", org, n, err)
	}
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("c", "body", 0)))
	lin := topo.NewLineage(topo.Tok("c", "x", 0))
	verts := make([]*topo.Vertex, len(loop))
	for i, p := range loop {
		verts[i] = bld.AddVertex(p, lin)
	}
	uses := make([]topo.Use, len(loop))
	for i := range loop {
		j := (i + 1) % len(loop)
		uses[i] = topo.Fwd(bld.AddEdge(geom.NewLineSegment(loop[i], loop[j]), verts[i], verts[j], lin))
	}
	bld.AddFace(pl, lin, topo.OuterLoop(uses...))
	return bld.Build()
}
