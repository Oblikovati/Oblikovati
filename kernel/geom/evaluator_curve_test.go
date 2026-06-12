// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// evaluatorCurves3 returns one representative of every 3D curve kind with
// non-trivial orientation, so closed forms are exercised off the world axes.
func evaluatorCurves3(t *testing.T) map[string]Curve3 {
	t.Helper()
	circle, err := NewCircle(math.P3(1, 2, 3), math.V3(1, 1, 1), 2)
	if err != nil {
		t.Fatalf("circle: %v", err)
	}
	arc, err := NewArc3d(math.P3(-1, 0, 2), math.V3(0, 1, 1), math.V3(1, 0, 0), 1.5, 0.4, 1.8)
	if err != nil {
		t.Fatalf("arc: %v", err)
	}
	ellipse, err := NewEllipseFull(math.P3(0, 1, 0), math.V3(0, 0, 1), math.V3(1, 1, 0), 3, 1)
	if err != nil {
		t.Fatalf("ellipse: %v", err)
	}
	earc, err := NewEllipticalArc(math.P3(2, 0, 1), math.V3(1, 0, 1), math.V3(0, 1, 0), 2, 0.5, 0.3, 2.1)
	if err != nil {
		t.Fatalf("elliptical arc: %v", err)
	}
	helix, err := NewHelix3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 0.8, 1.0, 0.2, 3, false)
	if err != nil {
		t.Fatalf("helix: %v", err)
	}
	bsp, err := NewBSplineCurve(3,
		[]math.Point3{{X: 0}, {X: 1, Y: 2}, {X: 2, Y: -1, Z: 1}, {X: 3, Y: 1}, {X: 4, Z: -1}},
		[]float64{1, 2, 1, 0.5, 1}, []float64{0, 0, 0, 0, 0.5, 1, 1, 1, 1})
	if err != nil {
		t.Fatalf("bspline: %v", err)
	}
	return map[string]Curve3{
		"segment": NewLineSegment(math.P3(1, 1, 1), math.P3(4, 5, 1)),
		"circle":  circle, "arc": arc, "ellipse": ellipse, "ellipticalArc": earc,
		"helix": helix, "bspline": bsp,
	}
}

// TestCurveDerivativesMatchFiniteDifferences uses central differences of
// PointAt as the oracle for the closed-form derivatives.
func TestCurveDerivativesMatchFiniteDifferences(t *testing.T) {
	for name, c := range evaluatorCurves3(t) {
		// Sample away from 0.5: the cubic's third derivative jumps at its
		// interior knot, where a straddling finite difference is meaningless.
		for _, tp := range []float64{0.21, 0.43, 0.83} {
			d1, d2, d3 := CurveDerivatives3(c, tp)
			f1, f2, f3 := fdDers3(c, tp)
			scale := stdmath.Max(1, float64(d1.Length()))
			if float64(d1.Sub(f1).Length()) > 1e-5*scale {
				t.Errorf("%s d1(%g) = %v, finite difference %v", name, tp, d1, f1)
			}
			if float64(d2.Sub(f2).Length()) > 1e-3*stdmath.Max(1, float64(d2.Length())) {
				t.Errorf("%s d2(%g) = %v, finite difference %v", name, tp, d2, f2)
			}
			if float64(d3.Sub(f3).Length()) > 1e-2*stdmath.Max(1, float64(d3.Length())) {
				t.Errorf("%s d3(%g) = %v, finite difference %v", name, tp, d3, f3)
			}
		}
	}
}

// fdDers3 estimates three derivative orders with per-order step sizes.
func fdDers3(c Curve3, t float64) (d1, d2, d3 math.Vector3) {
	const h1, h2, h3 = 1e-6, 1e-5, 1e-4
	d1 = c.PointAt(t + h1).AsVector().Sub(c.PointAt(t - h1).AsVector()).Scale(1 / (2 * h1))
	d2 = c.PointAt(t + h2).AsVector().Add(c.PointAt(t - h2).AsVector()).
		Sub(c.PointAt(t).AsVector().Scale(2)).Scale(1 / (h2 * h2))
	d3 = c.PointAt(t + 2*h3).AsVector().Sub(c.PointAt(t + h3).AsVector().Scale(2)).
		Add(c.PointAt(t - h3).AsVector().Scale(2)).Sub(c.PointAt(t - 2*h3).AsVector()).
		Scale(1 / (2 * h3 * h3 * h3))
	return d1, d2, d3
}

