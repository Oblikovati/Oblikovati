// SPDX-License-Identifier: GPL-2.0-only

package geomapi

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
)

// TestCurveEvaluatorThroughContract drives the full contract path: factory →
// curve → evaluator, checking the members against the circle's closed forms.
func TestCurveEvaluatorThroughContract(t *testing.T) {
	f := New()
	circle, err := f.CreateCircle(types.NewPoint(0, 0, 0), types.UnitVector{X: 0, Y: 0, Z: 1}, 2)
	if err != nil {
		t.Fatalf("CreateCircle: %v", err)
	}
	ev := circle.Evaluator()

	if got := ev.Length(0, 1); stdmath.Abs(got-2*stdmath.Pi*2) > 1e-12 {
		t.Errorf("circle length = %g, want 2πR", got)
	}
	dir, k := ev.Curvature(0.3)
	if stdmath.Abs(k-0.5) > 1e-12 {
		t.Errorf("circle curvature = %g, want 0.5", k)
	}
	foot := circle.Evaluate(0.3)
	toCenter := types.NewVector(-foot.X, -foot.Y, -foot.Z)
	if dot := dir.Dot(toCenter); dot < 0.99*toCenter.Length() {
		t.Errorf("curvature direction %+v should point at the center", dir)
	}
	if back := ev.ParamAtLength(0.1, ev.Length(0.1, 0.6)); stdmath.Abs(back-0.6) > 1e-9 {
		t.Errorf("length round trip = %g, want 0.6", back)
	}
	if _, nature := ev.ParamAtPoint(types.NewPoint(0, 0, 4)); nature != types.InfinitelyManySolutions {
		t.Errorf("axis query nature = %v", nature)
	}
	box := ev.RangeBox()
	if box.Min.DistanceTo(types.NewPoint(-2, -2, 0)) > 1e-9 || box.Max.DistanceTo(types.NewPoint(2, 2, 0)) > 1e-9 {
		t.Errorf("circle range box = %+v", box)
	}
	if a := ev.ParamAnomaly(); !a.Periodic || a.Period != 1 {
		t.Errorf("circle anomaly = %+v", a)
	}
	if ev.Continuity() != types.ContinuityInfinite {
		t.Errorf("circle continuity = %d", ev.Continuity())
	}
	if pts := ev.Strokes(0, 1, 1e-3); len(pts) < 8 {
		t.Errorf("circle strokes = %d points", len(pts))
	}
	start, end, bounded := ev.EndPoints()
	if !bounded || start.DistanceTo(end) > 1e-9 {
		t.Errorf("closed circle end points = (%+v, %+v, %v)", start, end, bounded)
	}
}

// TestCurve2dEvaluatorThroughContract drives the 2D path on a sketch arc.
func TestCurve2dEvaluatorThroughContract(t *testing.T) {
	f := New()
	arc, err := f.CreateArc2d(types.NewPoint2d(1, 1), 2, 0, stdmath.Pi/2)
	if err != nil {
		t.Fatalf("CreateArc2d: %v", err)
	}
	ev := arc.Evaluator()
	if got := ev.Length(0, 1); stdmath.Abs(got-stdmath.Pi) > 1e-12 {
		t.Errorf("quarter-arc length = %g, want πR/2 = π", got)
	}
	if k := ev.Curvature(0.5); stdmath.Abs(k-0.5) > 1e-12 {
		t.Errorf("arc signed curvature = %g, want +0.5", k)
	}
	box := ev.RangeBox()
	if box.Min.DistanceTo(types.NewPoint2d(1, 1)) > 1e-9 || box.Max.DistanceTo(types.NewPoint2d(3, 3)) > 1e-9 {
		t.Errorf("quarter-arc box = %+v", box)
	}
	d1, d2, _ := ev.Derivatives(0.25)
	if speed := d1.Length(); stdmath.Abs(speed-stdmath.Pi) > 1e-9 {
		t.Errorf("arc |d1| = %g, want R·sweep = π", speed)
	}
	if d2.Length() == 0 {
		t.Error("arc second derivative must be nonzero")
	}
}

