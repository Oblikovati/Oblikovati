// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestSpliceCornerBite_RemovesSmallAreaNotFewestSegs is the geometric-criterion regression (Important
// #3): spliceCornerBite must remove the SMALLER-AREA span the bite arc cuts off, NOT the fewer-segment
// one. The fixture is built so the two criteria DISAGREE — the small bitten corner (near the origin) is
// cut into FOUR sub-edges while the large surviving span is a single two-edge triangle, so the retired
// len(fwd)<=len(bwd) proxy would splice out the LARGE region (keeping the tiny corner) and the area
// criterion splices out the tiny corner (keeping the large region). Asserting the result keeps the far
// vertex F and drops the corner vertex P0 fails loudly if the count proxy ever returns.
func TestSpliceCornerBite_RemovesSmallAreaNotFewestSegs(t *testing.T) {
	t.Parallel()
	b0, b1 := math.P3(2, 0, 0), math.P3(0, 2, 0)
	a, p0, b := math.P3(1, 0, 0), math.P3(0, 0, 0), math.P3(0, 1, 0)
	far := math.P3(20, 20, 0)
	// ring: small corner b0→a→p0→b→b1 (4 segs, area 2), then large span b1→far→b0 (2 segs, area 38).
	ring := []endSeg{
		straightSeg(b0, a), straightSeg(a, p0), straightSeg(p0, b), straightSeg(b, b1),
		straightSeg(b1, far), straightSeg(far, b0),
	}
	bite := biteArc(t, b0, math.P3(1.2, 1.2, 0), b1)
	out, ok := spliceCornerBite(ring, bite, 1e-6)
	if !ok {
		t.Fatalf("spliceCornerBite declined a well-formed corner bite")
	}
	if !ringHasPoint(out, far) {
		t.Fatalf("result dropped the far vertex %v — the LARGE region was spliced out (count proxy chose wrong span)", far)
	}
	if ringHasPoint(out, p0) {
		t.Fatalf("result kept the corner vertex %v — the SMALL corner was not bitten off (count proxy chose wrong span)", p0)
	}
	if !ringEndsWithBite(out, bite) {
		t.Fatalf("result does not close on the bite arc (kept span not sealed by the bite)")
	}
}

// TestCornerBiteArea_MatchesTriangle certifies the area helper on a known planar triangle: the region a
// two-edge span (b1→far→b0) encloses with a straight closing chord must equal the shoelace triangle
// area (38), so the splice's small-vs-large decision rests on a correct measure, not the sampling noise.
func TestCornerBiteArea_MatchesTriangle(t *testing.T) {
	t.Parallel()
	b0, b1, far := math.P3(2, 0, 0), math.P3(0, 2, 0), math.P3(20, 20, 0)
	span := []endSeg{straightSeg(b1, far), straightSeg(far, b0)}
	chord := straightSeg(b0, b1) // straight closing edge → exact triangle, no arc sampling
	if got := cornerBiteArea(span, chord); got < 37.999 || got > 38.001 {
		t.Fatalf("cornerBiteArea = %.6f, want 38 (triangle (0,2)-(20,20)-(2,0))", got)
	}
}

// straightSeg is a straight boundary edge from a to b (no curve).
func straightSeg(from, to math.Point3) endSeg { return endSeg{from: from, to: to} }

// biteArc builds a bite endSeg carrying the arc through the three points (from, mid, to).
func biteArc(t *testing.T, from, mid, to math.Point3) endSeg {
	t.Helper()
	arc, err := geom.Arc3dByThreePoints(from, mid, to)
	if err != nil {
		t.Fatalf("build bite arc: %v", err)
	}
	return endSeg{from: from, to: to, curve: arc, mid: mid, arc: true}
}

// ringHasPoint reports whether p is one of the ring's segment endpoints.
func ringHasPoint(ring []endSeg, p math.Point3) bool {
	for _, s := range ring {
		if float64(s.from.DistanceTo(p)) < 1e-6 || float64(s.to.DistanceTo(p)) < 1e-6 {
			return true
		}
	}
	return false
}

// ringEndsWithBite reports whether the ring's last segment is the bite arc (same endpoints, curved).
func ringEndsWithBite(ring []endSeg, bite endSeg) bool {
	last := ring[len(ring)-1]
	if !last.arc {
		return false
	}
	near := float64(last.from.DistanceTo(bite.from)) < 1e-6 && float64(last.to.DistanceTo(bite.to)) < 1e-6
	rev := float64(last.from.DistanceTo(bite.to)) < 1e-6 && float64(last.to.DistanceTo(bite.from)) < 1e-6
	return near || rev
}
