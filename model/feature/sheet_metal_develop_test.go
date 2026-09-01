// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// devForTest builds a development for a 90° fold about the X axis at z=t: up=+Z, out=+Y, inner
// radius r=0.2, gauge t=0.1, neutral ρ=r+0.44t — the canonical top-edge flange frame.
func devForTest(t *testing.T) bendDevelop {
	t.Helper()
	dev, err := newBendDevelop(BendTransform{
		LinePoint: math.P3(0, 0, 0.1), LineDir: math.V3(1, 0, 0),
		Up: math.V3(0, 0, 1), Out: math.V3(0, 1, 0),
		Angle: stdmath.Pi / 2, Radius: 0.2, Thickness: 0.1, Neutral: 0.2 + 0.44*0.1,
	})
	if err != nil {
		t.Fatalf("newBendDevelop: %v", err)
	}
	return dev
}

// TestDevelopRoundTrip flatToFolded inverts foldedToFlat across the base, the bend arc, and the
// flange — the bijection that lets a cut placed while flat ride back through the bend.
func TestDevelopRoundTrip(t *testing.T) {
	t.Parallel()
	dev := devForTest(t)
	// Sample points spanning the three regions: base (y<0 side of the line), the arc (a point on
	// the inner surface part-way round), and the flange (well up the wall). Use folded positions.
	centre := math.P3(0, 0, 0.1+0.2) // line + up·r
	pts := []math.Point3{
		math.P3(1, -1, 0.05), // base, below the bend line
		centre.TranslateBy(math.V3(0, stdmath.Sin(0.6), -stdmath.Cos(0.6)).Scale(0.25)), // arc, φ≈0.6, ρ=0.25
		math.P3(2, 0.2, 0.8), // flange wall (up high, out by ~r)
	}
	for i, q := range pts {
		flat := dev.foldedToFlat(q)
		back := dev.flatToFolded(flat)
		if back.DistanceTo(q) > 1e-9 {
			t.Errorf("point %d: round-trip moved it %.3g (q=%v flat=%v back=%v)", i, back.DistanceTo(q), q, flat, back)
		}
	}
}

// TestDevelopFlattensArc the inner-surface arc develops to a flat strip: a folded inner point at
// 90° lands at across = (π/2)·neutral and z = t (coplanar with the base top), proving the bend
// is unrolled, not rigidly rotated.
func TestDevelopFlattensArc(t *testing.T) {
	t.Parallel()
	dev := devForTest(t)
	centre := math.P3(0, 0, 0.3)
	innerAt90 := centre.TranslateBy(math.V3(0, 1, 0).Scale(0.2)) // φ=90°, ρ=r ⇒ inner end of the arc
	flat := dev.foldedToFlat(innerAt90)
	wantAcross := (stdmath.Pi / 2) * (0.2 + 0.44*0.1)
	if stdmath.Abs(float64(flat.Y)-wantAcross) > 1e-9 {
		t.Errorf("developed across = %.4f, want %.4f (neutral arc length)", flat.Y, wantAcross)
	}
	if stdmath.Abs(float64(flat.Z)-0.1) > 1e-9 {
		t.Errorf("developed inner point z = %.4f, want 0.1 (flush with the base top)", flat.Z)
	}
}
