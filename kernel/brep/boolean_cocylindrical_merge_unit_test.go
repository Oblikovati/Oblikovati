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
	loops, ok := dissolveSharedEdges(a, b, res)
	if !ok {
		t.Fatal("two bands meeting at a rim share that rim; the dissolve declined")
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
	if _, ok := dissolveSharedEdges(a, b, res); ok {
		t.Error("two separated bands were merged; they share no edge")
	}
}

// TestSharedEdgeTwinsPairsWholeEdgesOnly: the pairing is one-to-one, and the run a boss shares with a
// host rim reaches it whole because splitAtSharedRunEnds cut both sides at the other's vertices first.
func TestSharedEdgeTwinsPairsWholeEdgesOnly(t *testing.T) {
	t.Parallel()
	a, b := wallFaceOf(t, math.P3(0, 0, 0), 2, 4), wallFaceOf(t, math.P3(0, 0, 4), 2, 3)
	res := geom.ResolutionForBox(faceLoopBox(a).Union(faceLoopBox(b)))
	twin, ok := sharedEdgeTwins(a, b, res)
	if !ok {
		t.Fatal("the abutting bands' shared rim was not paired")
	}
	if len(twin) != 2 { // the pair is recorded from both ends
		t.Errorf("the pairing holds %d entries, want 2 (one edge pair, both ways)", len(twin))
	}
}

// TestDropSeamSlitsRemovesACurveWalkedBothWays: a seam the dissolve orphans bounds nothing, and a loop
// that is nothing but that slit disappears with it.
func TestDropSeamSlitsRemovesACurveWalkedBothWays(t *testing.T) {
	t.Parallel()
	seg := geom.NewLineSegment(math.P3(0, 0, 0), math.P3(0, 0, 1))
	circle := geom.Circle{Center: math.P3(0, 0, 0), Normal: math.V3(0, 0, 1).AsUnit(),
		RefDir: math.V3(1, 0, 0).AsUnit(), Radius: 2}
	up, down := loopEdge{curve: seg, t0: 0, t1: 1}, loopEdge{curve: seg, t0: 1, t1: 0}
	rim := loopEdge{curve: circle, t0: 0, t1: 1}
	got := dropSeamSlits([]curvedLoop{{edges: []loopEdge{up, down, rim}}, {edges: []loopEdge{up, down}}})
	if len(got) != 1 || len(got[0].edges) != 1 {
		t.Fatalf("dropSeamSlits gave %v, want one loop of one edge (the rim)", got)
	}
	if got[0].edges[0].curve != circle {
		t.Error("dropSeamSlits kept the wrong edge")
	}
}

// TestIsReverseTwinIsExact: the slit test compares the curve and the SWAPPED span, never a distance.
// Curve equality is by value, which is the right question — one curve walked back over exactly the same
// span IS a slit, whichever loop copy carries it — while a curve of a different shape, or the same
// curve over a different span, is a boundary and stays.
func TestIsReverseTwinIsExact(t *testing.T) {
	t.Parallel()
	seg := geom.NewLineSegment(math.P3(0, 0, 0), math.P3(0, 0, 1))
	other := geom.NewLineSegment(math.P3(0, 0, 0), math.P3(0, 1, 0))
	up := loopEdge{curve: seg, t0: 0, t1: 1}
	if !isReverseTwin(up, loopEdge{curve: seg, t0: 1, t1: 0}) {
		t.Error("one segment walked both ways is not reported as a slit")
	}
	if isReverseTwin(up, loopEdge{curve: seg, t0: 0.5, t1: 0}) {
		t.Error("a partial reverse run was reported as a slit")
	}
	if isReverseTwin(up, loopEdge{curve: other, t0: 1, t1: 0}) {
		t.Error("a different curve was reported as this seam's other side")
	}
}

// TestWeldedCutsDropsAStationNamedTwice: one station is named by the rim's end and by the wall edge
// that starts there, and the two arrive as different bits. Cutting at both mints a zero-length edge.
func TestWeldedCutsDropsAStationNamedTwice(t *testing.T) {
	t.Parallel()
	res := geom.ResolutionForBox(math.BoxFromPoints(math.P3(0, 0, 0), math.P3(3, 3, 3)))
	p := math.P3(1, 2, 3)
	near := math.P3(1+1e-16, 2, 3)
	got := weldedCuts([]math.Point3{p, near, math.P3(0, 0, 0)}, res)
	if len(got) != 2 {
		t.Fatalf("weldedCuts kept %d points, want 2 (the repeated station welds)", len(got))
	}
	if got[0] != p {
		t.Error("weldedCuts did not keep the FIRST of a welded group; the order must be the caller's")
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
