// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/math"
)

// TestNearestCrossingCrispNearSurface is the regression gate for the nearest-crossing classifier: a
// point a hair off a face must classify DECISIVELY on the correct side, on both sides of the wall. This is
// the property the winding number lacks — its field reads ≈½ that close to the boundary — and the reason a
// fillet gate's small-offset material probe needs the first-hit ray cast, not the winding.
func TestNearestCrossingCrispNearSurface(t *testing.T) {
	cyl, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	q, box := newFluxQuery(facesOfAny(cyl)), cyl.RangeBox()
	cases := []struct {
		name string
		p    math.Point3
		want bool
	}{
		{"a hair inside the wall", math.P3(1.999, 0, 2), true},
		{"a hair outside the wall", math.P3(2.001, 0, 2), false},
		{"a hair below the top cap", math.P3(0, 0, 3.999), true},
		{"a hair above the top cap", math.P3(0, 0, 4.001), false},
	}
	for _, c := range cases {
		in, ok := q.nearestCrossingInside(c.p, box)
		if !ok {
			t.Errorf("%s: every ray direction grazed (no clean first hit)", c.name)
			continue
		}
		if in != c.want {
			t.Errorf("%s: nearestCrossingInside(%v) = %v, want %v", c.name, c.p, in, c.want)
		}
	}
}

// TestNearestCrossingOrientationIndependent checks the first-hit side reads the orientation-normalized
// outward normal, not the stored Reversed flag: flipping every face's flag must not change the verdict.
func TestNearestCrossingOrientationIndependent(t *testing.T) {
	sph, err := SolidSphere(math.P3(0, 0, 0), 3, "sphere")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	faces := facesOfAny(sph)
	for i := range faces {
		faces[i].reversed = !faces[i].reversed
	}
	q, box := newFluxQuery(faces), sph.RangeBox()
	if in, ok := q.nearestCrossingInside(math.P3(0, 0, 0), box); !ok || !in {
		t.Errorf("sphere centre with flipped flags: nearestCrossingInside = (%v, ok=%v), want (true, true)", in, ok)
	}
	if in, ok := q.nearestCrossingInside(math.P3(9, 0, 0), box); !ok || in {
		t.Errorf("point outside sphere with flipped flags: nearestCrossingInside = (%v, ok=%v), want (false, true)", in, ok)
	}
}
