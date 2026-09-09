// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Two overlapping spheres are the simplest curved-versus-curved crossing there is — they meet in the
// circle of their radical plane — and the mixed pipeline had no pairing for two closed surfaces at all,
// so a ball meeting a ball declined on box overlap alone and the whole family went to the faceted
// engines (ADR-0061 stage 4).
//
// Each operation keeps two spherical caps. The exact volumes are checked where the integrator lives,
// in kernel/ops/boolean's TestSpherePairVolumesAreExact.
func TestSpherePairBooleansAreExact(t *testing.T) {
	t.Parallel()
	const r, d = 2.0, 2.0
	for _, tc := range []struct {
		name string
		op   Op
	}{
		{"intersection is the lens", Intersection},
		{"union is both balls less the lens", Union},
		{"difference is a ball less the lens", Difference},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a, err := SolidSphere(math.P3(0, 0, 0), r, "a")
			if err != nil {
				t.Fatal(err)
			}
			b, err := SolidSphere(math.P3(math.Scalar(d), 0, 0), r, "b")
			if err != nil {
				t.Fatal(err)
			}
			res, err := Boolean(tc.op, a, b)
			if err != nil {
				t.Fatalf("Boolean(%v) on two spheres: %v", tc.op, err)
			}
			assertWatertight(t, res)
			assertEveryFaceWinds(t, res)
			if n := len(res.Faces()); n != 2 {
				t.Errorf("%d faces, want 2 (one spherical cap from each operand)", n)
			}
			for _, f := range res.Faces() {
				if _, isSphere := f.Geometry().(geom.Sphere); !isSphere {
					t.Errorf("a face is a %T, not a sphere — the pair fell to a faceted path", f.Geometry())
				}
			}
		})
	}
}

// TestSpherePairSectionIsOneCircle pins the imprint itself: two closed surfaces cross in one closed
// curve, written into BOTH sides' lists so the two charts split on identical coordinates.
func TestSpherePairSectionIsOneCircle(t *testing.T) {
	t.Parallel()
	a, _ := SolidSphere(math.P3(0, 0, 0), 2, "a")
	b, _ := SolidSphere(math.P3(2, 0, 0), 2, "b")
	pa, pb := partitionFaces(a), partitionFaces(b)
	fa, _ := pa.closedSurfaces()
	fb, _ := pb.closedSurfaces()
	curves, _, ok := closedSurfacePairImprint(fa[0], fb[0])
	if !ok {
		t.Fatal("two crossing spheres are undecided; their section is a circle in closed form")
	}
	if len(curves) != 1 {
		t.Fatalf("two spheres imprinted %d curves, want 1", len(curves))
	}
	if _, isCircle := curves[0].(geom.Circle); !isCircle {
		t.Errorf("the imprint is a %T, want a geom.Circle", curves[0])
	}
}

// TestDisjointSpheresImprintNothing: spheres apart are DECIDED clear, not undecided — the same reading
// the wall pairing takes, so a boolean between them is not declined for lack of a crossing.
func TestDisjointSpheresImprintNothing(t *testing.T) {
	t.Parallel()
	a, _ := SolidSphere(math.P3(0, 0, 0), 2, "a")
	b, _ := SolidSphere(math.P3(9, 0, 0), 2, "b")
	pa, pb := partitionFaces(a), partitionFaces(b)
	fa, _ := pa.closedSurfaces()
	fb, _ := pb.closedSurfaces()
	curves, _, ok := closedSurfacePairImprint(fa[0], fb[0])
	if !ok || len(curves) != 0 {
		t.Errorf("disjoint spheres: ok=%v, %d curves; want decided and empty", ok, len(curves))
	}
}
