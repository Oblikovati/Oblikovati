// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati/math"
)

func sq(z float64, pts ...[2]float64) []math.Point3 {
	out := make([]math.Point3, len(pts))
	for i, p := range pts {
		out[i] = math.P3(math.Scalar(p[0]), math.Scalar(p[1]), math.Scalar(z))
	}
	return out
}

// TestAlignSectionsUntwists checks the correspondence step: a section given with a rotated point
// order is realigned so corresponding points track the previous section (minimum twist), rather
// than connecting mismatched corners (which would self-intersect the loft).
func TestAlignSectionsUntwists(t *testing.T) {
	ref := sq(0, [2]float64{0, 0}, [2]float64{1, 0}, [2]float64{1, 1}, [2]float64{0, 1})
	// Same square at z=1 but listed starting two corners later (a "twist" if matched by index).
	cur := sq(1, [2]float64{1, 1}, [2]float64{0, 1}, [2]float64{0, 0}, [2]float64{1, 0})
	got := alignSections([][]math.Point3{ref, cur})[1]
	// After alignment, got[k] should sit directly above ref[k] (same x,y).
	for k := range ref {
		if stdmath.Abs(float64(got[k].X-ref[k].X)) > 1e-9 || stdmath.Abs(float64(got[k].Y-ref[k].Y)) > 1e-9 {
			t.Fatalf("alignSections did not untwist: got[%d]=%v over ref[%d]=%v", k, got[k], k, ref[k])
		}
	}
}

// TestSplineTwoSectionsIsRuled checks that a 2-section loft stays a straight (ruled) blend —
// every interpolated point keeps the endpoints' x,y (Inventor's 2-section Free loft is ruled),
// only sampled densely.
func TestSplineTwoSectionsIsRuled(t *testing.T) {
	tri0 := sq(0, [2]float64{0, 0}, [2]float64{2, 0}, [2]float64{1, 2})
	tri1 := sq(3, [2]float64{0, 0}, [2]float64{2, 0}, [2]float64{1, 2})
	out := splineSections([][]math.Point3{tri0, tri1}, false, loftEnds{})
	if len(out) < 6 {
		t.Fatalf("2-section loft densified to %d sections, want many", len(out))
	}
	for _, sec := range out {
		for j, p := range sec {
			if stdmath.Abs(float64(p.X-tri0[j].X)) > 1e-9 || stdmath.Abs(float64(p.Y-tri0[j].Y)) > 1e-9 {
				t.Fatalf("2-section loft is not ruled: point drifted in x/y: %v (want xy of %v)", p, tri0[j])
			}
		}
	}
}

// TestSplineThreeSectionsBulges checks the spline blend: a 3-section loft whose middle section is
// offset sideways must CURVE through it — the blend passes through the middle (reaches its
// offset) and bulges off the straight line between the end sections.
func TestSplineThreeSectionsBulges(t *testing.T) {
	tri := func(z, dx float64) []math.Point3 {
		return sq(z, [2]float64{dx, 0}, [2]float64{dx + 2, 0}, [2]float64{dx + 1, 2})
	}
	out := splineSections([][]math.Point3{tri(0, 0), tri(1, 1), tri(2, 0)}, false, loftEnds{})
	// The end sections sit at dx=0; a ruled (straight) blend would keep x of point0 at 0
	// throughout. The spline must reach the middle's dx=1 and stay between.
	var maxX float64
	for _, sec := range out {
		if x := float64(sec[0].X); x > maxX {
			maxX = x
		}
	}
	if maxX < 0.9 {
		t.Fatalf("3-section loft did not bulge to the offset middle: max x of point0 = %.3f, want ≈1", maxX)
	}
	if maxX > 1.2 {
		t.Errorf("3-section loft overshoots the middle: max x = %.3f, want ≈1", maxX)
	}
}
