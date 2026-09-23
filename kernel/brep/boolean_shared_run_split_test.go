// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Unit statements for the shared-run split (boolean_mixed_dissolve_split.go, ADR-0061 stage 5).
//
// The split is what makes a partial shared boundary pairable: a host wall's rim is ONE closed circle
// and the boss seated on it presents a bare ARC of that circle, so each side is cut at the other's
// vertices before the pairing runs. Four predicates decide where, and until now every one of them was
// reached only through a whole boolean — so a row could say "the body came out watertight" and nothing
// could say WHICH station the split chose or why a pair that merely touches is left alone (#3527).

// rimArc is one traversal of the unit-domain circle of radius r, from parameter a to b (the circle's
// domain is [0, 1], not radians). source is nil, which is what a synthesized split edge carries.
func rimArc(t *testing.T, r, a, b float64) loopEdge {
	t.Helper()
	c, err := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), r)
	if err != nil {
		t.Fatalf("NewCircle(r=%g): %v", r, err)
	}
	return loopEdge{curve: c, t0: a, t1: b}
}

// arcFace is a face on the circle's own cylinder whose single loop is the given arcs — the shape the
// split reads, with no more of a face than the predicates touch.
func arcFace(t *testing.T, edges ...loopEdge) curvedFace {
	t.Helper()
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	return curvedFace{surface: cyl, loops: []curvedLoop{{edges: edges}}}
}

// splitRes is the resolution the rows read at: the circle of radius 2 lives well inside a 10 mm span.
func splitRes() geom.Resolution { return geom.ResolutionForSize(10) }

// TestEdgesOverlapSeparatesARunFromATouch is the predicate the whole split rests on: two edges of one
// curve that WALK a common stretch overlap, and two that merely meet at a vertex do not. Getting the
// second wrong cuts a face at a station its neighbours do not carry, which is the defect the file's own
// doc says the restriction exists to avoid.
func TestEdgesOverlapSeparatesARunFromATouch(t *testing.T) {
	t.Parallel()
	res := splitRes()
	half, second := rimArc(t, 2, 0, 0.5), rimArc(t, 2, 0.25, 0.75)
	if !edgesOverlap(half, second, res) {
		t.Error("two arcs sharing a quarter turn do not overlap")
	}
	if !edgesOverlap(second, half, res) {
		t.Error("edgesOverlap is not symmetric on a shared run")
	}
	abutting := rimArc(t, 2, 0.5, 1)
	if edgesOverlap(half, abutting, res) {
		t.Error("two arcs that meet only at their shared end overlap; a touch is not a run")
	}
	inner := rimArc(t, 2, 0.1, 0.2)
	if !edgesOverlap(half, inner, res) {
		t.Error("an arc wholly inside another does not overlap it; the midpoint station exists for this case")
	}
	whole := rimArc(t, 2, 0, 1)
	if !edgesOverlap(whole, second, res) {
		t.Error("a bare arc of a closed rim does not overlap that rim — the configuration the split exists for")
	}
}

// TestEndsInsideSpanIsStrict: the ends test answers for an end that falls INSIDE the other's span and
// not for one that coincides with its end. It is the half of edgesOverlap that catches a pair whose
// midpoints both miss, and a non-strict reading of it would cut every abutting pair at its own joint.
func TestEndsInsideSpanIsStrict(t *testing.T) {
	t.Parallel()
	res := splitRes()
	half, second := rimArc(t, 2, 0, 0.5), rimArc(t, 2, 0.25, 0.75)
	if !endsInsideSpan(half, second, res) {
		t.Error("the second arc's start at 0.25 is inside [0, 0.5] and was not reported")
	}
	if endsInsideSpan(half, rimArc(t, 2, 0.5, 1), res) {
		t.Error("an end coincident with the span's own end was reported as inside it")
	}
	if endsInsideSpan(half, rimArc(t, 2, 0.6, 0.9), res) {
		t.Error("a disjoint arc's ends were reported inside the span")
	}
}

