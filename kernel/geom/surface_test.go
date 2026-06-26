// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// sampleSurface pairs a surface with an interior (u, v) for the generic
// (metamorphic) normal and derivative checks.
type sampleSurface struct {
	name string
	s    Surface
	u, v float64
}

func sampleSurfaces(t *testing.T) []sampleSurface {
	t.Helper()
	plane, _ := NewPlane(math.P3(1, 2, 3), math.V3(0, 0, 1))
	cyl, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	cone, _ := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), 0.5)
	sph, _ := NewSphere(math.P3(0, 0, 0), 2)
	tor, _ := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1)
	return []sampleSurface{
		{"plane", plane, 0.3, 0.7},
		{"cylinder", cyl, 1.0, 0.5},
		{"cone", cone, 1.0, 2.0},
		{"sphere", sph, 1.0, 0.3},
		{"torus", tor, 1.0, 0.8},
		{"bspline", sampleBSplineSurface(t), 0.5, 0.5},
	}
}

func TestSurfaceNormalMatchesPartials(t *testing.T) {
	for _, c := range sampleSurfaces(t) {
		du, dv := c.s.DerivativesAt(c.u, c.v)
		want := normalFromPartials(du, dv)
		if got := c.s.NormalAt(c.u, c.v); !got.IsEqualTo(want, 1e-9) {
			t.Errorf("%s: NormalAt = %v, want du×dv normalized = %v", c.name, got, want)
		}
	}
}

func TestSurfacePartialsMatchFiniteDifference(t *testing.T) {
	const h = 1e-6
	for _, c := range sampleSurfaces(t) {
		du, dv := c.s.DerivativesAt(c.u, c.v)
		fdu := c.s.PointAt(c.u-h, c.v).VectorTo(c.s.PointAt(c.u+h, c.v)).Scale(1 / (2 * h))
		fdv := c.s.PointAt(c.u, c.v-h).VectorTo(c.s.PointAt(c.u, c.v+h)).Scale(1 / (2 * h))
		if !du.IsEqualTo(fdu, 1e-5) {
			t.Errorf("%s: ∂u = %v, finite-diff %v", c.name, du, fdu)
		}
		if !dv.IsEqualTo(fdv, 1e-5) {
			t.Errorf("%s: ∂v = %v, finite-diff %v", c.name, dv, fdv)
		}
	}
}

func TestPlanePointsLieInPlane(t *testing.T) {
	p, _ := NewPlane(math.P3(0, 0, 1), math.V3(0, 0, 1))
	for _, uv := range [][2]float64{{0, 0}, {3, -2}, {10, 7}} {
		pt := p.PointAt(uv[0], uv[1])
		offset := p.Origin.VectorTo(pt).Dot(p.Normal())
		approxScalar(t, offset, 0, "plane offset")
	}
}

func TestSpherePointAndNormalReference(t *testing.T) {
	s, _ := NewSphere(math.P3(1, 1, 1), 2)
	for _, uv := range [][2]float64{{0, 0}, {1.2, 0.4}, {3, -0.5}, {0, stdmath.Pi / 2}} {
		pt := s.PointAt(uv[0], uv[1])
		approxScalar(t, s.Center.DistanceTo(pt), 2, "sphere radius")
		want := s.Center.VectorTo(pt).Scale(1.0 / 2) // outward unit
		if got := s.NormalAt(uv[0], uv[1]); !got.IsEqualTo(want, 1e-12) {
			t.Errorf("sphere normal = %v, want %v", got, want)
		}
	}
}

func TestCylinderRadiusAndNormal(t *testing.T) {
	c, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	for _, uv := range [][2]float64{{0, 0}, {1, 5}, {4, -3}} {
		pt := c.PointAt(uv[0], uv[1])
		approxScalar(t, stdmath.Hypot(pt.X, pt.Y), 3, "cylinder radius") // axis is +Z
		n := c.NormalAt(uv[0], uv[1])
		approxScalar(t, n.Length(), 1, "cylinder normal unit")
		if !n.IsPerpendicularTo(c.AxisDir.AsVector(), 1e-9) {
			t.Errorf("cylinder normal not perpendicular to axis: %v", n)
		}
	}
}

// TestCylinderWithRefPinsAngleZero proves NewCylinderWithRef parameterizes angle 0 along the
// in-plane projection of refHint (the axial component dropped), so an extruded circle can record
// its generating sketch +X and be re-faceted in that exact frame (#129).
func TestCylinderWithRefPinsAngleZero(t *testing.T) {
	// refHint tilted out of the axis plane: its axial (+Z) component must be dropped.
	c, err := NewCylinderWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 5), 3)
	if err != nil {
		t.Fatalf("NewCylinderWithRef: %v", err)
	}
	p0 := c.PointAt(0, 0) // angle 0 ⇒ Origin + Radius·Ref
	approxScalar(t, p0.X, 3, "angle-0 lies on +X (refHint projection)")
	approxScalar(t, p0.Y, 0, "angle-0 has no Y component")
	approxScalar(t, p0.Z, 0, "angle-0 has no axial component")
	if !c.Ref.AsVector().IsPerpendicularTo(c.AxisDir.AsVector(), 1e-9) {
		t.Errorf("Ref not perpendicular to axis: %v", c.Ref)
	}
}

