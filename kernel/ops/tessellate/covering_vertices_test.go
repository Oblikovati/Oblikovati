// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"testing"

	"oblikovati.org/math"
)

// flatCover is a covering accumulator over a plane whose normal is +Z and whose metric is (2, 3), so a
// test can tell the scaled (u,v) from the raw one.
func flatCover(carry func(u, v float64) bool) *coverVertices {
	return &coverVertices{
		normalAt: func(_, _ float64) math.Vector3 { return math.V3(0, 0, 1) },
		carry:    carry, su: 2, sv: 3,
	}
}

// unitChain is a three-point chain with its first point repeated, the shape a lifted boundary loop has.
func unitChain() ([]math.Point3, []math.Point2) {
	p3 := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0), math.P3(0, 0, 0)}
	uv := []math.Point2{math.P2(0, 0), math.P2(1, 0), math.P2(1, 1), math.P2(0, 0)}
	return p3, uv
}

// TestACoveringVertexKeepsBothItsParameters: the triangulation runs in the METRIC-scaled (u,v) and the
// canonical selection reads the RAW one, so the accumulator has to carry both.
func TestACoveringVertexKeepsBothItsParameters(t *testing.T) {
	t.Parallel()
	c := flatCover(nil)
	i := c.add(math.P3(7, 8, 9), 1.5, 2.5)
	if i != 0 || len(c.pos) != 1 {
		t.Fatalf("add returned index %d with %d positions, want the first vertex", i, len(c.pos))
	}
	if c.xy[0] != [2]float64{3, 7.5} {
		t.Errorf("scaled (u,v) = %v, want the metric (2,3) applied", c.xy[0])
	}
	if c.uu[0] != 1.5 || c.vv[0] != 2.5 {
		t.Errorf("raw (u,v) = (%g,%g), want it unscaled", c.uu[0], c.vv[0])
	}
	if c.nrm[0] != math.V3(0, 0, 1) {
		t.Errorf("normal = %v, want the owner's folded normal", c.nrm[0])
	}
}

// TestACarryGateDropsAReplicaAndItsSegments: a covering that carries only the points near its window
// must not constrain a segment with an end outside it — that segment's own replica carries it.
func TestACarryGateDropsAReplicaAndItsSegments(t *testing.T) {
	t.Parallel()
	c := flatCover(func(_, v float64) bool { return v < 0.5 }) // drops the chain's third point only
	p3, uv := unitChain()
	segs := c.addChain(p3, uv, 0, 0)
	if len(c.pos) != 3 {
		t.Errorf("the covering carries %d of 4 chain points, want the 3 the gate accepts", len(c.pos))
	}
	if len(segs) != 1 || segs[0][0] != 0 || segs[0][1] != 1 {
		t.Errorf("addChain constrained %v, want only the segment with both ends carried", segs)
	}
}

// TestAnUngatedChainConstrainsEverySegment is the other side: with no gate, a chain of n points
// constrains its n−1 segments and no closing edge, which is what an open covering chain needs.
func TestAnUngatedChainConstrainsEverySegment(t *testing.T) {
	t.Parallel()
	c := flatCover(nil)
	p3, uv := unitChain()
	if segs := c.addChain(p3, uv, 0, 0); len(segs) != 3 {
		t.Errorf("addChain constrained %d segments, want 3 for a 4-point chain", len(segs))
	}
}

// TestARingIsOneLoopConstraint: a chain that DOES close in the covering space goes in as a loop, whose
// closing edge the triangulation supplies.
func TestARingIsOneLoopConstraint(t *testing.T) {
	t.Parallel()
	c := flatCover(nil)
	p3, uv := unitChain()
	idx := c.addRing(p3[:3], uv[:3], 10, 20)
	if len(idx) != 3 {
		t.Fatalf("addRing returned %d indices, want 3", len(idx))
	}
	if c.uu[0] != 10 || c.vv[0] != 20 {
		t.Errorf("the ring landed at (%g,%g), want the whole-period offset applied", c.uu[0], c.vv[0])
	}
}

// TestTheCanonicalSelectionReadsTheCentroid: a triangle belongs to the covering's canonical copy by
// where its CENTROID is, which is what makes exactly one replica of a seam-spanning triangle survive.
func TestTheCanonicalSelectionReadsTheCentroid(t *testing.T) {
	t.Parallel()
	c := flatCover(nil)
	for _, uv := range [][2]float64{{0, 0}, {3, 0}, {0, 3}, {9, 9}} {
		c.add(math.P3(0, 0, 0), uv[0], uv[1])
	}
	if u, v := c.centroid([3]int{0, 1, 2}); u != 1 || v != 1 {
		t.Errorf("centroid = (%g,%g), want (1,1)", u, v)
	}
	kept := c.keepCanonical([][3]int{{0, 1, 2}, {1, 2, 3}}, func(u, v float64) bool { return u < 2 && v < 2 })
	if len(kept) != 1 || kept[0] != [3]int{0, 1, 2} {
		t.Errorf("keepCanonical kept %v, want only the triangle whose centroid the predicate accepts", kept)
	}
}

// TestALoopsParametersPairIntoPoints: the periodic band's loops keep u and v in parallel slices; the
// covering accumulator lays out points, so they pair on the way in.
func TestALoopsParametersPairIntoPoints(t *testing.T) {
	t.Parallel()
	l := cylLoop{p3: make([]math.Point3, 2), u: []float64{1, 2}, v: []float64{3, 4}}
	got := l.uvPoints()
	if len(got) != 2 || got[0] != math.P2(1, 3) || got[1] != math.P2(2, 4) {
		t.Errorf("uvPoints = %v, want the parallel slices paired", got)
	}
	if n := len(cylLoop{}.uvPoints()); n != 0 {
		t.Errorf("uvPoints of an empty loop = %d points, want none", n)
	}
}
