// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestEverySectionCurveReportsItsIncidence: a section curve's incidence conditions are what any other
// curve is solved against it with, and a curve that reports NONE cannot be intersected at all —
// curvePairMeets needs roots on both curves and pairs them by distance, so an empty condition list is a
// silent "they never meet". geom.CurveIncidence knew a RuledQuadricArc and none of the three section
// curves added after it, which is how an axial drill through a ring lost the crossings between its bore
// seams and the wall chart's own seam (ADR-0061 stage 5). This row fails when the next one is added.
func TestEverySectionCurveReportsItsIncidence(t *testing.T) {
	t.Parallel()
	cyl, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	ball, _ := NewSphere(math.P3(4, 0, 0), 2)
	ring, _ := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5)
	drill, _ := NewCylinder(math.P3(5, 0, 0), math.V3(0, 0, 1), 0.8)
	onTube, _ := NewSphere(math.P3(5, 0, 0), 2)

	for _, c := range []struct {
		name string
		pair [2]Surface
		size float64
	}{
		{"ruled wrap (crossing cylinders)", [2]Surface{cyl, mustRod(t)}, 12},
		{"ruled window (ball off a rod's axis)", [2]Surface{ball, cyl}, 10},
		{"torus window (an axial drill)", [2]Surface{ring, drill}, 12},
		{"torus wrap (a ball on the tube)", [2]Surface{ring, onTube}, 12},
	} {
		curves, handled := IntersectSurfacesAnalytic(c.pair[0], c.pair[1], ResolutionForSize(c.size))
		if !handled || len(curves) == 0 {
			t.Fatalf("%s: handled=%v curves=%d", c.name, handled, len(curves))
		}
		for i, cv := range curves {
			if _, isCircle := cv.(Circle); isCircle {
				continue // a canonicalised conic carries the ordinary planar condition
			}
			on := CurveIncidence(cv)
			if len(on) == 0 {
				t.Errorf("%s curve %d (%T) reports no incidence condition — it cannot be intersected", c.name, i, cv)
				continue
			}
			// Every condition must vanish along the whole curve, which is what makes it an incidence.
			lo, hi := cv.Domain()
			for k := 0; k <= 64; k++ {
				p := cv.PointAt(lo + (hi-lo)*float64(k)/64)
				for j, f := range on {
					if stdmath.Abs(f(p)) > 1e-9 { // tol:numeric — an implicit form's own rounding on the curve
						t.Fatalf("%s curve %d condition %d is %.3e at t=%g, want zero along the section",
							c.name, i, j, f(p), float64(k)/64)
					}
				}
			}
		}
	}
}

// mustRod is the crossing rod of the ruled-wrap row.
func mustRod(t *testing.T) Cylinder {
	t.Helper()
	rod, err := NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), 1.5)
	if err != nil {
		t.Fatalf("rod: %v", err)
	}
	return rod
}