// TestCylinderWithRefParallelHintErrors guards the degenerate case: a refHint parallel to the axis
// has no in-plane component to use as angle 0.
func TestCylinderWithRefParallelHintErrors(t *testing.T) {
	if _, err := NewCylinderWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(0, 0, 2), 3); err == nil {
		t.Fatal("expected an error for a refHint parallel to the axis")
	}
}

func TestConeRadiusGrowsWithDistance(t *testing.T) {
	half := 0.6
	c, _ := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), half)
	for _, v := range []float64{1, 3, 7} {
		pt := c.PointAt(0.8, v)
		approxScalar(t, stdmath.Hypot(pt.X, pt.Y), v*stdmath.Tan(half), "cone radius at v")
		approxScalar(t, c.NormalAt(0.8, v).Length(), 1, "cone normal unit")
	}
}

func TestTorusLiesOnTube(t *testing.T) {
	major, minor := 5.0, 1.5
	tor, _ := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), major, minor)
	for _, uv := range [][2]float64{{0, 0}, {1, 2}, {4, 5}} {
		pt := tor.PointAt(uv[0], uv[1])
		axial := pt.Z // axis is +Z through origin
		planar := stdmath.Hypot(pt.X, pt.Y)
		// (planarDist − Major)² + axial² == Minor².
		got := (planar-major)*(planar-major) + axial*axial
		approxScalar(t, got, minor*minor, "torus tube")
	}
}

// TestTorusParamAtInvertsPointAt: ParamAt is the exact inverse of PointAt over the (u,v) torus — a sampled
// (u,v) round-trips back to itself (within wrap). The (u,v)-arrangement torus trimmer (Oblikovati#1406)
// inverts each sampled spiric point through ParamAt, so the inverse must be tight, not just close.
func TestTorusParamAtInvertsPointAt(t *testing.T) {
	tor, _ := NewTorus(math.P3(1, -2, 3), math.V3(0, 0, 1), 5, 2)
	for i := 0; i < 12; i++ {
		u := 2 * stdmath.Pi * float64(i) / 12
		for j := 0; j < 12; j++ {
			v := 2 * stdmath.Pi * float64(j) / 12
			gu, gv := tor.ParamAt(tor.PointAt(u, v))
			if du := angleGap(gu, u); du > 1e-9 {
				t.Errorf("ParamAt(PointAt(%.4f,%.4f)).u = %.9f, want %.4f (gap %.2e)", u, v, gu, u, du)
			}
			if dv := angleGap(gv, v); dv > 1e-9 {
				t.Errorf("ParamAt(PointAt(%.4f,%.4f)).v = %.9f, want %.4f (gap %.2e)", u, v, gv, v, dv)
			}
		}
	}
}

// angleGap returns the absolute difference between two angles, accounting for the 0≡2π wrap.
func angleGap(a, b float64) float64 {
	d := stdmath.Abs(a - b)
	if d > stdmath.Pi {
		d = 2*stdmath.Pi - d
	}
	return d
}

func TestSurfaceDomainsAreOrdered(t *testing.T) {
	for _, c := range sampleSurfaces(t) {
		ulo, uhi := c.s.UDomain()
		vlo, vhi := c.s.VDomain()
		if ulo >= uhi || vlo >= vhi {
			t.Errorf("%s: domains U[%v,%v] V[%v,%v] must each have lo < hi", c.name, ulo, uhi, vlo, vhi)
		}
	}
}

func TestNewPlaneFromAxes(t *testing.T) {
	// Non-orthogonal V is orthogonalized against U; the plane normal is +Z.
	p, err := NewPlaneFromAxes(math.P3(0, 0, 0), math.V3(2, 0, 0), math.V3(1, 1, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.Normal().IsEqualTo(math.V3(0, 0, 1), 1e-12) {
		t.Errorf("normal = %v, want +Z", p.Normal())
	}
	if !p.UAxis.IsPerpendicularTo(p.VAxis, 1e-12) {
		t.Error("U and V axes should be orthogonal after Gram-Schmidt")
	}
	if _, err := NewPlaneFromAxes(math.P3(0, 0, 0), math.V3(1, 0, 0), math.V3(2, 0, 0)); err == nil {
		t.Error("parallel axes should error")
	}
}

func TestNURBSSurfaceWeightNetMismatchErrors(t *testing.T) {
	ctrl := [][]math.Point3{
		{math.P3(0, 0, 0), math.P3(0, 1, 0)},
		{math.P3(1, 0, 0), math.P3(1, 1, 0)},
	}
	badWeights := [][]float64{{1, 1}} // only one row for a 2×2 net
	if _, err := NewBSplineSurface(1, 1, ctrl, badWeights, []float64{0, 0, 1, 1}, []float64{0, 0, 1, 1}); err == nil {
		t.Error("weight net not matching control net should error")
	}
}

func TestSurfaceConstructorErrors(t *testing.T) {
	if _, err := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 0)); err == nil {
		t.Error("zero normal plane should error")
	}
	if _, err := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), 0); err == nil {
		t.Error("zero half-angle cone should error")
	}
	if _, err := NewSphere(math.P3(0, 0, 0), -1); err == nil {
		t.Error("negative-radius sphere should error")
	}
	if _, err := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 0, 1); err == nil {
		t.Error("zero major-radius torus should error")
	}
}
