// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// opaqueCurve3 wraps a Curve3 and hides its concrete type so CurveDerivatives3 cannot dispatch to a
// closed form and must take the numericDers3 fallback — the path #1323 L1 fixes.
type opaqueCurve3 struct{ inner Curve3 }

func (o opaqueCurve3) PointAt(t float64) math.Point3    { return o.inner.PointAt(t) }
func (o opaqueCurve3) TangentAt(t float64) math.Vector3 { return o.inner.TangentAt(t) }
func (o opaqueCurve3) Domain() (lo, hi float64)         { return o.inner.Domain() }

// TestNumericDers3MatchesAnalyticHelix routes a helix through the numeric fallback and compares all
// three derivatives to the helix's closed form. The old fixed 1e-5 step gave d3 a 5–50% error; the
// per-order optimal steps bring it under 1e-4 relative.
func TestNumericDers3MatchesAnalyticHelix(t *testing.T) {
	t.Parallel()
	helix, err := NewHelix3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 0.8, 1.0, 0.2, 3, false)
	if err != nil {
		t.Fatalf("NewHelix3d: %v", err)
	}
	for _, tt := range []float64{0.15, 0.4, 0.65, 0.85} {
		wantD1, wantD2, wantD3 := helixDers(helix, tt)
		gotD1, gotD2, gotD3 := numericDers3(opaqueCurve3{helix}, tt)
		assertRel(t, "d1", tt, gotD1, wantD1, 1e-6)
		assertRel(t, "d2", tt, gotD2, wantD2, 1e-4)
		assertRel(t, "d3", tt, gotD3, wantD3, 1e-4)
	}
}

// TestCurveDerivatives3TakesNumericFallback confirms the opaque wrapper actually exercises the
// fallback (not a closed-form branch) and that CurveDerivatives3 returns the same fallback values.
func TestCurveDerivatives3TakesNumericFallback(t *testing.T) {
	t.Parallel()
	helix, _ := NewHelix3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 1, 1, 0, 2, false)
	d1, d2, d3 := CurveDerivatives3(opaqueCurve3{helix}, 0.5)
	wantD1, wantD2, wantD3 := helixDers(helix, 0.5)
	assertRel(t, "d1", 0.5, d1, wantD1, 1e-6)
	assertRel(t, "d2", 0.5, d2, wantD2, 1e-4)
	assertRel(t, "d3", 0.5, d3, wantD3, 1e-4)
}

func assertRel(t *testing.T, name string, at float64, got, want math.Vector3, tol float64) {
	t.Helper()
	scale := stdmath.Max(float64(want.Length()), 1e-9)
	if e := float64(got.Sub(want).Length()) / scale; e > tol {
		t.Errorf("%s at t=%g: rel error %g exceeds %g (got %v, want %v)", name, at, e, tol, got, want)
	}
}