// TestSurfaceEvaluatorThroughContract drives the surface path on a sphere and
// checks the iso-curve comes back as a contract curve on the surface.
func TestSurfaceEvaluatorThroughContract(t *testing.T) {
	f := New()
	sphere, err := f.CreateSphere(types.NewPoint(1, 2, 3), 2)
	if err != nil {
		t.Fatalf("CreateSphere: %v", err)
	}
	ev := sphere.Evaluator()

	if got := ev.Area(); stdmath.Abs(got-4*stdmath.Pi*4) > 1e-9 {
		t.Errorf("sphere area = %g, want 4πR²", got)
	}
	_, kMax, kMin := ev.Curvatures(0.7, 0.2)
	// Loose umbilic tolerance: see TestSurfaceCurvaturesClosedForms in kernel/geom.
	if stdmath.Abs(kMax-kMin) > 1e-6 || stdmath.Abs(stdmath.Abs(kMax)-0.5) > 1e-7 {
		t.Errorf("sphere curvatures = (%g, %g)", kMax, kMin)
	}
	box := ev.RangeBox()
	if box.Min.DistanceTo(types.NewPoint(-1, 0, 1)) > 1e-9 || box.Max.DistanceTo(types.NewPoint(3, 4, 5)) > 1e-9 {
		t.Errorf("sphere range box = %+v", box)
	}
	if n := ev.NormalAtPoint(types.NewPoint(10, 2, 3)); n.AngleTo(types.NewVector(1, 0, 0)) > 1e-9 {
		t.Errorf("normal at off-surface point = %+v, want +X", n)
	}
	if _, _, nature := ev.ParamAtPoint(types.NewPoint(1, 2, 3)); nature != types.InfinitelyManySolutions {
		t.Errorf("center query nature = %v", nature)
	}

	iso, err := ev.IsoCurve(false, 0.4) // latitude circle
	if err != nil {
		t.Fatalf("IsoCurve: %v", err)
	}
	if iso.CurveType() != types.CircleCurve {
		t.Errorf("latitude iso-curve type = %v, want circle", iso.CurveType())
	}
	p := iso.Evaluate(0.3)
	if d := sphere.Evaluate(sphere.Parameter(p)).DistanceTo(p); d > 1e-9 {
		t.Errorf("iso-curve point %+v sits %g off the sphere", p, d)
	}

	uA, vA := ev.ParamAnomaly()
	if !uA.Periodic || !vA.Singular {
		t.Errorf("sphere anomaly = (%+v, %+v)", uA, vA)
	}
	rect := ev.ParamRangeRect()
	if rect.Max.X != 2*stdmath.Pi || stdmath.Abs(rect.Max.Y-stdmath.Pi/2) > 1e-12 {
		t.Errorf("sphere param rect = %+v", rect)
	}
}

// TestEvaluatorPartialsConsistency cross-checks first partials between the
// surface umbrella and its evaluator.
func TestEvaluatorPartialsConsistency(t *testing.T) {
	f := New()
	torus, err := f.CreateTorus(types.NewPoint(0, 0, 0), types.UnitVector{X: 0, Y: 0, Z: 1}, 3, 1)
	if err != nil {
		t.Fatalf("CreateTorus: %v", err)
	}
	ev := torus.Evaluator()
	pu, pv := ev.FirstPartials(0.5, 1.1)
	ut, vt := ev.Tangents(0.5, 1.1)
	if pu.AngleTo(ut) > 1e-9 || pv.AngleTo(vt) > 1e-9 {
		t.Error("tangents must be the normalized first partials")
	}
	puu, _, _ := ev.SecondPartials(0.5, 1.1)
	puuu, pvvv := ev.ThirdPartials(0.5, 1.1)
	if puu.Length() == 0 || puuu.Length() == 0 || pvvv.Length() == 0 {
		t.Error("torus higher partials must be nonzero")
	}
	// Angular directions: ∂³P/∂u³ = −∂P/∂u on a torus.
	sum := types.NewVector(puuu.X+pu.X, puuu.Y+pu.Y, puuu.Z+pu.Z)
	if sum.Length() > 1e-9 {
		t.Errorf("torus ∂³/∂u³ should equal −∂/∂u, residual %g", sum.Length())
	}
}

// TestEvaluatorInterfacesAreReachable asserts every umbrella exposes its
// evaluator (compile-time coverage for the contract addition).
func TestEvaluatorInterfacesAreReachable(t *testing.T) {
	f := New()
	seg, err := f.CreateLineSegment(types.NewPoint(0, 0, 0), types.NewPoint(1, 0, 0))
	if err != nil {
		t.Fatalf("CreateLineSegment: %v", err)
	}
	assertCurveEvaluator(seg.Evaluator())
	seg2, err := f.CreateLineSegment2d(types.NewPoint2d(0, 0), types.NewPoint2d(1, 0))
	if err != nil {
		t.Fatalf("CreateLineSegment2d: %v", err)
	}
	assertCurve2dEvaluator(seg2.Evaluator())
	plane, err := f.CreatePlane(types.NewPoint(0, 0, 0), types.UnitVector{X: 0, Y: 0, Z: 1})
	if err != nil {
		t.Fatalf("CreatePlane: %v", err)
	}
	assertSurfaceEvaluator(plane.Evaluator())
	if !stdmath.IsInf(plane.Evaluator().Area(), 1) {
		t.Error("plane area must be +Inf")
	}
}

// The assert helpers pin the static types of the Evaluator() accessors.
func assertCurveEvaluator(contract.CurveEvaluator)     {}
func assertCurve2dEvaluator(contract.Curve2dEvaluator) {}
func assertSurfaceEvaluator(contract.SurfaceEvaluator) {}
