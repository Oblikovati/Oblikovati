// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// evaluatorSurfaces returns one representative of every surface kind.
func evaluatorSurfaces(t *testing.T) map[string]Surface {
	t.Helper()
	plane, err := NewPlane(math.P3(1, 0, 0), math.V3(0, 1, 1))
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	cyl, err := NewCylinder(math.P3(0, 1, 0), math.V3(1, 1, 0), 2)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	cone, err := NewCone(math.P3(0, 0, 1), math.V3(0, 0, 1), 0.5)
	if err != nil {
		t.Fatalf("cone: %v", err)
	}
	sphere, err := NewSphere(math.P3(1, 2, 3), 2.5)
	if err != nil {
		t.Fatalf("sphere: %v", err)
	}
	torus, err := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 1)
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	ecyl, err := NewEllipticalCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 3, 1)
	if err != nil {
		t.Fatalf("elliptical cylinder: %v", err)
	}
	econe, err := NewEllipticalCone(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 0.6, 0.3)
	if err != nil {
		t.Fatalf("elliptical cone: %v", err)
	}
	return map[string]Surface{
		"plane": plane, "cylinder": cyl, "cone": cone, "sphere": sphere,
		"torus": torus, "ellipticalCylinder": ecyl, "ellipticalCone": econe,
		"bspline": evaluatorBSplineSurface(t),
	}
}

// evaluatorBSplineSurface builds a rational biquadratic test patch.
func evaluatorBSplineSurface(t *testing.T) BSplineSurface {
	t.Helper()
	ctrl := [][]math.Point3{
		{{X: 0, Y: 0, Z: 0}, {X: 0, Y: 1, Z: 1}, {X: 0, Y: 2, Z: 0}},
		{{X: 1, Y: 0, Z: 1}, {X: 1, Y: 1, Z: 2}, {X: 1, Y: 2, Z: 1}},
		{{X: 2, Y: 0, Z: 0}, {X: 2, Y: 1, Z: 1}, {X: 2, Y: 2, Z: 0}},
		{{X: 3, Y: 0, Z: -1}, {X: 3, Y: 1, Z: 0}, {X: 3, Y: 2, Z: -1}},
	}
	weights := [][]float64{{1, 2, 1}, {1, 1, 1}, {2, 1, 2}, {1, 1, 1}}
	s, err := NewBSplineSurface(2, 2, ctrl, weights,
		[]float64{0, 0, 0, 0.5, 1, 1, 1}, []float64{0, 0, 0, 1, 1, 1})
	if err != nil {
		t.Fatalf("bspline surface: %v", err)
	}
	return s
}

// surfaceProbe returns an interior probe parameter per surface (clear of
// apexes, poles and knots).
func surfaceProbe(s Surface) (u, v float64) {
	uLo, uHi := s.UDomain()
	vLo, vHi := s.VDomain()
	u, v = 0.7, 0.6
	if !stdmath.IsInf(uHi, 0) {
		u = uLo + 0.37*(uHi-uLo)
	}
	if !stdmath.IsInf(vHi, 0) {
		v = vLo + 0.41*(vHi-vLo)
	} else if vLo == 0 {
		v = 1.3 // cone-like half-open axial range: stay off the apex
	}
	return u, v
}

