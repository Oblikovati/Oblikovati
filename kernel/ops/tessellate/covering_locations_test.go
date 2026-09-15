// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// The covering's ONE-LOCATION-ONE-VERTEX invariant (Oblikovati/Oblikovati#3551).
//
// A covering is the 2D point set a constrained triangulation runs on. Two vertices at one location are
// indistinguishable to every orientation predicate the constraint recovery walks with, so a constraint
// incident to either may simply not be recovered — silently, and differently on different platforms.
// The unit rows below hold the merge itself; the corpus row that holds it on real bodies is
// TestNoChartedCoveringLaysTwoVerticesAtOneLocation (chart_face_mesh_test.go).

// newTestCover is a bare accumulator with a unit metric, for the rows that exercise the merge alone.
func newTestCover() *coverVertices {
	c := &coverVertices{su: 1, sv: 1}
	c.normalAt = func(u, v float64) math.Vector3 { return math.V3(0, 0, 1) }
	return c
}

// TestWithoutTheMergeEveryCallLaysANewVertex is the control: the merge is opt-in, and an accumulator
// that was never asked for it behaves exactly as it did before (#3551).
func TestWithoutTheMergeEveryCallLaysANewVertex(t *testing.T) {
	t.Parallel()
	c := newTestCover()
	a := c.add(math.P3(1, 2, 3), 0.5, 0.25, 0)
	b := c.add(math.P3(1, 2, 3), 0.5, 0.25, 1)
	if a == b {
		t.Errorf("an un-asked accumulator merged two vertices at one location (%d == %d)", a, b)
	}
	if len(c.pos) != 2 {
		t.Errorf("the accumulator holds %d vertices, want 2", len(c.pos))
	}
}

// TestTheMergeReturnsTheVertexAlreadyAtThatLocation is the merge's own row, over the three ways one
// location arrives twice: bit-identically, a last place apart, and from a different period shift.
func TestTheMergeReturnsTheVertexAlreadyAtThatLocation(t *testing.T) {
	t.Parallel()
	for _, row := range []struct {
		name string
		u, v float64
	}{
		{"bit-identical", 0.5, 0.25},
		{"one ulp apart", stdmath.Nextafter(0.5, 1), 0.25},
		{"a few ulps apart in both axes", stdmath.Nextafter(stdmath.Nextafter(0.5, 1), 1), stdmath.Nextafter(0.25, 0)},
	} {
		c := newTestCover()
		c.MergeCoincidentLocations(1e-9)
		first := c.add(math.P3(1, 2, 3), 0.5, 0.25, 0)
		again := c.add(math.P3(1, 2, 3), row.u, row.v, 2)
		if again != first {
			t.Errorf("%s: the merge laid a second vertex (%d) at the location of %d", row.name, again, first)
		}
		if len(c.pos) != 1 {
			t.Errorf("%s: the accumulator holds %d vertices, want 1", row.name, len(c.pos))
		}
		if c.at[first] != 0 {
			t.Errorf("%s: the merged vertex took the later shift index %d, want the one that laid it", row.name, c.at[first])
		}
	}
}

// TestTheMergeKeepsApartWhatIsFurtherThanTheWeld is the other direction, and it is what says the merge
// is a weld and not a grid snap: a pair one weld apart is two vertices, not one.
func TestTheMergeKeepsApartWhatIsFurtherThanTheWeld(t *testing.T) {
	t.Parallel()
	c := newTestCover()
	c.MergeCoincidentLocations(1e-9)
	first := c.add(math.P3(0, 0, 0), 0, 0, 0)
	apart := c.add(math.P3(0, 0, 0), 1e-8, 0, 0)
	if apart == first {
		t.Error("the merge joined two locations ten welds apart")
	}
	if len(c.pos) != 2 {
		t.Errorf("the accumulator holds %d vertices, want 2", len(c.pos))
	}
}

// TestTheMergeFindsALocationAcrossACellBoundary is why locationOf scans the 3x3 block of cells rather
// than one: two values of a single location computed two ways land either side of a quantisation edge
// about as often as not, and a lookup that read only its own cell would miss half the duplicates.
func TestTheMergeFindsALocationAcrossACellBoundary(t *testing.T) {
	t.Parallel()
	c := newTestCover()
	c.MergeCoincidentLocations(1e-9)
	onEdge := 7e-9 // exactly seven cells along, so the pair below straddles the boundary
	first := c.add(math.P3(0, 0, 0), onEdge, 0, 0)
	under := c.add(math.P3(0, 0, 0), stdmath.Nextafter(onEdge, 0), 0, 0)
	if under != first {
		t.Errorf("a location a last place BELOW a cell boundary laid a new vertex (%d) instead of "+
			"finding %d", under, first)
	}
}

// TestASelfLoopIsNoConstraint holds the other half of the merge: once two ends of a boundary step are
// one vertex, that step is not a segment a triangulation can hold, and chainConstraints drops it.
func TestASelfLoopIsNoConstraint(t *testing.T) {
	t.Parallel()
	got := chainConstraints([]int{4, 5, 5, 6, -1, 7})
	want := [][]int{{4, 5}, {5, 6}}
	if len(got) != len(want) {
		t.Fatalf("chainConstraints gave %v, want %v", got, want)
	}
	for i := range want {
		if got[i][0] != want[i][0] || got[i][1] != want[i][1] {
			t.Errorf("segment %d is %v, want %v", i, got[i], want[i])
		}
	}
}
