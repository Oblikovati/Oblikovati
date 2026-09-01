// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// The oracle below is deliberately independent of the machinery under test: it evaluates the EXACT
// crossing-cylinder intersection curve in closed form and measures a chord midpoint's distance to each
// cylinder radially, never through ClosestPointOnSurface. MarchedDeviation must reproduce it.

const (
	oracleFatRadius = 3.0
	oracleRodRadius = 1.5
)

// crossingCylinderPoint is the EXACT point at rod angle phi on the +x branch of the intersection of the
// z-axis cylinder (radius fatR) with the x-axis cylinder (radius rodR): y²+z²=rodR² fixes y and z, and
// x²+y²=fatR² then fixes x. Valid for rodR < fatR, where the branch is a single closed loop.
func crossingCylinderPoint(fatR, rodR, phi float64) math.Point3 {
	y := rodR * stdmath.Cos(phi)
	z := rodR * stdmath.Sin(phi)
	return math.P3(math.Scalar(stdmath.Sqrt(fatR*fatR-y*y)), math.Scalar(y), math.Scalar(z))
}

// coarseCrossingCylinderMarch samples the exact curve at `chords` equal rod-angle steps and closes the
// loop — a deliberately COARSE march whose chord bow is large enough to read without noise.
func coarseCrossingCylinderMarch(fatR, rodR float64, chords int) []math.Point3 {
	pts := make([]math.Point3, 0, chords+1)
	for i := 0; i <= chords; i++ {
		pts = append(pts, crossingCylinderPoint(fatR, rodR, 2*stdmath.Pi*float64(i)/float64(chords)))
	}
	return pts
}

// radialDistanceToAxisCylinder is the closed-form distance from p to a cylinder of radius r about a
// coordinate axis: |√(the two off-axis coordinates squared) − r|.
func radialDistanceToAxisCylinder(a, b math.Scalar, r float64) float64 {
	return stdmath.Abs(stdmath.Hypot(float64(a), float64(b)) - r)
}

// oracleChordBow is MarchedDeviation computed the closed-form way for the crossing-cylinder pair.
func oracleChordBow(fatR, rodR float64, pts []math.Point3) float64 {
	worst := 0.0
	for i := 0; i+1 < len(pts); i++ {
		m := pts[i].Midpoint(pts[i+1])
		worst = stdmath.Max(worst, stdmath.Max(
			radialDistanceToAxisCylinder(m.X, m.Y, fatR), // z-axis cylinder
			radialDistanceToAxisCylinder(m.Y, m.Z, rodR)) /* x-axis cylinder */)
	}
	return worst
}

// TestMarchedDeviationMatchesClosedFormChordBow: the achieved deviation the marcher reports must BE the
// chord bow, not an approximation of it — so on a march whose bow is known in closed form the two agree
// to round-off. It also pins the magnitude: the rod-circle sagitta rodR(1−cos(Δφ/2)) is one of the two
// distances taken in the max, so the reported deviation can never fall below it (#3489).
func TestMarchedDeviationMatchesClosedFormChordBow(t *testing.T) {
	fat, err := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), oracleFatRadius)
	if err != nil {
		t.Fatalf("fat cylinder: %v", err)
	}
	rod, err := NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), oracleRodRadius)
	if err != nil {
		t.Fatalf("rod cylinder: %v", err)
	}
	for _, chords := range []int{16, 32, 64} {
		pts := coarseCrossingCylinderMarch(oracleFatRadius, oracleRodRadius, chords)
		got := MarchedDeviation(fat, rod, pts)
		want := oracleChordBow(oracleFatRadius, oracleRodRadius, pts)
		if stdmath.Abs(got-want) > 1e-12*want { // tol:relative — agreement with the closed form, to round-off
			t.Errorf("%d chords: MarchedDeviation = %.12g, closed-form chord bow = %.12g", chords, got, want)
		}
		sagitta := oracleRodRadius * (1 - stdmath.Cos(stdmath.Pi/float64(chords)))
		if got < sagitta {
			t.Errorf("%d chords: deviation %.6g is below the rod-circle sagitta %.6g it contains", chords, got, sagitta)
		}
	}
}

