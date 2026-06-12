// SPDX-License-Identifier: GPL-2.0-only

package geomapi

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
)

// unit is a test shorthand for a known-good unit vector.
func unit(t *testing.T, x, y, z float64) types.UnitVector {
	t.Helper()
	u, err := types.NewUnitVector(x, y, z)
	if err != nil {
		t.Fatalf("unit(%g,%g,%g): %v", x, y, z, err)
	}
	return u
}

func TestFactoryCircleEvaluatesOnTheCircle(t *testing.T) {
	tg := New()
	circle, err := tg.CreateCircle(types.NewPoint(1, 2, 3), unit(t, 0, 0, 1), 2)
	if err != nil {
		t.Fatalf("CreateCircle: %v", err)
	}
	if circle.CurveType() != types.CircleCurve || circle.GeometryForm() != types.CurveFormNotNURBS {
		t.Errorf("discriminators = %v/%v", circle.CurveType(), circle.GeometryForm())
	}
	if circle.Radius() != 2 || circle.Center() != types.NewPoint(1, 2, 3) {
		t.Errorf("getters = r%v c%+v", circle.Radius(), circle.Center())
	}
	// Every sample sits at radius distance from the center, in the plane.
	for _, s := range []float64{0, 0.25, 0.5, 0.9} {
		p := circle.Evaluate(s)
		if d := p.DistanceTo(circle.Center()); stdmath.Abs(d-2) > 1e-12 {
			t.Errorf("|P(%v)-C| = %v, want 2", s, d)
		}
		if stdmath.Abs(p.Z-3) > 1e-12 {
			t.Errorf("P(%v).Z = %v, want the circle plane z=3", s, p.Z)
		}
	}
}

func TestFactoryRejectsDegenerateInput(t *testing.T) {
	tg := New()
	if _, err := tg.CreateCircle(types.NewPoint(0, 0, 0), unit(t, 0, 0, 1), -1); err == nil {
		t.Error("negative radius must be rejected")
	}
	if _, err := tg.CreateLineSegment(types.NewPoint(1, 1, 1), types.NewPoint(1, 1, 1)); err == nil {
		t.Error("a zero-length segment must be rejected")
	}
	if _, err := tg.CreateCircle2d(types.NewPoint2d(0, 0), 0); err == nil {
		t.Error("a zero 2D radius must be rejected")
	}
	if _, err := tg.CreatePolyline(nil); err == nil {
		t.Error("an empty polyline must be rejected")
	}
	if _, err := tg.CreateBSplineSurface(types.BSplineSurfaceDef{PolesU: 2, PolesV: 2}); err == nil {
		t.Error("a pole-less surface net must be rejected")
	}
}

func TestFactoryArcGettersAndTangent(t *testing.T) {
	tg := New()
	arc, err := tg.CreateArc(types.NewPoint(0, 0, 0), unit(t, 0, 0, 1), unit(t, 1, 0, 0), 1, 0, stdmath.Pi/2)
	if err != nil {
		t.Fatalf("CreateArc: %v", err)
	}
	if arc.SweepAngle() != stdmath.Pi/2 || arc.Radius() != 1 {
		t.Errorf("getters = sweep %v r %v", arc.SweepAngle(), arc.Radius())
	}
	start, end := arc.Evaluate(0), arc.Evaluate(1)
	if !start.IsEqualTo(types.NewPoint(1, 0, 0), 1e-12) || !end.IsEqualTo(types.NewPoint(0, 1, 0), 1e-12) {
		t.Errorf("arc span = %+v → %+v, want +X → +Y", start, end)
	}
	// The tangent at the start of a CCW quarter arc points along +Y.
	tan := arc.Tangent(0)
	if tan.X > 1e-9 || tan.Y <= 0 {
		t.Errorf("tangent(0) = %+v, want +Y direction", tan)
	}
}

func TestFactoryBSplineDefinitionRoundTrips(t *testing.T) {
	tg := New()
	def := types.BSplineCurveDef{
		Degree: 2,
		Poles:  []types.Point{{X: 0}, {X: 1, Y: 2}, {X: 2}, {X: 3, Y: -1}},
		Knots:  []float64{0, 0, 0, 0.5, 1, 1, 1},
	}
	spline, err := tg.CreateBSplineCurve(def)
	if err != nil {
		t.Fatalf("CreateBSplineCurve: %v", err)
	}
	if spline.GeometryForm() != types.CurveFormNURBS {
		t.Errorf("form = %v, want NURBS", spline.GeometryForm())
	}
	got := spline.Definition()
	if got.Degree != 2 || len(got.Poles) != 4 || len(got.Knots) != 7 {
		t.Fatalf("definition = %+v, want the input recipe back", got)
	}
	if got.Poles[1] != def.Poles[1] {
		t.Errorf("pole[1] = %+v, want %+v", got.Poles[1], def.Poles[1])
	}
}

