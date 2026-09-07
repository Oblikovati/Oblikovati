// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
)

// Two overlapping spheres meet in the circle of their radical plane, so every one of their booleans is
// two spherical caps and its volume is a classical closed form. No oracle is needed: the lens volume
// of two equal spheres is (π/12d)(2r−d)²(d²+4dr), and the rest follows.
//
// The pipeline had no pairing for two closed surfaces at all before ADR-0061 stage 4, so this whole
// family was served by the faceted engines — the reason the volumes below were never asserted.
func TestSpherePairVolumesAreExact(t *testing.T) {
	t.Parallel()
	const r, d = 2.0, 2.0
	lens := stdmath.Pi / (12 * d) * (2*r - d) * (2*r - d) * (d*d + 4*d*r)
	ball := 4.0 / 3 * stdmath.Pi * r * r * r
	for _, tc := range []struct {
		name string
		op   PartFeatureOperation
		want float64
	}{
		{"intersect", Intersect, lens},
		{"join", Join, 2*ball - lens},
		{"cut", Cut, ball - lens},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a, err := brep.SolidSphere(math.P3(0, 0, 0), r, "a")
			if err != nil {
				t.Fatal(err)
			}
			b, err := brep.SolidSphere(math.P3(math.Scalar(d), 0, 0), r, "b")
			if err != nil {
				t.Fatal(err)
			}
			res, err := Boolean(tc.op, a, b)
			if err != nil {
				t.Fatalf("Boolean(%v): %v", tc.op, err)
			}
			if v := Validate(res); !v.ValidSolid() {
				t.Fatalf("sphere pair %v is not a valid solid: %+v", tc.op, v.Issues)
			}
			got, ok := query.AnalyticShellVolume(res.Shells()[0])
			if !ok {
				t.Fatal("the sphere pair's volume did not integrate analytically — its faces are not exact")
			}
			if stdmath.Abs(got-tc.want) > 1e-9 {
				t.Errorf("volume %.9f, want %.9f (the closed form)", got, tc.want)
			}
		})
	}
}