// TestSharedRunEndsNamesOnlyTheOverlappingEdgesStations: the stations handed to the cut are the
// endpoints of the edges that share a run, and nothing else. A face whose boundary only touches the
// other's contributes none, so no face gains a vertex its neighbours do not carry.
func TestSharedRunEndsNamesOnlyTheOverlappingEdgesStations(t *testing.T) {
	t.Parallel()
	res := splitRes()
	boss := arcFace(t, rimArc(t, 2, 0.25, 0.75))
	host := arcFace(t, rimArc(t, 2, 0, 1))
	got := sharedRunEnds(boss, host, res)
	if len(got) != 2 {
		t.Fatalf("the boss arc's run with the host rim named %d stations, want its 2 ends", len(got))
	}
	arc := rimArc(t, 2, 0.25, 0.75)
	for i, want := range []math.Point3{arc.start(), arc.end()} {
		if got[i] != want {
			t.Errorf("station %d = %v, want the arc's own end %v", i, got[i], want)
		}
	}
	touching := arcFace(t, rimArc(t, 2, 0.5, 1))
	if n := len(sharedRunEnds(touching, arcFace(t, rimArc(t, 2, 0, 0.5)), res)); n != 0 {
		t.Errorf("two arcs meeting at one vertex named %d cut stations, want none", n)
	}
}

// TestSplitAtSharedRunEndsCutsTheHostAndLeavesTheBossWhole is the split's own statement, on the exact
// configuration ADR-0061 recorded as left unmerged: the host's closed rim is cut at the boss arc's two
// ends, so the run becomes a whole edge on BOTH sides, and the boss — whose arc already is one — is
// returned unchanged. The host's only station on the boss's span is its own seam at parameter 0, which
// lies outside the arc, so nothing cuts the boss.
//
// The host comes back in THREE arcs, not two: a closed edge already breaks at its own start vertex, so
// two interior cuts leave the shared run plus the two halves of its complement. What the pairing needs
// is not a piece count but that exactly ONE of those pieces is the whole run, and that is asserted
// directly — a count would hold just as well if the cut had landed at two wrong stations.
func TestSplitAtSharedRunEndsCutsTheHostAndLeavesTheBossWhole(t *testing.T) {
	t.Parallel()
	res := splitRes()
	arc := rimArc(t, 2, 0.25, 0.75)
	host, boss := arcFace(t, rimArc(t, 2, 0, 1)), arcFace(t, arc)
	gotHost, gotBoss := splitAtSharedRunEnds(host, boss, res)
	if n := len(gotHost.loops[0].edges); n != 3 {
		t.Errorf("the host's closed rim came back in %d edges, want 3 — the run plus the two halves of "+
			"its complement, which the rim's own start vertex separates", n)
	}
	whole := 0
	for _, e := range gotHost.loops[0].edges {
		if spansTheSameRun(e, arc, res) {
			whole++
		}
	}
	if whole != 1 {
		t.Errorf("%d of the host's pieces walk the boss arc end to end, want exactly 1 — the split's "+
			"whole purpose is that the shared run is one edge on both sides", whole)
	}
	if n := len(gotBoss.loops[0].edges); n != 1 {
		t.Errorf("the boss's arc came back in %d edges, want 1 — it already is the whole shared run", n)
	}
}

// spansTheSameRun reports whether two edges run between the SAME two stations, in either direction.
func spansTheSameRun(a, b loopEdge, res geom.Resolution) bool {
	sameWay := welds(a.start(), b.start(), res) && welds(a.end(), b.end(), res)
	return sameWay || (welds(a.start(), b.end(), res) && welds(a.end(), b.start(), res))
}

// welds reports whether two points are one point at this resolution.
func welds(p, q math.Point3, res geom.Resolution) bool {
	return float64(p.DistanceTo(q)) <= res.Weld()
}

// TestSplitAtSharedRunEndsLeavesATouchingPairAlone is the negative row: two arcs that meet at a vertex
// share no run, so neither is cut. A split that fired on a touch would hand the stitch a vertex one
// side carries and the other does not.
func TestSplitAtSharedRunEndsLeavesATouchingPairAlone(t *testing.T) {
	t.Parallel()
	res := splitRes()
	a, b := arcFace(t, rimArc(t, 2, 0, 0.5)), arcFace(t, rimArc(t, 2, 0.5, 1))
	gotA, gotB := splitAtSharedRunEnds(a, b, res)
	if n, m := len(gotA.loops[0].edges), len(gotB.loops[0].edges); n != 1 || m != 1 {
		t.Errorf("a touching pair was cut into %d and %d edges, want 1 and 1", n, m)
	}
}