// TestCurveCurvatureClosedForms pins the analytic curvature of the circle and
// the cylindrical helix.
func TestCurveCurvatureClosedForms(t *testing.T) {
	circle, _ := NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	dir, k := CurveCurvature3(circle, 0.125)
	if stdmath.Abs(k-0.5) > 1e-12 {
		t.Errorf("circle curvature = %g, want 1/R = 0.5", k)
	}
	toCenter := circle.PointAt(0.125).VectorTo(circle.Center)
	if float64(dir.Sub(unitOrZero(toCenter)).Length()) > 1e-9 {
		t.Errorf("circle curvature direction %v should point at the center", dir)
	}

	// Cylindrical helix: κ = r / (r² + c²) with c = pitch/2π.
	helix, _ := NewHelix3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 0.8, 1.0, 0, 5, false)
	cc := 1.0 / twoPi
	want := 0.8 / (0.8*0.8 + cc*cc)
	if _, k := CurveCurvature3(helix, 0.4); stdmath.Abs(k-want) > 1e-9*want {
		t.Errorf("helix curvature = %g, want %g", k, want)
	}

	if dir, k := CurveCurvature3(NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0)), 0.5); k != 0 || dir != (math.Vector3{}) {
		t.Errorf("segment curvature = (%v, %g), want zero", dir, k)
	}
}

// TestCurveLengthAgreesWithDenseSampling uses a fine chord sum as the oracle.
func TestCurveLengthAgreesWithDenseSampling(t *testing.T) {
	for name, c := range evaluatorCurves3(t) {
		got := CurveLength3(c, 0.1, 0.9)
		want := chordLength3(c, 0.1, 0.9, 20000)
		if stdmath.Abs(got-want) > 1e-5*stdmath.Max(1, want) {
			t.Errorf("%s Length(0.1, 0.9) = %g, chord oracle %g", name, got, want)
		}
		if rev := CurveLength3(c, 0.9, 0.1); rev != got {
			t.Errorf("%s length must be order-agnostic: %g vs %g", name, rev, got)
		}
	}
	circle, _ := NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	if got := CurveLength3(circle, 0, 1); stdmath.Abs(got-twoPi*3) > 1e-12 {
		t.Errorf("full circle length = %g, want 2πR", got)
	}
}

// chordLength3 sums n chords across [a, b].
func chordLength3(c Curve3, a, b float64, n int) float64 {
	total, prev := 0.0, c.PointAt(a)
	for i := 1; i <= n; i++ {
		next := c.PointAt(a + (b-a)*float64(i)/float64(n))
		total += prev.DistanceTo(next)
		prev = next
	}
	return total
}

// TestParamAtLengthInvertsLength checks ParamAtLength ∘ Length ≈ identity.
func TestParamAtLengthInvertsLength(t *testing.T) {
	for name, c := range evaluatorCurves3(t) {
		for _, target := range []float64{0.35, 0.7} {
			l := CurveLength3(c, 0.1, target)
			back := CurveParamAtLength3(c, 0.1, l)
			if stdmath.Abs(back-target) > 1e-6 {
				t.Errorf("%s ParamAtLength(0.1, Length(0.1, %g)) = %g", name, target, back)
			}
		}
		if got := CurveParamAtLength3(c, 0.5, 0); got != 0.5 {
			t.Errorf("%s zero length must return the start parameter, got %g", name, got)
		}
	}
}

// TestStrokesStayWithinTolerance verifies every dense curve sample sits within
// tolerance of the stroked polyline.
func TestStrokesStayWithinTolerance(t *testing.T) {
	const tol = 1e-3
	for name, c := range evaluatorCurves3(t) {
		pts := CurveStrokes3(c, 0, 1, tol)
		if len(pts) < 3 {
			t.Errorf("%s strokes produced only %d points", name, len(pts))
			continue
		}
		worst := 0.0
		for i := 0; i <= 500; i++ {
			p := c.PointAt(float64(i) / 500)
			worst = stdmath.Max(worst, distanceToPolyline(pts, p))
		}
		if worst > tol*1.5 { // strokes bound the chord midpoint; allow slack off-midpoint
			t.Errorf("%s stroke deviation %g exceeds tolerance %g", name, worst, tol)
		}
	}
}

// distanceToPolyline returns the distance from p to the closest chord.
func distanceToPolyline(pts []math.Point3, p math.Point3) float64 {
	best := stdmath.Inf(1)
	for i := 0; i+1 < len(pts); i++ {
		best = stdmath.Min(best, chordDeviation3(pts[i], pts[i+1], p))
	}
	return best
}

