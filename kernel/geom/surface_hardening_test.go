// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// scaledSkewedPatch is skewedPatch's control net uniformly scaled, so the surface
// is geometrically self-similar at every scale: the tangent angles (and thus the
// dimensionless first-form conditioning) are identical, only the lengths change.
// It is the fixture that separates a scale-invariant degeneracy test from the old
// absolute determinant cutoff.
func scaledSkewedPatch(t *testing.T, scale float64) BSplineSurface {
	t.Helper()
	base := [][]math.Point3{
		{{X: 0, Y: 0, Z: 0}, {X: 1, Y: 1, Z: 0.5}, {X: 2, Y: 2, Z: 0}},
		{{X: 1, Y: 0, Z: 1}, {X: 2, Y: 1, Z: 1.5}, {X: 3, Y: 2, Z: 1}},
		{{X: 2, Y: 0, Z: 0}, {X: 3, Y: 1, Z: 0.5}, {X: 4, Y: 2, Z: 0}},
	}
	ctrl := make([][]math.Point3, len(base))
	for i, row := range base {
		ctrl[i] = make([]math.Point3, len(row))
		for j, p := range row {
			ctrl[i][j] = math.P3(p.X*scale, p.Y*scale, p.Z*scale)
		}
	}
	w := [][]float64{{1, 1, 1}, {1, 1, 1}, {1, 1, 1}}
	s, err := NewBSplineSurface(2, 2, ctrl, w, []float64{0, 0, 0, 1, 1, 1}, []float64{0, 0, 0, 1, 1, 1})
	if err != nil {
		t.Fatalf("scaled skewed patch (scale %g): %v", scale, err)
	}
	return s
}

// opaqueSurface wraps a Surface so the SurfaceSecondPartials type switch misses it
// and falls through to the numeric path — the way to exercise numericSecondPartials
// against a fixture whose curvature has a known closed form (the embedded sphere).
type opaqueSurface struct{ Surface }

// TestFirstFormDegeneracyIsScaleInvariant is acceptance criterion 1 of #1402: the
// degeneracy verdict on the first fundamental form depends only on the tangent
// ANGLE, not the model scale — where the retired absolute determinant cutoff flips.
func TestFirstFormDegeneracyIsScaleInvariant(t *testing.T) {
	for _, k := range []float64{1e-8, 1e-4, 1, 1e4, 1e8} {
		a := k * k // tangents scale by k, so a = c = |Su|² scale by k²
		if degenerateFirstForm(a, 0, a) {
			t.Errorf("scale k=%g: orthogonal frame wrongly flagged degenerate", k)
		}
		if !degenerateFirstForm(a, a, a) {
			t.Errorf("scale k=%g: parallel frame not flagged degenerate", k)
		}
	}
	// Contrast with the retired absolute cutoff (det = a·c − b² < 1e-18): at k=1e-5 a
	// perfectly orthogonal frame has det = 1e-20 and would have been rejected, while
	// the scale-invariant test correctly accepts it.
	const a = 1e-5 * 1e-5
	if det := a * a; det >= 1e-18 {
		t.Fatalf("fixture invalid: det=%g is not below the legacy cutoff", det)
	}
	if degenerateFirstForm(a, 0, a) {
		t.Error("scale-invariant test must accept the orthogonal frame the legacy cutoff rejected")
	}
	// A collapsed tangent (a·c ≤ 0, e.g. a sphere pole) is degenerate regardless of b.
	if !degenerateFirstForm(0, 0, 1) {
		t.Error("a collapsed tangent (a=0) must be flagged degenerate")
	}
}

// TestParamAtConvergesAtExtremeScale is acceptance criterion 1 of #1402 end to end:
// the NURBS point inversion (whose Gauss–Newton step guards the frame with the new
// scale-invariant test) recovers a perpendicular foot on a patch shrunk to µm and
// grown to km, where the old absolute determinant cutoff aborts the small-scale step.
func TestParamAtConvergesAtExtremeScale(t *testing.T) {
	for _, scale := range []float64{1e-5, 1, 1e5} {
		s := scaledSkewedPatch(t, scale)
		foot := s.PointAt(0.35, 0.65)
		q := foot.TranslateBy(s.NormalAt(0.35, 0.65).Scale(math.Scalar(0.3 * scale)))
		u, v := s.ParamAt(q)
		if r := perpCosine(s, q, u, v); r > 1e-7 {
			t.Errorf("scale %g: ParamAt foot not perpendicular, residual %g", scale, r)
		}
	}
}

