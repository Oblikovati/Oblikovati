// SPDX-License-Identifier: GPL-2.0-only

package geom_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Two cylinders of EQUAL radius whose axes meet are the one pair the ruled∩quadric form declines for
// exactness rather than conditioning: its quadratic's two roots coincide at a fold. Subtracting the two
// implicit forms leaves a difference of squares, so the section lies in the two bisector planes and is
// an ellipse in each. Every point of both must lie on BOTH cylinders (ADR-0061 stage 4).
func TestEqualCylindersSectionInTwoExactEllipses(t *testing.T) {
	t.Parallel()
	const r = 3.0
	for _, row := range []struct {
		name string
		axis math.Vector3
	}{
		{"perpendicular axes", math.V3(1, 0, 0)},
		{"oblique axes", math.V3(1, 0, 1)},
	} {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			cz, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r)
			if err != nil {
				t.Fatalf("cylinder z: %v", err)
			}
			other, err := geom.NewCylinder(math.P3(0, 0, 0), row.axis, r)
			if err != nil {
				t.Fatalf("cylinder other: %v", err)
			}
			curves, ok := geom.IntersectSurfacesAnalytic(cz, other, geom.ResolutionForSize(12))
			if !ok || len(curves) != 2 {
				t.Fatalf("ok=%v, %d curves, want 2 ellipses", ok, len(curves))
			}
			for i, c := range curves {
				if _, isEllipse := c.(geom.EllipseFull); !isEllipse {
					t.Errorf("curve %d is %T, want a geom.EllipseFull", i, c)
				}
				assertOnBothCylinders(t, c, cz, other, r)
			}
		})
	}
}

// A cylinder of a DIFFERENT radius keeps the general form: the closed form must fire at the degeneracy
// and nowhere else.
func TestUnequalCylindersKeepTheGeneralForm(t *testing.T) {
	t.Parallel()
	cz, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	cx, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), 1.5)
	curves, ok := geom.IntersectSurfacesAnalytic(cz, cx, geom.ResolutionForSize(12))
	if !ok {
		t.Fatal("unequal cylinders declined")
	}
	for i, c := range curves {
		if _, isEllipse := c.(geom.EllipseFull); isEllipse {
			t.Errorf("curve %d came back an ellipse: the equal-radius form fired on an unequal pair", i)
		}
	}
}

// assertOnBothCylinders walks a section curve and checks every sample sits on both walls.
func assertOnBothCylinders(t *testing.T, c geom.Curve3, a, b geom.Cylinder, r float64) {
	t.Helper()
	lo, hi := c.Domain()
	for k := 0; k <= 32; k++ {
		p := c.PointAt(lo + (hi-lo)*float64(k)/32)
		for _, cyl := range []geom.Cylinder{a, b} {
			d := cyl.Origin.VectorTo(p)
			axial := d.Dot(cyl.AxisDir.AsVector())
			radial := float64(d.Sub(cyl.AxisDir.AsVector().Scale(axial)).Length())
			if stdmath.Abs(radial-r) > 1e-9 { // tol:calibrated — an exact section, to a few ulps of r
				t.Fatalf("sample %d sits %g from the axis, want %g: the section is not on both walls", k, radial, r)
			}
		}
	}
}
