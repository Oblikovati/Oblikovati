// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"strings"
	"testing"

	"oblikovati.org/math"
)

// TestASlitIsNoRimSegment: a boundary that walks a segment in BOTH directions — the artificial seam
// the piston head's merged wall carries up and back — bounds material on both sides, so neither
// direction may constrain a triangle. The two traversals are SEPARATE covering vertices at the same
// 3D point, which is why the count is taken on the weld and not on the index pair (#3518).
func TestASlitIsNoRimSegment(t *testing.T) {
	t.Parallel()
	b, chains := slitCover(t)
	got := b.directedRimSegments([]rimSegment{{0, 1, 0}, {1, 2, 0}, {3, 4, 0}, {4, 5, 0}}, chains, weldGrid([][]math.Point3{b.pos}))
	if len(got) != 0 {
		t.Errorf("directedRimSegments kept %d segment(s) of a boundary that is entirely slit; want 0 — "+
			"the two traversals are different covering vertices at the same 3D points", len(got))
	}
}

// TestEachChainKeepsItsOwnSegments: two chains laid into one covering are told apart, so their
// material sides are voted separately — the fixture wall's window hole is wound the other way from
// its rims and a single per-face bit threw it away (#3518).
func TestEachChainKeepsItsOwnSegments(t *testing.T) {
	t.Parallel()
	b, chains := openChainCover(t)
	got := b.directedRimSegments([]rimSegment{{0, 1, 0}, {1, 2, 0}, {2, 3, 1}}, chains, weldGrid([][]math.Point3{b.pos}))
	if got[[2]int{0, 1}] != 0 || got[[2]int{1, 2}] != 0 {
		t.Errorf("chain 0's segments are filed under %d/%d; want 0", got[[2]int{0, 1}], got[[2]int{1, 2}])
	}
	if got[[2]int{2, 3}] != 1 {
		t.Errorf("chain 1's segment is filed under %d; want 1", got[[2]int{2, 3}])
	}
}

// slitCover holds one three-point run laid TWICE, forward then backward, at the same 3D points — the
// slit shape, and the chain that says so.
func slitCover(t *testing.T) (*chartCover, []chartChain) {
	t.Helper()
	b := newBareCover(t)
	p := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(2, 0, 0)}
	order := []int{0, 1, 2, 2, 1, 0}
	run := make([]math.Point3, 0, len(order))
	for _, i := range order {
		b.add(p[i], float64(i), 0, 0)
		run = append(run, p[i])
	}
	return b, []chartChain{{p3: run}}
}

// openChainCover holds four distinct 3D points in a row, each segment walked once, split into two
// chains that share the middle point.
func openChainCover(t *testing.T) (*chartCover, []chartChain) {
	t.Helper()
	b := newBareCover(t)
	run := make([]math.Point3, 0, 4)
	for i := range 4 {
		p := math.P3(float64(i), 0, 0)
		b.add(p, float64(i), 0, 0)
		run = append(run, p)
	}
	return b, []chartChain{{p3: run[:3]}, {p3: run[2:]}}
}

// newBareCover is a covering with nothing but the metric and a normal, for the segment-bookkeeping rows.
func newBareCover(t *testing.T) *chartCover {
	t.Helper()
	b := &chartCover{}
	b.normalAt = func(float64, float64) math.Vector3 { return math.V3(0, 0, 1) }
	b.su, b.sv = 1, 1
	return b
}

// TestOnlyAUnanimousChainGetsASide: the side is READ from the chart, so a chain the chart never
// bounded exactly once has no side — and a chain whose decisive segments name BOTH sides has none
// either. A loop is wound consistently, so a disagreement means the chart is wrong about that chain in
// a way no count repairs, and binding it to the majority would ship the wrong side confidently past a
// rim gate that cannot see it (#3518 review I4).
func TestOnlyAUnanimousChainGetsASide(t *testing.T) {
	t.Parallel()
	s := rimSideFrom([]int{3, 0, 2, 0}, []int{0, 0, 2, 5})
	for _, c := range []struct {
		ci            int
		left, decided bool
		why           string
	}{
		{0, true, true, "unanimous left"},
		{1, false, false, "no decisive segment"},
		{2, true, false, "both sides named"},
		{3, false, true, "unanimous right"},
	} {
		if s.left[c.ci] != c.left || s.decided[c.ci] != c.decided {
			t.Errorf("chain %d (%s): left=%v decided=%v; want %v/%v", c.ci, c.why, s.left[c.ci], s.decided[c.ci], c.left, c.decided)
		}
	}
}

// TestADisagreeingChainIsRefusedByName: the conflict is not silence. It carries the chain and both
// counts, so the router's decline says what to look at.
func TestADisagreeingChainIsRefusedByName(t *testing.T) {
	t.Parallel()
	got := rimSideFrom([]int{0, 7}, []int{0, 4}).conflict
	for _, want := range []string{"chain 1", "7 segments", "4 say right"} {
		if !strings.Contains(got, want) {
			t.Errorf("the refusal %q does not name %q — a reader cannot act on it", got, want)
		}
	}
	if quiet := rimSideFrom([]int{3, 0}, []int{0, 0}).conflict; quiet != "" {
		t.Errorf("a unanimous corpus produced the refusal %q; it must be empty", quiet)
	}
}