// TestSurfacePartialsMatchFiniteDifferences oracles the closed-form second and
// third partials with central differences of DerivativesAt.
func TestSurfacePartialsMatchFiniteDifferences(t *testing.T) {
	const h = 1e-5
	for name, s := range evaluatorSurfaces(t) {
		u, v := surfaceProbe(s)
		puu, puv, pvv := SurfaceSecondPartials(s, u, v)
		fuu, fuv, fvv := numericSecondPartials(s, u, v)
		for _, pair := range [][2]math.Vector3{{puu, fuu}, {puv, fuv}, {pvv, fvv}} {
			if float64(pair[0].Sub(pair[1]).Length()) > 1e-4*stdmath.Max(1, float64(pair[0].Length())) {
				t.Errorf("%s second partials: %v vs finite difference %v", name, pair[0], pair[1])
			}
		}
		puuu, pvvv := SurfaceThirdPartials(s, u, v)
		guu1, _, _ := SurfaceSecondPartials(s, u+h, v)
		guu0, _, _ := SurfaceSecondPartials(s, u-h, v)
		fuuu := guu1.Sub(guu0).Scale(1 / (2 * h))
		if float64(puuu.Sub(fuuu).Length()) > 1e-3*stdmath.Max(1, float64(puuu.Length())) {
			t.Errorf("%s ∂³/∂u³ = %v, finite difference %v", name, puuu, fuuu)
		}
		_, _, gvv1 := SurfaceSecondPartials(s, u, v+h)
		_, _, gvv0 := SurfaceSecondPartials(s, u, v-h)
		fvvv := gvv1.Sub(gvv0).Scale(1 / (2 * h))
		if float64(pvvv.Sub(fvvv).Length()) > 1e-3*stdmath.Max(1, float64(pvvv.Length())) {
			t.Errorf("%s ∂³/∂v³ = %v, finite difference %v", name, pvvv, fvvv)
		}
	}
}

// TestSurfaceCurvaturesClosedForms pins the analytic principal curvatures.
func TestSurfaceCurvaturesClosedForms(t *testing.T) {
	sphere, _ := NewSphere(math.P3(0, 0, 0), 2)
	_, kMax, kMin := SurfaceCurvatures(sphere, 0.7, 0.3)
	if stdmath.Abs(kMax-kMin) > 1e-9 || stdmath.Abs(stdmath.Abs(kMax)-0.5) > 1e-9 {
		t.Errorf("sphere curvatures = (%g, %g), want equal magnitude 1/R", kMax, kMin)
	}

	cyl, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	dir, kMax, kMin := SurfaceCurvatures(cyl, 0.9, 1.4)
	if stdmath.Abs(kMax) > 1e-9 || stdmath.Abs(kMin+0.5) > 1e-9 {
		t.Errorf("cylinder curvatures = (%g, %g), want (0, -1/R)", kMax, kMin)
	}
	if stdmath.Abs(float64(dir.Dot(cyl.AxisDir.AsVector())))-1 > 1e-9 {
		t.Errorf("cylinder flat direction %v should be axial", dir)
	}

	// Torus at tube angle 0 (outer equator): −1/r and −cos v/(R + r·cos v).
	torus, _ := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 1)
	_, kMax, kMin = SurfaceCurvatures(torus, 0.5, 0)
	if stdmath.Abs(kMax+0.25) > 1e-9 || stdmath.Abs(kMin+1) > 1e-9 {
		t.Errorf("torus outer-equator curvatures = (%g, %g), want (-0.25, -1)", kMax, kMin)
	}

	plane, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if _, kMax, kMin := SurfaceCurvatures(plane, 1, 2); kMax != 0 || kMin != 0 {
		t.Errorf("plane curvatures = (%g, %g)", kMax, kMin)
	}
}

// TestSurfaceAreaClosedForms pins the bounded areas and NURBS quadrature.
func TestSurfaceAreaClosedForms(t *testing.T) {
	sphere, _ := NewSphere(math.P3(0, 0, 0), 2)
	if got := SurfaceArea(sphere); stdmath.Abs(got-2*twoPi*4) > 1e-9 {
		t.Errorf("sphere area = %g, want 4πR² = %g", got, 2*twoPi*4)
	}
	torus, _ := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 1)
	if got := SurfaceArea(torus); stdmath.Abs(got-twoPi*twoPi*3) > 1e-9 {
		t.Errorf("torus area = %g, want 4π²Rr = %g", got, twoPi*twoPi*3)
	}
	plane, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if got := SurfaceArea(plane); !stdmath.IsInf(got, 1) {
		t.Errorf("plane area = %g, want +Inf", got)
	}
	// A flat bilinear patch over a 3×2 rectangle has area 6.
	flat, err := NewBSplineSurface(1, 1,
		[][]math.Point3{{{X: 0, Y: 0}, {X: 0, Y: 2}}, {{X: 3, Y: 0}, {X: 3, Y: 2}}},
		[][]float64{{1, 1}, {1, 1}}, []float64{0, 0, 1, 1}, []float64{0, 0, 1, 1})
	if err != nil {
		t.Fatalf("flat patch: %v", err)
	}
	if got := SurfaceArea(flat); stdmath.Abs(got-6) > 1e-9 {
		t.Errorf("flat patch area = %g, want 6", got)
	}
}

