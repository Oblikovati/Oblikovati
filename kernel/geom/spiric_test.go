// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// spiricSetup builds the section coefficients for a torus cut by the axis-parallel plane y=a (normal
// +y), the single-oval cap geometry the oblique torus half-space exercises (Oblikovati/Oblikovati#1375).
func spiricSetup(t *testing.T, a float64) (Torus, Plane, float64, float64) {
	t.Helper()
	tor, err := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	if err != nil {
		t.Fatalf("NewTorus: %v", err)
	}
	pl, err := NewPlane(math.P3(0, a, 0), math.V3(0, 1, 0))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	// w(v)=±1 at the v-extremes: a/(R+r·cos v)=1 → cos v=(a−R)/r.
	vc := stdmath.Acos((a - tor.MajorRadius) / tor.MinorRadius)
	return tor, pl, vc, a
}

// TestSpiricArcLiesOnTorusAndPlane checks the spiric edge is exact: every sampled point sits on the
// torus surface (distance to the tube centre equals the minor radius) and on the cut plane.
func TestSpiricArcLiesOnTorusAndPlane(t *testing.T) {
	t.Parallel()
	tor, pl, vc, a := spiricSetup(t, 6)
	phi, m, k, c := TorusSectionCoeffs(tor, pl)
	arc := SpiricArc{Torus: tor, Phi: phi, M: m, K: k, C: c, Branch: +1, V0: -vc, V1: vc}
	for i := 0; i <= 20; i++ {
		p := arc.PointAt(float64(i) / 20)
		// On the plane y=a.
		if d := stdmath.Abs(float64(p.Y) - a); d > 1e-9 {
			t.Fatalf("point %v off plane y=%g by %g", p, a, d)
		}
		// On the torus: distance from the axis, minus major radius, then with z gives the tube radius.
		rad := stdmath.Hypot(float64(p.X), float64(p.Y)) - tor.MajorRadius
		tube := stdmath.Hypot(rad, float64(p.Z))
		if d := stdmath.Abs(tube - tor.MinorRadius); d > 1e-9 {
			t.Fatalf("point %v off torus tube by %g", p, d)
		}
	}
}

// TestSpiricArcBranchesMeetAtExtremes checks the two branches over the same v-range close the oval:
// they coincide at v=±vc (where w=±1, both roots collapse to u=Phi).
func TestSpiricArcBranchesMeetAtExtremes(t *testing.T) {
	t.Parallel()
	tor, pl, vc, _ := spiricSetup(t, 6)
	phi, m, k, c := TorusSectionCoeffs(tor, pl)
	plus := SpiricArc{Torus: tor, Phi: phi, M: m, K: k, C: c, Branch: +1, V0: -vc, V1: vc}
	minus := SpiricArc{Torus: tor, Phi: phi, M: m, K: k, C: c, Branch: -1, V0: -vc, V1: vc}
	for _, tt := range []float64{0, 1} {
		if d := plus.PointAt(tt).DistanceTo(minus.PointAt(tt)); d > 1e-9 {
			t.Fatalf("branches differ at t=%g by %g (should meet at the oval extreme)", tt, d)
		}
	}
	// Away from the extremes the branches are distinct (the oval has interior width).
	if d := plus.PointAt(0.5).DistanceTo(minus.PointAt(0.5)); d < 0.1 {
		t.Fatalf("branches coincide at t=0.5 (oval should have width), distance %g", d)
	}
}

// TestSpiricArcTangentMatchesFiniteDifference checks TangentAt agrees with a central finite difference
// of PointAt in the oval's interior (away from the v-extremes, where du/dv diverges by construction).
func TestSpiricArcTangentMatchesFiniteDifference(t *testing.T) {
	t.Parallel()
	tor, pl, vc, _ := spiricSetup(t, 6)
	phi, m, k, c := TorusSectionCoeffs(tor, pl)
	arc := SpiricArc{Torus: tor, Phi: phi, M: m, K: k, C: c, Branch: +1, V0: -vc, V1: vc}
	if lo, hi := arc.Domain(); lo != 0 || hi != 1 {
		t.Fatalf("Domain() = (%g, %g), want (0, 1)", lo, hi)
	}
	const h = 1e-6
	for _, tt := range []float64{0.35, 0.5, 0.65} {
		fd := arc.PointAt(tt + h).VectorTo(arc.PointAt(tt - h)).Scale(math.Scalar(-1 / (2 * h)))
		got := arc.TangentAt(tt)
		if d := float64(got.Sub(fd).Length()); d > 1e-3*float64(fd.Length())+1e-6 {
			t.Errorf("TangentAt(%g)=%v vs finite-diff %v, diff %g", tt, got, fd, d)
		}
	}
}
