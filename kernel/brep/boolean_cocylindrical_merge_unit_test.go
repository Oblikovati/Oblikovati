// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Unit statements for the cocylindrical merge's parts (ADR-0061 stage 5).

// wallFaceOf is the cylinder wall of a cylinder primitive, as the merge sees it.
func wallFaceOf(t *testing.T, at math.Point3, r, h float64) curvedFace {
	t.Helper()
	b, err := SolidCylinder(at, math.V3(0, 0, 1), math.Scalar(r), math.Scalar(h))
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	for _, f := range facesOfAny(b) {
		if _, ok := f.surface.(geom.Cylinder); ok {
			return f
		}
	}
	t.Fatal("the cylinder has no wall face")
	return curvedFace{}
}

// TestDissolveJoinsTwoAbuttingBandsIntoOneLoop: two bands meeting at a rim dissolve into one boundary,
// which is the seam-slit form a full band has — the seam walked twice and the two surviving rims.
func TestDissolveJoinsTwoAbuttingBandsIntoOneLoop(t *testing.T) {
	t.Parallel()
	a, b := wallFaceOf(t, math.P3(0, 0, 0), 2, 4), wallFaceOf(t, math.P3(0, 0, 4), 2, 3)
	res := geom.ResolutionForBox(faceLoopBox(a).Union(faceLoopBox(b)))
	loops, why := dissolveSharedEdges(a, b, res)
	if why != mergeJoined {
		t.Fatalf("two bands meeting at a rim share that rim; the dissolve declined: %s", why)
	}
	if len(loops) != 1 || len(loops[0].edges) != 6 {
		t.Fatalf("the dissolve gave %d loops (first has %d edges), want 1 of 6 (two seams twice, two rims)",
			len(loops), len(loops[0].edges))
	}
}

// TestDissolveDeclinesTwoBandsThatShareNoEdge is the negative row at the unit: the two walls ARE on one
// surface — the premise the e2e row rests on — and the dissolve still declines, because nothing they
// carry is a shared boundary.
func TestDissolveDeclinesTwoBandsThatShareNoEdge(t *testing.T) {
	t.Parallel()
	a, b := wallFaceOf(t, math.P3(0, 0, 0), 2, 4), wallFaceOf(t, math.P3(0, 0, 9), 2, 4)
	if !onOneSurface(a, b) {
		t.Fatal("two coaxial walls of one radius are not reported on one surface; the negative row is vacuous")
	}
	res := geom.ResolutionForBox(faceLoopBox(a).Union(faceLoopBox(b)))
	_, why := dissolveSharedEdges(a, b, res)
	if why != declineUnshared {
		t.Errorf("two separated bands gave %q, want the ordinary unshared exit — and it must stay the "+
			"one reason that records nothing, since nothing was given up", why)
	}
	if why.reportable() {
		t.Error("the ordinary two-faces-are-two-faces exit is reported as a degradation")
	}
}

// TestSharedEdgeTwinsPairsWholeEdgesOnly: the pairing is one-to-one, and the run a boss shares with a
// host rim reaches it whole because splitAtSharedRunEnds cut both sides at the other's vertices first.
func TestSharedEdgeTwinsPairsWholeEdgesOnly(t *testing.T) {
	t.Parallel()
	a, b := wallFaceOf(t, math.P3(0, 0, 0), 2, 4), wallFaceOf(t, math.P3(0, 0, 4), 2, 3)
	res := geom.ResolutionForBox(faceLoopBox(a).Union(faceLoopBox(b)))
	twin, why := sharedEdgeTwins(a, b, res)
	if why != mergeJoined {
		t.Fatalf("the abutting bands' shared rim was not paired: %s", why)
	}
	if len(twin) != 2 { // the pair is recorded from both ends
		t.Errorf("the pairing holds %d entries, want 2 (one edge pair, both ways)", len(twin))
	}
}