// TestMarchedDeviationQuadratic: a chord bow is second order in the step, so halving the step must
// quarter the reported deviation. This is what makes the number a usable tolerance rather than a
// magic constant — it tracks the march density it came from.
func TestMarchedDeviationQuadratic(t *testing.T) {
	fat, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), oracleFatRadius)
	rod, _ := NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), oracleRodRadius)
	coarse := MarchedDeviation(fat, rod, coarseCrossingCylinderMarch(oracleFatRadius, oracleRodRadius, 32))
	fine := MarchedDeviation(fat, rod, coarseCrossingCylinderMarch(oracleFatRadius, oracleRodRadius, 64))
	ratio := coarse / fine
	if ratio < 3.8 || ratio > 4.2 { // tol:relative — second-order convergence, 4× per halved step
		t.Errorf("coarse/fine deviation ratio = %.3f (coarse %.6g, fine %.6g), want ≈4 (second order)", ratio, coarse, fine)
	}
}

// TestMarchedDeviationOfDegenerateInputIsZero: fewer than two points is no chord, so there is no bow to
// report. It must be 0 rather than a panic or a garbage magnitude.
func TestMarchedDeviationOfDegenerateInputIsZero(t *testing.T) {
	fat, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), oracleFatRadius)
	rod, _ := NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), oracleRodRadius)
	for _, pts := range [][]math.Point3{nil, {math.P3(3, 0, 0)}} {
		if got := MarchedDeviation(fat, rod, pts); got != 0 {
			t.Errorf("MarchedDeviation of %d point(s) = %g, want 0", len(pts), got)
		}
	}
}

// TestSurfaceIntersectMarchedPairReportsAchievedTolerance: a torus crossed by a cylinder has NO closed
// form in either bucket (a torus is quartic, so it is neither a straight-ruled parametrisation nor an
// implicit quadric), so SurfaceIntersect marches it — and every curve it returns must carry the achieved
// deviation of that march, not a silent claim of exactness. The magnitude is checked against the sagitta
// of the marched loops' own chord spacing, so the test measures the pipeline rather than freezing a
// constant (#3489).
func TestSurfaceIntersectMarchedPairReportsAchievedTolerance(t *testing.T) {
	tor, err := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 4, 1)
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	drill, _ := NewCylinder(math.P3(4, 0, 0), math.V3(0, 0, 1), 0.5)
	box := math.NewBox(math.P3(-6, -6, -6), math.P3(6, 6, 6))
	curves, handled := SurfaceIntersect(tor, drill, box, ResolutionForBox(box))
	if !handled || len(curves) == 0 {
		t.Fatalf("torus ∩ cylinder: handled=%v, %d curves; want a marched result", handled, len(curves))
	}
	for i, c := range curves {
		pl, ok := c.(Polyline)
		if !ok {
			t.Fatalf("curve %d is %T, want a marched Polyline", i, c)
		}
		dev := CurveDeviation(c)
		if dev <= 0 {
			t.Fatalf("curve %d (%d pts) reports deviation %g; a marched chord approximation is never exact", i, len(pl.Vertices), dev)
		}
		lo, hi := sagittaBand(0.5, len(pl.Vertices))
		if dev < lo || dev > hi {
			t.Errorf("curve %d (%d pts) deviation %.6g outside the chord-bow band [%.6g, %.6g]", i, len(pl.Vertices), dev, lo, hi)
		}
	}
}

