// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Regression for Oblikovati#2074. coplanarOverlap reported ANY two coplanar triangles that
// touched — including the edge contact every face makes with its coplanar neighbour — and
// returned the first triangle's CENTRE as the witness. The caller filters legitimate contact by
// asking whether the witness lies on the two faces' shared boundary, and a centre never does,
// so the filter could not fire. Every sheet-metal wall makes such a pair where its end cap meets
// the sheet's side face, so a plain flange reported four self-intersections on a solid whose
// topology was perfect.

// TestCoplanarEdgeContactIsNotAnOverlap: triangles sharing only an edge overlap by no area.
func TestCoplanarEdgeContactIsNotAnOverlap(t *testing.T) {
	p := math.P3
	left := [3]math.Point3{p(0, 0, 0), p(0, 1, 0), p(0, 0, 1)}
	right := [3]math.Point3{p(0, 0, 0), p(0, 0, 1), p(0, -1, 0)}
	if w, hit := trianglesIntersect(left, right); hit {
		t.Errorf("coplanar triangles sharing only an edge report an intersection at %v", w)
	}
}

// TestCoplanarCornerContactIsNotAnOverlap: touching at a single corner is contact too.
func TestCoplanarCornerContactIsNotAnOverlap(t *testing.T) {
	p := math.P3
	a := [3]math.Point3{p(0, 0, 0), p(0, 1, 0), p(0, 0, 1)}
	b := [3]math.Point3{p(0, 0, 0), p(0, -1, 0), p(0, 0, -1)}
	if w, hit := trianglesIntersect(a, b); hit {
		t.Errorf("coplanar triangles meeting at one corner report an intersection at %v", w)
	}
}

// TestCoplanarOverlapWitnessLiesInTheSharedRegion: a real overlap is still reported, and the
// witness has to sit INSIDE the region the two triangles share — not at either triangle's
// centre, which is what made the caller's shared-boundary filter useless.
func TestCoplanarOverlapWitnessLiesInTheSharedRegion(t *testing.T) {
	p := math.P3
	big := [3]math.Point3{p(0, 0, 0), p(0, 6, 0), p(0, 0, 6)}
	// Wholly inside big, and deliberately far from big's centre (0, 2, 2): a witness taken from
	// the containing triangle instead of the shared region lands outside this corner.
	corner := [3]math.Point3{p(0, 4, 0), p(0, 6, 0), p(0, 4, 1)}
	w, hit := trianglesIntersect(big, corner)
	if !hit {
		t.Fatal("a genuine coplanar overlap was missed")
	}
	if y, z := float64(w.Y), float64(w.Z); y < 4 || z > 1 {
		t.Errorf("witness %v is outside the shared region (y >= 4 and z <= 1)", w)
	}
}

// TestCoplanarOverlapIsWindingIndependent: the same pair must be judged the same however the
// triangles are wound, since a tessellator is free to emit either.
func TestCoplanarOverlapIsWindingIndependent(t *testing.T) {
	p := math.P3
	big := [3]math.Point3{p(0, 0, 0), p(0, 4, 0), p(0, 0, 4)}
	small := [3]math.Point3{p(0, 1, 1), p(0, 2, 1), p(0, 1, 2)}
	flipped := [3]math.Point3{small[0], small[2], small[1]}
	_, hitA := trianglesIntersect(big, small)
	_, hitB := trianglesIntersect(big, flipped)
	if !hitA || !hitB {
		t.Errorf("a contained overlap read as hit=%v wound one way and %v the other", hitA, hitB)
	}
}

// TestDegenerateTriangleSharesNothing: a tessellator can emit a sliver whose three points are
// collinear. It has no area, so the share ratio would divide by zero and compare false — which
// reads as an overlap. Such a triangle can share nothing with anything.
func TestDegenerateTriangleSharesNothing(t *testing.T) {
	p := math.P3
	real3 := [3]math.Point3{p(0, 0, 0), p(0, 4, 0), p(0, 0, 4)}
	collinear := [3]math.Point3{p(0, 1, 1), p(0, 2, 2), p(0, 3, 3)} // inside real3, but no area
	if w, hit := trianglesIntersect(real3, collinear); hit {
		t.Errorf("a zero-area triangle reported an overlap at %v", w)
	}
}

// TestCoplanarNeighbourFacesAreClean is the body-level regression: two coplanar quads meeting
// along a shared edge — the shape a sheet-metal wall's end cap makes with the sheet's side —
// must report no self-intersection.
func TestCoplanarNeighbourFacesAreClean(t *testing.T) {
	p := math.P3
	left := quadBody("left", p(0, 0, 0), p(0, 2, 0), p(0, 2, 2), p(0, 0, 2))
	right := quadBody("right", p(0, 2, 0), p(0, 4, 0), p(0, 4, 2), p(0, 2, 2))
	merged := topo.MergeBodies(topo.NewLineage(topo.Tok("imp", "body", 0)), false, left, right)
	if hits := SelfIntersections(merged, DefaultQuality()); len(hits) != 0 {
		t.Errorf("coplanar faces meeting along an edge report %d self-intersection(s): %+v",
			len(hits), hits)
	}
}

// TestCoplanarShareThresholdHasAPlateau justifies coplanarShareRatio. Contact overlaps by
// rounding noise and a real overlap by a large fraction of a triangle, so the verdict must be
// the same for every threshold across many orders of magnitude. A constant that only worked at
// its own value would be tuned to the fixture rather than derived from the problem.
func TestCoplanarShareThresholdHasAPlateau(t *testing.T) {
	clip := [3][2]float64{{0, 0}, {4, 0}, {0, 4}}
	contact := [3][2]float64{{0, 0}, {0, -4}, {4, -4}}      // shares only the corner run
	overlap := [3][2]float64{{1, 1}, {3, 1}, {1, 3}}        // wholly inside
	sliver := [3][2]float64{{0.1, 0.1}, {2, 0.1}, {2, 0.2}} // small but real
	for _, ratio := range []float64{1e-12, 1e-10, 1e-9, 1e-8, 1e-6} {
		if got := shareIsDegenerate(clip, contact, ratio); !got {
			t.Errorf("at ratio %g, edge contact was read as an overlap", ratio)
		}
		if got := shareIsDegenerate(clip, overlap, ratio); got {
			t.Errorf("at ratio %g, a contained triangle was read as contact", ratio)
		}
		if got := shareIsDegenerate(clip, sliver, ratio); got {
			t.Errorf("at ratio %g, a real sliver overlap was read as contact", ratio)
		}
	}
}

// shareIsDegenerate re-runs the clip-and-measure step at a chosen ratio, so the sweep above
// tests the threshold rather than a re-implementation of it.
func shareIsDegenerate(clip, other [3][2]float64, ratio float64) bool {
	flat := clipProjected(other, clip)
	if len(flat) < 3 {
		return true
	}
	smaller := stdmath.Min(triArea2D(clip), triArea2D(other))
	return polygonArea2D(flat)/smaller < ratio
}

// clipProjected runs clipToTriangle for a purely 2D pair by giving it flat 3D points.
func clipProjected(t, clip [3][2]float64) [][2]float64 {
	pts := [3]math.Point3{
		math.P3(t[0][0], t[0][1], 0), math.P3(t[1][0], t[1][1], 0), math.P3(t[2][0], t[2][1], 0),
	}
	_, flat := clipToTriangle(pts, t, clip)
	return flat
}