// seamWalkedWall is a cylinder wall whose ONE outer loop walks its seam twice — bottom rim, seam up,
// top rim, seam down — with the seam's curve chosen by the caller: the slit shape dropSeamSlits exists
// for, built through the topo builder so every loop edge carries its source edge as the merge sees it.
// twoSeams builds the down traversal on a SECOND edge along the same curve, which is two edges and not a
// slit whatever their curves look like.
func seamWalkedWall(t *testing.T, seam func(a, b math.Point3) geom.Curve3, twoSeams bool) []loopEdge {
	t.Helper()
	const r, h = 2.0, 4.0
	bottom, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), r)
	top := geom.Circle{Center: math.P3(0, 0, h), Normal: bottom.Normal, RefDir: bottom.RefDir, Radius: r}
	side, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r)
	lin := func(role string, i int) topo.Lineage { return topo.NewLineage(topo.Tok("slit", role, i)) }
	bld := topo.NewBuilder(true, lin("body", 0))
	vb, vt := bld.AddVertex(bottom.PointAt(0), lin("v", 0)), bld.AddVertex(top.PointAt(0), lin("v", 1))
	eb, et := bld.AddEdge(bottom, vb, vb, lin("e", 0)), bld.AddEdge(top, vt, vt, lin("e", 1))
	up := bld.AddEdge(seam(vb.Point(), vt.Point()), vb, vt, lin("e", 2))
	down := up
	if twoSeams {
		down = bld.AddEdge(seam(vb.Point(), vt.Point()), vb, vt, lin("e", 3))
	}
	f := bld.AddFace(side, lin("f", 0), topo.OuterLoop(topo.Fwd(eb), topo.Fwd(up), topo.Rev(et), topo.Rev(down)))
	return loopEdgesOf(f.Loops()[0])
}

// straightSeam is the ordinary seam: one line segment.
func straightSeam(a, b math.Point3) geom.Curve3 { return geom.NewLineSegment(a, b) }

// polylineSeam is a seam carried as a VALUE polyline — the kind a marched section leaves on an edge, and
// the kind `==` panics on.
func polylineSeam(a, b math.Point3) geom.Curve3 {
	return geom.Polyline{Vertices: []math.Point3{a, math.P3(a.X, a.Y, (a.Z+b.Z)/2), b}}
}

// TestDropSeamSlitsRemovesAnEdgeWalkedBothWays: a seam the dissolve orphans bounds nothing, and a loop
// that is nothing but that slit disappears with it.
func TestDropSeamSlitsRemovesAnEdgeWalkedBothWays(t *testing.T) {
	t.Parallel()
	edges := seamWalkedWall(t, straightSeam, false) // rim, up, rim, down
	rim, up, down := edges[0], edges[1], edges[3]
	got := dropSeamSlits([]curvedLoop{{edges: []loopEdge{up, down, rim}}, {edges: []loopEdge{up, down}}})
	if len(got) != 1 || len(got[0].edges) != 1 {
		t.Fatalf("dropSeamSlits gave %v, want one loop of one edge (the rim)", got)
	}
	if got[0].edges[0].source != rim.source {
		t.Error("dropSeamSlits kept the wrong edge")
	}
}

// TestIsReverseTwinReadsTheEdgeIdentity: the slit test compares the source EDGE and the SWAPPED span,
// never a distance and never the curve's value. One edge walked back over exactly the same span IS a
// slit; the same edge over a different span is a boundary and stays; and two edges carrying equal
// curves are two edges — a synthesized copy of the seam's curve is nobody's twin.
func TestIsReverseTwinReadsTheEdgeIdentity(t *testing.T) {
	t.Parallel()
	edges := seamWalkedWall(t, straightSeam, false)
	up, down := edges[1], edges[3]
	if !isReverseTwin(up, down) {
		t.Error("one edge walked both ways is not reported as a slit")
	}
	if isReverseTwin(up, loopEdge{curve: down.curve, t0: (down.t0 + down.t1) / 2, t1: down.t1, source: down.source}) {
		t.Error("a partial reverse run was reported as a slit")
	}
	if isReverseTwin(up, loopEdge{curve: down.curve, t0: down.t0, t1: down.t1}) {
		t.Error("a synthesized edge carrying the seam's curve was reported as the seam's other side")
	}
	if isReverseTwin(loopEdge{curve: up.curve, t0: up.t0, t1: up.t1}, loopEdge{curve: down.curve, t0: down.t0, t1: down.t1}) {
		t.Error("two source-less edges were reported as twins of one another")
	}
}