// TestSurfaceIsoCurvesLieOnSurface samples every iso-curve against PointAt.
func TestSurfaceIsoCurvesLieOnSurface(t *testing.T) {
	for name, s := range evaluatorSurfaces(t) {
		u, v := surfaceProbe(s)
		for _, uDir := range []bool{true, false} {
			param := v
			if uDir {
				param = u
			}
			iso, err := SurfaceIsoCurve(s, uDir, param)
			if err != nil {
				t.Errorf("%s iso(uDir=%v): %v", name, uDir, err)
				continue
			}
			if !isoCurveOnSurface(s, iso) {
				t.Errorf("%s iso-curve (uDir=%v, param=%g) leaves the surface", name, uDir, param)
			}
		}
	}
}

// isoCurveOnSurface checks sampled iso-curve points sit on the surface (via
// the perpendicular foot).
func isoCurveOnSurface(s Surface, c Curve3) bool {
	lo, hi := c.Domain()
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
		lo, hi = 0, 2 // probe a finite window of a ruling line
	}
	for i := 0; i <= 16; i++ {
		p := c.PointAt(lo + (hi-lo)*float64(i)/16)
		_, _, foot := ClosestPointOnSurface(s, p)
		if foot.DistanceTo(p) > 1e-6 {
			return false
		}
	}
	return true
}

// TestSurfaceParamAtPointClassification pins the SolutionNature cases.
func TestSurfaceParamAtPointClassification(t *testing.T) {
	cyl, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	if _, _, nature := SurfaceParamAtPoint(cyl, math.P3(0, 0, 3)); nature != InfinitelyManySolutions {
		t.Errorf("cylinder axis query: nature = %v", nature)
	}
	u, v, nature := SurfaceParamAtPoint(cyl, math.P3(4, 0, 1))
	if nature != UniqueSolution {
		t.Errorf("cylinder generic query: nature = %v", nature)
	}
	if foot := cyl.PointAt(u, v); foot.DistanceTo(math.P3(2, 0, 1)) > 1e-9 {
		t.Errorf("cylinder foot = %v, want (2,0,1)", foot)
	}

	sphere, _ := NewSphere(math.P3(1, 1, 1), 2)
	if _, _, nature := SurfaceParamAtPoint(sphere, math.P3(1, 1, 1)); nature != InfinitelyManySolutions {
		t.Errorf("sphere center query: nature = %v", nature)
	}

	ecyl, _ := NewEllipticalCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 3, 1)
	if _, _, nature := SurfaceParamAtPoint(ecyl, math.P3(0, 0, 2)); nature != DistinctlyManySolutions {
		t.Errorf("elliptical cylinder axis query: nature = %v", nature)
	}

	bsp := evaluatorBSplineSurface(t)
	pu, pv := 0.41, 0.55
	near := bsp.PointAt(pu, pv).TranslateBy(bsp.NormalAt(pu, pv).Scale(0.05))
	gu, gv, nature := SurfaceParamAtPoint(bsp, near)
	if nature != UniqueSolution {
		t.Errorf("bspline near-surface query: nature = %v", nature)
	}
	if bsp.PointAt(gu, gv).DistanceTo(bsp.PointAt(pu, pv)) > 1e-3 {
		t.Errorf("bspline closest params = (%g, %g), want ≈(%g, %g)", gu, gv, pu, pv)
	}
}

// TestSurfaceNormalAtPoint projects an off-surface point and compares with the
// on-surface normal.
func TestSurfaceNormalAtPoint(t *testing.T) {
	sphere, _ := NewSphere(math.P3(0, 0, 0), 2)
	n := SurfaceNormalAtPoint(sphere, math.P3(5, 0, 0))
	if float64(n.Sub(math.V3(1, 0, 0)).Length()) > 1e-9 {
		t.Errorf("sphere normal at (5,0,0) = %v, want +X", n)
	}
}