func TestFactorySphereEvaluation(t *testing.T) {
	tg := New()
	sphere, err := tg.CreateSphere(types.NewPoint(0, 0, 0), 3)
	if err != nil {
		t.Fatalf("CreateSphere: %v", err)
	}
	if sphere.SurfaceType() != types.SphereSurface {
		t.Errorf("kind = %v", sphere.SurfaceType())
	}
	p := sphere.Evaluate(0.3, -0.7)
	if d := p.DistanceTo(types.NewPoint(0, 0, 0)); stdmath.Abs(d-3) > 1e-12 {
		t.Errorf("|P| = %v, want 3", d)
	}
	// The unit normal of a centered sphere points along the position.
	n := sphere.Normal(0.3, -0.7)
	want := types.NewPoint(0, 0, 0).VectorTo(p).Scale(1.0 / 3.0)
	if !n.IsEqualTo(want, 1e-9) {
		t.Errorf("normal = %+v, want radial %+v", n, want)
	}
	// Parameter inverts Evaluate.
	u, v := sphere.Parameter(p)
	if q := sphere.Evaluate(u, v); !q.IsEqualTo(p, 1e-9) {
		t.Errorf("Evaluate(Parameter(p)) = %+v, want %+v", q, p)
	}
}

func TestFactoryEllipticalQuadricsAreFirstClass(t *testing.T) {
	tg := New()
	cyl, err := tg.CreateEllipticalCylinder(types.NewPoint(0, 0, 0), unit(t, 0, 0, 1), unit(t, 1, 0, 0), 3, 1)
	if err != nil {
		t.Fatalf("CreateEllipticalCylinder: %v", err)
	}
	if cyl.SurfaceType() != types.EllipticalCylinderSurface || cyl.MajorRadius() != 3 {
		t.Errorf("cylinder = %v r %v", cyl.SurfaceType(), cyl.MajorRadius())
	}
	cone, err := tg.CreateEllipticalCone(types.NewPoint(0, 0, 0), unit(t, 0, 0, 1), unit(t, 1, 0, 0), 0.5, 0.3)
	if err != nil {
		t.Fatalf("CreateEllipticalCone: %v", err)
	}
	if cone.SurfaceType() != types.EllipticalConeSurface || cone.MajorHalfAngle() != 0.5 {
		t.Errorf("cone = %v a %v", cone.SurfaceType(), cone.MajorHalfAngle())
	}
}

func TestFactoryHelixIsTheHouseExtension(t *testing.T) {
	tg := New()
	helix, err := tg.CreateHelix(types.NewPoint(0, 0, 0), unit(t, 0, 0, 1), unit(t, 1, 0, 0), 2, 1, 0, 3, false)
	if err != nil {
		t.Fatalf("CreateHelix: %v", err)
	}
	if helix.CurveType() != types.HelixCurve || helix.Turns() != 3 || helix.Pitch() != 1 {
		t.Errorf("helix = %v turns %v pitch %v", helix.CurveType(), helix.Turns(), helix.Pitch())
	}
	// After all turns the curve has climbed turns×pitch along the axis.
	end := helix.Evaluate(1)
	if stdmath.Abs(end.Z-3) > 1e-9 {
		t.Errorf("end.Z = %v, want 3", end.Z)
	}
}

// TestAdaptersSatisfyUmbrellas pins the assertion set: every adapter is usable
// through the umbrella interfaces.
func TestAdaptersSatisfyUmbrellas(t *testing.T) {
	tg := New()
	seg, _ := tg.CreateLineSegment(types.NewPoint(0, 0, 0), types.NewPoint(1, 0, 0))
	var c contract.Curve = seg
	if c.CurveType() != types.LineSegmentCurve {
		t.Errorf("umbrella curve kind = %v", c.CurveType())
	}
	circle2d, _ := tg.CreateCircle2d(types.NewPoint2d(0, 0), 1)
	var c2 contract.Curve2d = circle2d
	if lo, hi := c2.Domain(); lo != 0 || hi != 1 {
		t.Errorf("2D domain = [%v, %v]", lo, hi)
	}
	plane, _ := tg.CreatePlane(types.NewPoint(0, 0, 0), unit(t, 0, 0, 1))
	var srf contract.Surface = plane
	if srf.SurfaceType() != types.PlaneSurface {
		t.Errorf("umbrella surface kind = %v", srf.SurfaceType())
	}
	if n := plane.PlaneNormal(); stdmath.Abs(n.Z) < 0.999 {
		t.Errorf("plane normal = %+v, want ±Z", n)
	}
}
