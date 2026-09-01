// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// cylLoop builds a closed on-cylinder imprint loop from (azimuth θ, height z) vertices on the r=3 axis-z
// cylinder — the synthetic second-cut imprint the disjoint gate classifies.
func cylLoop(pts ...[2]float64) geom.Curve3 {
	verts := make([]math.Point3, 0, len(pts)+1)
	for _, p := range pts {
		th, z := p[0], p[1]
		verts = append(verts, math.P3(math.Scalar(3*stdmath.Cos(th)), math.Scalar(3*stdmath.Sin(th)), math.Scalar(z)))
	}
	verts = append(verts, verts[0]) // close the loop
	pl, _ := geom.NewPolyline(verts)
	return &pl
}

// TestDisjointFromPriorClassifies exercises the ship/decline gate for the partial-rim second cut against the
// notched cylinder (first cut = the plane x+z<=9.5, notch floor at the front reaching z=6.5). The prior loop is
// the notched top boundary.
func TestDisjointFromPriorClassifies(t *testing.T) {
	t.Parallel()
	c := cutUVFromNotched(t)
	pi := stdmath.Pi
	cases := []struct {
		name     string
		imprint  geom.Curve3
		disjoint bool
	}{
		// A small loop on the back wall, low, far from the front notch — the shippable disjoint case.
		{"back-clear", cylLoop([2]float64{pi - 0.3, 2.5}, [2]float64{pi + 0.3, 2.5}, [2]float64{pi + 0.3, 3.5}, [2]float64{pi - 0.3, 3.5}), true},
		// A front loop straddling the notch floor (z 5.5..7.5 at θ≈0, floor 6.5) — crosses into removed material.
		{"crosses-notch", cylLoop([2]float64{-0.3, 5.5}, [2]float64{0.3, 5.5}, [2]float64{0.3, 7.5}, [2]float64{-0.3, 7.5}), false},
		// A loop entirely inside the removed notch (z 7.5..9 at θ≈0) — nothing survived to cut.
		{"in-notch", cylLoop([2]float64{-0.3, 7.5}, [2]float64{0.3, 7.5}, [2]float64{0.3, 9}, [2]float64{-0.3, 9}), false},
		// A loop grazing the notch floor within the ~30µm margin (top vertex 10µm below the z=6.5 boundary point).
		{"near-tangent", cylLoop([2]float64{0, 6.499}, [2]float64{-0.08, 6.35}, [2]float64{0.08, 6.35}), false},
		// A full belt around the whole azimuth — non-contractible, outside the disjoint sub-family.
		{"wraps-azimuth", cylLoop([2]float64{0, 3}, [2]float64{pi / 2, 3}, [2]float64{pi, 3}, [2]float64{3 * pi / 2, 3}), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := c.disjointFromPrior([]geom.Curve3{tc.imprint}); got != tc.disjoint {
				t.Errorf("disjointFromPrior(%s) = %v, want %v", tc.name, got, tc.disjoint)
			}
		})
	}
}
