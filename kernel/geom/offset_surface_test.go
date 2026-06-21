// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestOffsetSurfaceDefiningProperty checks the parallel-surface identities on a cylinder: a point of
// the offset surface is the base point displaced by Distance along the base normal, and the offset
// normal equals the base normal. Exercised over several (u,v) and both offset directions.
func TestOffsetSurfaceDefiningProperty(t *testing.T) {
	cyl, err := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 5)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []float64{2, -1.5} {
		off := OffsetSurface{Base: cyl, Distance: d}
		for _, u := range []float64{0, 1, 2.5, 4} {
			for _, v := range []float64{-3, 0, 6} {
				want := cyl.PointAt(u, v).TranslateBy(cyl.NormalAt(u, v).Scale(math.Scalar(d)))
				if got := off.PointAt(u, v); got.DistanceTo(want) > 1e-9 {
					t.Errorf("d=%g PointAt(%g,%g) off by %g", d, u, v, got.DistanceTo(want))
				}
				if n, bn := off.NormalAt(u, v), cyl.NormalAt(u, v); n.Add(bn.Scale(-1)).Length() > 1e-12 {
					t.Errorf("d=%g NormalAt(%g,%g) != base normal", d, u, v)
				}
				// A point of the offset cylinder lies at radius (5+d) from the Z axis.
				p := off.PointAt(u, v)
				if r := stdmath.Hypot(float64(p.X), float64(p.Y)); stdmath.Abs(r-(5+d)) > 1e-9 {
					t.Errorf("d=%g radius = %g, want %g", d, r, 5+d)
				}
			}
		}
	}
}