// TestASlitOfValuePolylinesDropsWithoutAPanic is the regression row for finding 3 of the final fix
// wave: `a.curve == b.curve` on two value Polylines is a run-time panic ("comparing uncomparable type
// geom.Polyline"), and a marched section leaves exactly that on an edge. A seam carried as a value
// polyline and walked both ways must still drop; two DIFFERENT polyline edges beside one another must
// stay — and neither may panic.
func TestASlitOfValuePolylinesDropsWithoutAPanic(t *testing.T) {
	t.Parallel()
	one := seamWalkedWall(t, polylineSeam, false)
	if got := withoutSlitPairs([]loopEdge{one[1], one[3]}); len(got) != 0 {
		t.Errorf("a polyline seam walked both ways left %d edge(s), want the slit gone", len(got))
	}
	two := seamWalkedWall(t, polylineSeam, true)
	if got := withoutSlitPairs([]loopEdge{two[1], two[3]}); len(got) != 2 {
		t.Errorf("two polyline edges of equal shape were dropped as a slit; they are two edges (%d left)", len(got))
	}
}

// TestCutSeamPiecesKeepTheirSource: splitAtSharedRunEnds cuts a seam into pieces, and the pieces of both
// traversals must still name the edge they came from, or a slit several edges deep could never unwind.
func TestCutSeamPiecesKeepTheirSource(t *testing.T) {
	t.Parallel()
	edges := seamWalkedWall(t, straightSeam, false)
	up := edges[1]
	res := geom.ResolutionForSize(10)
	pieces := splitEdgeAtPoints(up, []math.Point3{up.curve.PointAt((up.t0 + up.t1) / 2)}, res)
	if len(pieces) != 2 {
		t.Fatalf("the seam cut at its midpoint gave %d pieces, want 2", len(pieces))
	}
	for i, p := range pieces {
		if p.source != up.source {
			t.Errorf("piece %d lost its source edge", i)
		}
	}
	if r := reverseEdge(up); r.source != up.source {
		t.Error("reverseEdge dropped the source edge")
	}
}

// TestWeldedCutsDropsAStationNamedTwice: one station is named by the rim's end and by the wall edge
// that starts there, and the two arrive as DIFFERENT BITS. Cutting at both mints a zero-length edge,
// because splitEdgeAtPoints's own duplicate test is exact. The near point here is asserted to differ
// bitwise, so the row exercises the weld and not that exactness; the far one is just outside the weld
// and must survive, so the dedup cannot be a blanket collapse.
func TestWeldedCutsDropsAStationNamedTwice(t *testing.T) {
	t.Parallel()
	res := geom.ResolutionForBox(math.BoxFromPoints(math.P3(0, 0, 0), math.P3(3, 3, 3)))
	p := math.P3(1, 2, 3)
	near := math.P3(1+math.Scalar(res.Weld()/2), 2, 3)
	far := math.P3(1+math.Scalar(res.Weld()*8), 2, 3)
	if near == p {
		t.Fatalf("the near station is bit-identical to the first at weld %g; the row would not test the weld", res.Weld())
	}
	got := weldedCuts([]math.Point3{p, near, far}, res)
	if len(got) != 2 {
		t.Fatalf("weldedCuts kept %d points, want 2 (the repeated station welds, the far one does not)", len(got))
	}
	if got[0] != p {
		t.Error("weldedCuts did not keep the FIRST of a welded group; the order must be the caller's")
	}
	if got[1] != far {
		t.Errorf("weldedCuts kept %v as the second point, want the far station %v — a point outside the "+
			"weld is a station of its own", got[1], far)
	}
}

// TestMergedAliasKeysCarriesEveryParentKey: the merged face resolves both parents and everything either
// of them had already absorbed (ADR-0043 verify-on-write, at the source).
func TestMergedAliasKeysCarriesEveryParentKey(t *testing.T) {
	t.Parallel()
	a := curvedFace{lineage: topo.NewLineage(topo.Tok("f", "a", 0)), aliasKeys: [][]byte{[]byte("old-a")}}
	b := curvedFace{lineage: topo.NewLineage(topo.Tok("f", "b", 0)), aliasKeys: [][]byte{[]byte("old-b")}}
	got := mergedAliasKeys(a, b)
	want := [][]byte{[]byte("old-a"), b.lineage.Key(), []byte("old-b")}
	if len(got) != len(want) {
		t.Fatalf("mergedAliasKeys gave %d keys, want %d", len(got), len(want))
	}
	for i := range want {
		if string(got[i]) != string(want[i]) {
			t.Errorf("alias key %d = %q, want %q", i, got[i], want[i])
		}
	}
}