// TestSurfaceRangeBoxes covers exact, conservative and unbounded boxes.
func TestSurfaceRangeBoxes(t *testing.T) {
	sphere, _ := NewSphere(math.P3(1, 2, 3), 2)
	b := SurfaceRangeBox(sphere)
	if b.Min.DistanceTo(math.P3(-1, 0, 1)) > 1e-12 || b.Max.DistanceTo(math.P3(3, 4, 5)) > 1e-12 {
		t.Errorf("sphere box = [%v, %v]", b.Min, b.Max)
	}

	torus, _ := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 1)
	tb := SurfaceRangeBox(torus)
	for i := 0; i <= 12; i++ {
		for j := 0; j <= 12; j++ {
			p := torus.PointAt(twoPi*float64(i)/12, twoPi*float64(j)/12)
			if !pointNearBox(tb, p, 1e-9) {
				t.Fatalf("torus box [%v, %v] misses %v", tb.Min, tb.Max, p)
			}
		}
	}

	plane, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if pb := SurfaceRangeBox(plane); !stdmath.IsInf(pb.Max.X, 1) {
		t.Error("a plane must report an infinite range box")
	}

	bsp := evaluatorBSplineSurface(t)
	bb := SurfaceRangeBox(bsp)
	for i := 0; i <= 10; i++ {
		for j := 0; j <= 10; j++ {
			if p := bsp.PointAt(float64(i)/10, float64(j)/10); !pointNearBox(bb, p, 1e-9) {
				t.Fatalf("bspline net box misses surface point %v", p)
			}
		}
	}
}

// TestSurfaceAnomalyAndContinuity pins the classification members.
func TestSurfaceAnomalyAndContinuity(t *testing.T) {
	sphere, _ := NewSphere(math.P3(0, 0, 0), 1)
	uA, vA := SurfaceAnomaly(sphere)
	if !uA.Periodic || uA.Period != twoPi || !vA.Singular {
		t.Errorf("sphere anomaly = (%+v, %+v)", uA, vA)
	}
	cone, _ := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), 0.4)
	if _, vA := SurfaceAnomaly(cone); !vA.Singular || !vA.Unbounded {
		t.Errorf("cone v anomaly = %+v", vA)
	}
	if got := SurfaceContinuity(sphere); got != ContinuityInfinite {
		t.Errorf("sphere continuity = %d", got)
	}
	if got := SurfaceContinuity(evaluatorBSplineSurface(t)); got != 1 {
		t.Errorf("biquadratic patch with simple interior u-knot: continuity = %d, want 1", got)
	}
}

// TestSurfaceDersMatchExistingFirstOrder ties SurfaceDersAt to the
// long-standing DerivativesAt implementation.
func TestSurfaceDersMatchExistingFirstOrder(t *testing.T) {
	s := evaluatorBSplineSurface(t)
	u, v := 0.3, 0.7
	ders := s.SurfaceDersAt(u, v, 1, 1)
	du, dv := s.DerivativesAt(u, v)
	if float64(ders[1][0].Sub(du).Length()) > 1e-9 || float64(ders[0][1].Sub(dv).Length()) > 1e-9 {
		t.Errorf("SurfaceDersAt first order (%v, %v) disagrees with DerivativesAt (%v, %v)",
			ders[1][0], ders[0][1], du, dv)
	}
	if pos := s.PointAt(u, v); float64(ders[0][0].Sub(pos.AsVector()).Length()) > 1e-9 {
		t.Errorf("SurfaceDersAt position %v disagrees with PointAt %v", ders[0][0], pos)
	}
}

// TestCurveDersMatchExistingFirstOrder ties DersAt to TangentAt.
func TestCurveDersMatchExistingFirstOrder(t *testing.T) {
	curves := evaluatorCurves3(t)
	b := curves["bspline"].(BSplineCurve)
	ders := b.DersAt(0.37, 1)
	if got, want := ders[1], b.TangentAt(0.37); float64(got.Sub(want).Length()) > 1e-9 {
		t.Errorf("DersAt order 1 = %v, TangentAt = %v", got, want)
	}
	if got, want := ders[0], b.PointAt(0.37).AsVector(); float64(got.Sub(want).Length()) > 1e-9 {
		t.Errorf("DersAt order 0 = %v, PointAt = %v", got, want)
	}
}