// TestCurveParamAtPointClassification pins the SolutionNature cases.
func TestCurveParamAtPointClassification(t *testing.T) {
	// Explicit RefDir = +X so the expected angle is exact (NewCircle picks an
	// arbitrary in-plane reference).
	circle := Circle{Center: math.P3(0, 0, 0), Normal: mustUnit3(t, 0, 0, 1),
		RefDir: mustUnit3(t, 1, 0, 0), Radius: 2}
	if _, nature := CurveParamAtPoint3(circle, math.P3(0, 0, 5)); nature != InfinitelyManySolutions {
		t.Errorf("circle from its axis: nature = %v, want infinitely many", nature)
	}
	tp, nature := CurveParamAtPoint3(circle, math.P3(3, 3, 1))
	if nature != UniqueSolution {
		t.Errorf("circle generic query: nature = %v", nature)
	}
	if want := 0.125; stdmath.Abs(tp-want) > 1e-12 {
		t.Errorf("circle closest param = %g, want %g", tp, want)
	}

	ellipse, _ := NewEllipseFull(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 3, 1)
	if _, nature := CurveParamAtPoint3(ellipse, math.P3(0, 0, 0)); nature != DistinctlyManySolutions {
		t.Errorf("ellipse from its center: nature = %v, want distinctly many", nature)
	}

	poly, _ := NewPolyline([]math.Point3{{X: -1}, {X: -1, Y: 2}, {X: 1, Y: 2}, {X: 1}})
	if _, nature := CurveParamAtPoint3(poly, math.P3(0, 0, 0)); nature != DistinctlyManySolutions {
		t.Errorf("polyline equidistant of two segments: nature = %v", nature)
	}

	seg := NewLineSegment(math.P3(0, 0, 0), math.P3(10, 0, 0))
	tp, nature = CurveParamAtPoint3(seg, math.P3(4, 3, 0))
	if nature != UniqueSolution || stdmath.Abs(tp-0.4) > 1e-12 {
		t.Errorf("segment closest = (%g, %v), want (0.4, unique)", tp, nature)
	}

	helix, _ := NewHelix3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 1, 1, 0, 2, false)
	near := helix.PointAt(0.3).TranslateBy(math.V3(0.01, -0.01, 0.005))
	tp, nature = CurveParamAtPoint3(helix, near)
	if nature != UniqueSolution || stdmath.Abs(tp-0.3) > 0.01 {
		t.Errorf("helix near-curve query = (%g, %v), want ≈0.3 unique", tp, nature)
	}
}

// TestCurveRangeBoxes pins closed-form boxes and containment of sampled boxes.
func TestCurveRangeBoxes(t *testing.T) {
	// Quarter arc in XY from angle 0 to π/2: box [cx, cx+r] × [cy, cy+r].
	arc := Arc3d{Center: math.P3(1, 1, 0), Normal: mustUnit3(t, 0, 0, 1),
		RefDir: mustUnit3(t, 1, 0, 0), Radius: 2, StartAngle: 0, SweepAngle: stdmath.Pi / 2}
	box := CurveRangeBox3(arc)
	wantMin, wantMax := math.P3(1, 1, 0), math.P3(3, 3, 0)
	if box.Min.DistanceTo(wantMin) > 1e-12 || box.Max.DistanceTo(wantMax) > 1e-12 {
		t.Errorf("quarter-arc box = [%v, %v], want [%v, %v]", box.Min, box.Max, wantMin, wantMax)
	}

	for name, c := range evaluatorCurves3(t) {
		b := CurveRangeBox3(c)
		for i := 0; i <= 100; i++ {
			p := c.PointAt(float64(i) / 100)
			if !b.Contains(p) && !pointNearBox(b, p, 1e-9) {
				t.Errorf("%s range box %v..%v misses curve point %v", name, b.Min, b.Max, p)
				break
			}
		}
	}

	line, _ := NewLine(math.P3(0, 0, 0), math.V3(1, 0, 0))
	if b := CurveRangeBox3(line); !stdmath.IsInf(b.Max.X, 1) {
		t.Error("an unbounded line must report an infinite range box")
	}
}

// pointNearBox tolerates floating-point skin on the box faces.
func pointNearBox(b math.Box, p math.Point3, eps float64) bool {
	return p.X >= b.Min.X-eps && p.X <= b.Max.X+eps &&
		p.Y >= b.Min.Y-eps && p.Y <= b.Max.Y+eps &&
		p.Z >= b.Min.Z-eps && p.Z <= b.Max.Z+eps
}

// mustUnit3 builds a unit vector or fails the test.
func mustUnit3(t *testing.T, x, y, z float64) math.UnitVector3 {
	t.Helper()
	u, err := math.NewUnitVector3(x, y, z)
	if err != nil {
		t.Fatalf("unit(%g,%g,%g): %v", x, y, z, err)
	}
	return u
}