// TestSurfaceIntersectRuledPairIsExact: two crossing cylinders of unequal radii are the ruled∩quadric
// closed form (#3489) — the pair that used to march. Every curve must be an exact section arc reporting a
// ZERO achieved tolerance, and must lie on BOTH cylinders to round-off, since an edge built from it claims
// exactness to the mass-properties integrator.
func TestSurfaceIntersectRuledPairIsExact(t *testing.T) {
	fat, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), oracleFatRadius)
	rod, _ := NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), oracleRodRadius)
	box := math.NewBox(math.P3(-6, -6, -6), math.P3(6, 6, 6))
	curves, handled := SurfaceIntersect(fat, rod, box, ResolutionForBox(box))
	if !handled || len(curves) != 2 {
		t.Fatalf("crossing cylinders: handled=%v, %d curves; want the 2 exact section loops", handled, len(curves))
	}
	for i, c := range curves {
		if _, ok := c.(RuledQuadricArc); !ok {
			t.Fatalf("curve %d is %T, want the exact geom.RuledQuadricArc", i, c)
		}
		if dev := CurveDeviation(c); dev != 0 {
			t.Errorf("curve %d reports deviation %g, want exactly 0 (the closed form is exact)", i, dev)
		}
		for k := range 129 {
			p := c.PointAt(float64(k) / 128)
			off := stdmath.Max(
				radialDistanceToAxisCylinder(p.X, p.Y, oracleFatRadius),
				radialDistanceToAxisCylinder(p.Y, p.Z, oracleRodRadius))
			if off > 1e-12 { // tol:absolute — round-off of the ruling quadratic at part scale
				t.Fatalf("curve %d at t=%g sits %g off the cylinder pair; the section must be exact", i, float64(k)/128, off)
			}
		}
	}
}

// sagittaBand brackets the chord bow expected of a closed loop of n vertices on a curve of radius r:
// r(1−cos(π/n)) is the bow of a uniform march, and the tracer's adaptive spacing puts the real number
// within an order of magnitude either side.
func sagittaBand(r float64, n int) (lo, hi float64) {
	nominal := r * (1 - stdmath.Cos(stdmath.Pi/float64(n)))
	return nominal / 10, nominal * 10 // tol:relative — an order-of-magnitude band around the uniform-march bow
}

// TestSurfaceIntersectAnalyticPairReportsZeroTolerance: the closed form is EXACT, so a pair it solves
// must report a zero achieved tolerance. This is the invariant that keeps the marched number meaningful —
// a non-zero reading has to mean "this boundary is an approximation".
func TestSurfaceIntersectAnalyticPairReportsZeroTolerance(t *testing.T) {
	sphere, err := NewSphere(math.P3(0, 0, 0), 5)
	if err != nil {
		t.Fatalf("sphere: %v", err)
	}
	plane, err := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	box := math.NewBox(math.P3(-6, -6, -6), math.P3(6, 6, 6))
	curves, handled := SurfaceIntersect(sphere, plane, box, ResolutionForBox(box))
	if !handled || len(curves) != 1 {
		t.Fatalf("sphere ∩ plane: handled=%v, %d curves; want the exact equator circle", handled, len(curves))
	}
	if dev := CurveDeviation(curves[0]); dev != 0 {
		t.Errorf("the closed-form %T reports deviation %g, want exactly 0 (it is exact)", curves[0], dev)
	}
}

// TestCurveDeviationReadsEveryCarrier: the achieved tolerance must survive the two ways a marched
// polyline travels — by value and by POINTER (the curved boolean carries imprint loops by identity so
// its run-merge can compare them with `==`) — and through a reversal wrapper. An analytic curve reports 0.
func TestCurveDeviationReadsEveryCarrier(t *testing.T) {
	pts := coarseCrossingCylinderMarch(oracleFatRadius, oracleRodRadius, 16)
	pl, err := NewMarchedPolyline(pts, 0.25)
	if err != nil {
		t.Fatalf("NewMarchedPolyline: %v", err)
	}
	circle, err := NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 5)
	if err != nil {
		t.Fatalf("circle: %v", err)
	}
	cases := []struct {
		name  string
		curve Curve3
		want  float64
	}{
		{"by value", pl, 0.25},
		{"by pointer", &pl, 0.25},
		{"reversed", ReverseCurve3(pl), 0.25},
		{"plain polyline", mustPolyline(t, pts), 0},
		{"analytic circle", circle, 0},
	}
	for _, c := range cases {
		if got := CurveDeviation(c.curve); got != c.want {
			t.Errorf("%s: CurveDeviation(%T) = %g, want %g", c.name, c.curve, got, c.want)
		}
	}
}

// mustPolyline builds a plain (exact) polyline or fails the test.
func mustPolyline(t *testing.T, pts []math.Point3) Polyline {
	t.Helper()
	pl, err := NewPolyline(pts)
	if err != nil {
		t.Fatalf("NewPolyline: %v", err)
	}
	return pl
}