// TestNumericSecondPartialsCurvature is acceptance criterion 2 of #1402: the numeric
// second-partial fallback (now an optimal, domain-scaled central difference) matches
// the analytic closed form on a known surface — a sphere, whose principal curvatures
// are ±1/R — within a tight tolerance, instead of the 5–50% error a fixed step admits.
// The genuine Sphere uses the closed-form second partials; the opaque wrapper forces
// the numeric path, so the two results must agree.
func TestNumericSecondPartialsCurvature(t *testing.T) {
	for _, r := range []float64{0.5, 2, 50} {
		sphere, err := NewSphere(math.P3(0, 0, 0), r)
		if err != nil {
			t.Fatalf("sphere r=%g: %v", r, err)
		}
		_, wantMax, wantMin := SurfaceCurvatures(sphere, 0.6, 0.3)              // analytic
		_, gotMax, gotMin := SurfaceCurvatures(opaqueSurface{sphere}, 0.6, 0.3) // numeric
		if relErr(gotMax, wantMax) > 1e-5 || relErr(gotMin, wantMin) > 1e-5 {
			t.Errorf("sphere r=%g: numeric curvatures (%g, %g), analytic (%g, %g)", r, gotMax, gotMin, wantMax, wantMin)
		}
	}
}

// relErr is the relative error |got − want|/|want| (want assumed nonzero here).
func relErr(got, want float64) float64 {
	return stdmath.Abs(got-want) / stdmath.Abs(want)
}

// opaqueCurve2 wraps a Curve2 so CurveDerivatives2's type switch misses it and falls
// through to numericDers2 — the 2D analogue of opaqueSurface, used to exercise the
// per-order finite-difference fallback against a known closed form.
type opaqueCurve2 struct{ Curve2 }

// TestNumericDers2Curvature exercises the per-order optimal steps in numericDers2 (the
// 2D twin of numericDers3, #1402): the curvature of a circle of radius R is 1/R, so the
// numeric fallback must recover it within a tight tolerance where a single fixed step
// would not. A circle is the simplest fixture with a nonzero, constant analytic curvature.
func TestNumericDers2Curvature(t *testing.T) {
	for _, r := range []float64{0.25, 3, 40} {
		circle := NewCircle2d(math.P2(0, 0), r)
		got := CurveCurvature2(opaqueCurve2{circle}, 0.3) // numeric path
		want := CurveCurvature2(circle, 0.3)              // analytic path (signed 1/R)
		if relErr(got, want) > 1e-4 {
			t.Errorf("circle r=%g: numeric curvature %g, analytic %g", r, got, want)
		}
	}
}

// TestRationalWeightCollapseReturnsFiniteZero is acceptance criterion 3 of #1402: a
// collapsed rational denominator (out-of-span / malformed evaluation) yields a finite
// zero from point and deriv instead of leaking NaN/Inf into downstream consumers.
func TestRationalWeightCollapseReturnsFiniteZero(t *testing.T) {
	collapsed := homog{a: math.V3(1, 2, 3), w: 0}
	p := collapsed.point()
	if stdmath.IsNaN(float64(p.X)) || stdmath.IsInf(float64(p.X), 0) || p != (math.Point3{}) {
		t.Errorf("point() on a collapsed weight = %v, want the finite origin", p)
	}
	d := collapsed.deriv(homog{a: math.V3(4, 5, 6), w: 1})
	if stdmath.IsNaN(float64(d.X)) || stdmath.IsInf(float64(d.X), 0) || d != (math.Vector3{}) {
		t.Errorf("deriv() on a collapsed weight = %v, want the finite zero vector", d)
	}
	// A usable weight still evaluates the rational point: A/w = (2,4,6)/2 = (1,2,3).
	if got := (homog{a: math.V3(2, 4, 6), w: 2}).point(); got != math.P3(1, 2, 3) {
		t.Errorf("point() with a valid weight = %v, want (1,2,3)", got)
	}
}