// TestDirectedTriangleEdgesReadCounterClockwise pins the convention the whole rule rests on: a CCW
// triangle lies to the LEFT of each edge it names, so carrying (a→b) is the same statement as
// "material is on the left of a→b".
func TestDirectedTriangleEdgesReadCounterClockwise(t *testing.T) {
	t.Parallel()
	got := directedTriangleEdges([3]int{4, 5, 6})
	want := [3][2]int{{4, 5}, {5, 6}, {6, 4}}
	if got != want {
		t.Errorf("directedTriangleEdges = %v, want %v (the traversal order, closing back to the first)", got, want)
	}
}

// TestOnlyKeptTrianglesVote: the side is read off the chart's own verdict, so a triangle the chart
// dropped may not cast a vote.
func TestOnlyKeptTrianglesVote(t *testing.T) {
	t.Parallel()
	tris := [][3]int{{0, 1, 2}, {1, 0, 3}}
	got := keptDirectedEdgeUse(tris, []bool{true, false})
	if got[[2]int{0, 1}] != 1 || got[[2]int{1, 0}] != 0 {
		t.Errorf("edge use = %d forward / %d back; want 1/0 — the dropped triangle voted", got[[2]int{0, 1}], got[[2]int{1, 0}])
	}
}

// TestAContradictionRefusesTheFaceAndOutranksTheOthers drives the wiring the conflict string exists
// for, which nothing did before (#3518 review N2): rimSideFrom raises it, bindToTheRim copies it onto
// the covering, and meshOrRefusal turns it into the reason chartRegionMesh returns and the router
// reports. No corpus body produces a contradiction — measured unanimous everywhere — so without this
// row the path is written and never executed.
func TestAContradictionRefusesTheFaceAndOutranksTheOthers(t *testing.T) {
	t.Parallel()
	b, _ := openChainCover(t)
	b.rimSideConflict = "its boundary chain 7 names both sides as material"
	kept, why := b.meshOrRefusal([][3]int{{0, 1, 2}}, true)
	if kept != nil || why != b.rimSideConflict {
		t.Errorf("a covering carrying a contradiction meshed %v with reason %q; want no mesh and the "+
			"contradiction itself", kept, why)
	}
	// It outranks the other two refusals, which are its symptoms rather than reasons of their own.
	if _, spent := b.meshOrRefusal(nil, false); spent != b.rimSideConflict {
		t.Errorf("a spent ear budget reported %q over the contradiction that caused it", spent)
	}
	if _, empty := b.meshOrRefusal(nil, true); empty != b.rimSideConflict {
		t.Errorf("an empty kept set reported %q over the contradiction that caused it", empty)
	}
}

// TestAUnanimousCoveringRaisesNoRefusal is the other direction: the ordinary covering must still get
// its own two reasons, so the row above pins an override and not a swallow.
func TestAUnanimousCoveringRaisesNoRefusal(t *testing.T) {
	t.Parallel()
	b, _ := openChainCover(t)
	if kept, why := b.meshOrRefusal([][3]int{{0, 1, 2}}, true); why != "" || len(kept) != 1 {
		t.Errorf("an unrefused covering came back with %d triangle(s) and reason %q; want 1 and none", len(kept), why)
	}
	if _, spent := b.meshOrRefusal(nil, false); !strings.Contains(spent, "ear still") {
		t.Errorf("a spent ear budget reported %q; want the ear reason", spent)
	}
	if _, empty := b.meshOrRefusal(nil, true); !strings.Contains(empty, "kept no triangle") {
		t.Errorf("an empty kept set reported %q; want the empty reason", empty)
	}
}

// TestBindToTheRimCarriesTheContradictionOntoTheCovering closes the first hop of that path: a chain
// whose decisive segments disagree must leave its refusal ON the covering, where meshOrRefusal reads
// it. Two triangles traverse the same rim segment in opposite directions, so chain 0 counts one vote
// each way.
func TestBindToTheRimCarriesTheContradictionOntoTheCovering(t *testing.T) {
	t.Parallel()
	b, chains := contradictingCover(t)
	b.chains = 1
	b.rimChain = b.directedRimSegments([]rimSegment{{0, 1, 0}, {2, 3, 0}}, chains, weldGrid([][]math.Point3{b.pos}))
	tris := [][3]int{{0, 1, 4}, {3, 2, 5}}
	b.bindToTheRim(tris, []bool{true, true})
	if !strings.Contains(b.rimSideConflict, "chain 0") {
		t.Errorf("bindToTheRim left %q on the covering; want a refusal naming chain 0", b.rimSideConflict)
	}
}

// contradictingCover is a covering holding two rim segments of ONE chain whose kept triangles bound
// them from opposite sides — the disagreement no consistently wound loop can produce.
func contradictingCover(t *testing.T) (*chartCover, []chartChain) {
	t.Helper()
	b := newBareCover(t)
	pts := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(2, 0, 0), math.P3(3, 0, 0),
		math.P3(0, 1, 0), math.P3(3, 1, 0)}
	for i, p := range pts {
		b.add(p, float64(i), 0, 0)
	}
	return b, []chartChain{{p3: []math.Point3{pts[0], pts[1]}}, {p3: []math.Point3{pts[2], pts[3]}}}
}