// TestCurveAnomalyAndContinuity pins the classification members.
func TestCurveAnomalyAndContinuity(t *testing.T) {
	line, _ := NewLine(math.P3(0, 0, 0), math.V3(0, 1, 0))
	if a := CurveAnomaly3(line); !a.Unbounded || a.Periodic {
		t.Errorf("line anomaly = %+v", a)
	}
	circle, _ := NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 1)
	if a := CurveAnomaly3(circle); !a.Periodic || a.Period != 1 {
		t.Errorf("circle anomaly = %+v", a)
	}
	poly, _ := NewPolyline([]math.Point3{{}, {X: 1}, {X: 1, Y: 1}})
	if a := CurveAnomaly3(poly); !a.Singular {
		t.Errorf("cornered polyline anomaly = %+v", a)
	}
	if got := CurveContinuity3(poly); got != 0 {
		t.Errorf("polyline continuity = %d, want 0", got)
	}
	if got := CurveContinuity3(circle); got != ContinuityInfinite {
		t.Errorf("circle continuity = %d, want C∞", got)
	}
	// Degree 3 with a doubled interior knot: C¹.
	bsp, err := NewBSplineCurveUniformWeights(3,
		[]math.Point3{{}, {X: 1}, {X: 2, Y: 1}, {X: 3}, {X: 4, Y: -1}, {X: 5}},
		[]float64{0, 0, 0, 0, 0.5, 0.5, 1, 1, 1, 1})
	if err != nil {
		t.Fatalf("bspline: %v", err)
	}
	if got := CurveContinuity3(bsp); got != 1 {
		t.Errorf("doubled-knot cubic continuity = %d, want 1", got)
	}

	// A geometrically closed NURBS curve reports its domain span as the period.
	closed, err := NewBSplineCurveUniformWeights(2,
		[]math.Point3{{X: 1}, {Y: 1}, {X: -1}, {Y: -1}, {X: 1}},
		[]float64{0, 0, 0, 0.4, 0.6, 1, 1, 1})
	if err != nil {
		t.Fatalf("closed bspline: %v", err)
	}
	if a := CurveAnomaly3(closed); !a.Periodic || a.Period != 1 {
		t.Errorf("closed bspline anomaly = %+v", a)
	}
}

// TestCurveEndPoints covers bounded and unbounded domains.
func TestCurveEndPoints(t *testing.T) {
	seg := NewLineSegment(math.P3(1, 2, 3), math.P3(4, 5, 6))
	start, end, bounded := CurveEndPoints3(seg)
	if !bounded || start != seg.StartPoint || end != seg.EndPoint {
		t.Errorf("segment end points = (%v, %v, %v)", start, end, bounded)
	}
	line, _ := NewLine(math.P3(0, 0, 0), math.V3(1, 0, 0))
	if _, _, bounded := CurveEndPoints3(line); bounded {
		t.Error("a line must report bounded=false")
	}
}

// TestCurve2dEvaluatorMirrors spot-checks the 2D twins against the same oracles.
func TestCurve2dEvaluatorMirrors(t *testing.T) {
	arc := NewArc2d(math.P2(1, 1), 2, 0.3, 1.9)
	d1, d2, _ := CurveDerivatives2(arc, 0.4)
	const h = 1e-6
	f1 := arc.PointAt(0.4 + h).AsVector().Sub(arc.PointAt(0.4 - h).AsVector()).Scale(1 / (2 * h))
	if float64(d1.Sub(f1).Length()) > 1e-5 {
		t.Errorf("arc2d d1 = %v, finite difference %v", d1, f1)
	}
	if k := CurveCurvature2(arc, 0.4); stdmath.Abs(k-0.5) > 1e-12 {
		t.Errorf("arc2d signed curvature = %g, want 1/R = 0.5 (counter-clockwise)", k)
	}
	_ = d2

	circle := NewCircle2d(math.P2(0, 0), 3)
	if got := CurveLength2(circle, 0, 1); stdmath.Abs(got-twoPi*3) > 1e-12 {
		t.Errorf("circle2d length = %g", got)
	}
	if _, nature := CurveParamAtPoint2(circle, math.P2(0, 0)); nature != InfinitelyManySolutions {
		t.Errorf("circle2d from center: nature = %v", nature)
	}
	tp, nature := CurveParamAtPoint2(circle, math.P2(0, 7))
	if nature != UniqueSolution || stdmath.Abs(tp-0.25) > 1e-12 {
		t.Errorf("circle2d closest = (%g, %v)", tp, nature)
	}

	ell, _ := NewEllipseFull2d(math.P2(0, 0), math.V2(1, 0), 3, 1)
	l := CurveLength2(ell, 0.1, 0.6)
	back := CurveParamAtLength2(ell, 0.1, l)
	if stdmath.Abs(back-0.6) > 1e-6 {
		t.Errorf("ellipse2d length round trip = %g, want 0.6", back)
	}
	pts := CurveStrokes2(ell, 0, 1, 1e-3)
	if len(pts) < 8 {
		t.Errorf("ellipse2d strokes produced only %d points", len(pts))
	}
	box := CurveRangeBox2(ell)
	if box.Min.DistanceTo(math.P2(-3, -1)) > 1e-12 || box.Max.DistanceTo(math.P2(3, 1)) > 1e-12 {
		t.Errorf("ellipse2d box = [%v, %v]", box.Min, box.Max)
	}
}
