// SPDX-License-Identifier: GPL-2.0-only

package geom_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A section that IS a circle must come back as a Circle. The one general intersector builds every
// ruled∩quadric section as a RuledQuadricArc, and delivering a circle that way loses everything
// downstream that reads the curve's kind — the per-face oracle's band walk, the tessellator's conformal
// rim stations, a bore rim's provenance name (ADR-0061 stage 4).
func TestCircularRuledSectionComesBackAsACircle(t *testing.T) {
	t.Parallel()
	sphere, err := geom.NewSphere(math.P3(0, 0, 0), 5)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	// A cylinder whose axis passes through the sphere's centre cuts it in two circles at y = ±4.
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 1, 0), 3)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	for _, tc := range []struct {
		name string
		a, b geom.Surface
	}{{"sphere base", sphere, cyl}, {"cylinder base", cyl, sphere}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			curves, handled := geom.IntersectSurfacesAnalytic(tc.a, tc.b, geom.ResolutionForSize(10))
			if !handled || len(curves) != 2 {
				t.Fatalf("IntersectSurfacesAnalytic: handled=%v curves=%d, want 2", handled, len(curves))
			}
			for i, cv := range curves {
				c, ok := cv.(geom.Circle)
				if !ok {
					t.Fatalf("section %d is %T, want geom.Circle", i, cv)
				}
				if stdmath.Abs(c.Radius-3) > 1e-9 {
					t.Errorf("section %d radius %.12f, want 3", i, c.Radius)
				}
				if y := stdmath.Abs(float64(c.Center.Y)); stdmath.Abs(y-4) > 1e-9 {
					t.Errorf("section %d centre at |y|=%.12f, want 4", i, y)
				}
			}
		})
	}
}

// Both operands of a boolean build the same section from OPPOSITE bases, and their circles must be
// identical — centre, normal and seam — or the shared imprint no longer welds.
func TestCircularSectionIsIdenticalFromEitherBase(t *testing.T) {
	t.Parallel()
	sphere, err := geom.NewSphere(math.P3(0, 0, 0), 5)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 1, 0), 3)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	res := geom.ResolutionForSize(10)
	fromA, _ := geom.IntersectSurfacesAnalytic(sphere, cyl, res)
	fromB, _ := geom.IntersectSurfacesAnalytic(cyl, sphere, res)
	if len(fromA) != len(fromB) {
		t.Fatalf("%d sections from one base, %d from the other", len(fromA), len(fromB))
	}
	for i := range fromA {
		for k := 0; k <= 8; k++ {
			s := float64(k) / 8
			if d := float64(fromA[i].PointAt(s).DistanceTo(fromB[i].PointAt(s))); d > 1e-9 {
				t.Errorf("section %d at t=%.3f differs by %.3e between the two bases", i, s, d)
			}
		}
	}
}
