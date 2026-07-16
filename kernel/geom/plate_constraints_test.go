// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/math"
)

// unitSquareDomain is the average plane Ω used by the synthetic fixtures: the z=0 plane
// with the unit square's centroid as origin and axis-aligned in-plane frame, so a rail
// point's (u,v) is trivially hand-computable ((p−(0.5,0.5,0))·(U,V)).
func unitSquareDomain() PlateDomain {
	return PlateDomain{
		Origin: math.P3(0.5, 0.5, 0),
		U:      math.V3(1, 0, 0),
		V:      math.V3(0, 1, 0),
		N:      math.V3(0, 0, 1),
	}
}

// fakeSegment is a straight Curve3 from start to end over t∈[0,1] (named fake, not an
// inline stub — the rail curves of the synthetic corner).
type fakeSegment struct{ start, end math.Point3 }

func (s fakeSegment) PointAt(t float64) math.Point3 {
	return s.start.TranslateBy(s.start.VectorTo(s.end).Scale(t))
}
func (s fakeSegment) TangentAt(float64) math.Vector3 { return s.start.VectorTo(s.end) }
func (s fakeSegment) Domain() (float64, float64)     { return 0, 1 }

// fakePlaneSurface is an infinite plane with a fixed unit normal — the known-in-closed-form
// Adjacent surface a G1 rail is tangent to. NormalAt is constant (a plane), which is all the
// G1 recipe reads; du/dv only satisfy the Surface interface.
type fakePlaneSurface struct {
	point  math.Point3
	normal math.Vector3
	du, dv math.Vector3
}

func (p fakePlaneSurface) PointAt(u, v float64) math.Point3 {
	return p.point.TranslateBy(p.du.Scale(u).Add(p.dv.Scale(v)))
}
func (p fakePlaneSurface) DerivativesAt(float64, float64) (math.Vector3, math.Vector3) {
	return p.du, p.dv
}
func (p fakePlaneSurface) NormalAt(float64, float64) math.Vector3 { return p.normal }
func (p fakePlaneSurface) UDomain() (float64, float64)            { return stdmath.Inf(-1), stdmath.Inf(1) }
func (p fakePlaneSurface) VDomain() (float64, float64)            { return stdmath.Inf(-1), stdmath.Inf(1) }
func (p fakePlaneSurface) ParamAt(q math.Point3) (float64, float64) {
	r := p.point.VectorTo(q)
	return r.Dot(p.du), r.Dot(p.dv)
}

// fakeDegenerateSurface is an Adjacent surface with an ill-defined (zero) normal everywhere —
// the failure case a G1 rail must reject rather than emit a silently-broken tangency row.
type fakeDegenerateSurface struct{}

func (fakeDegenerateSurface) PointAt(float64, float64) math.Point3 { return math.P3(0, 0, 0) }
func (fakeDegenerateSurface) DerivativesAt(float64, float64) (math.Vector3, math.Vector3) {
	return math.Vector3{}, math.Vector3{}
}
func (fakeDegenerateSurface) NormalAt(float64, float64) math.Vector3 { return math.Vector3{} }
func (fakeDegenerateSurface) UDomain() (float64, float64)            { return 0, 1 }
func (fakeDegenerateSurface) VDomain() (float64, float64)            { return 0, 1 }
func (fakeDegenerateSurface) ParamAt(math.Point3) (float64, float64) { return 0, 0 }

// invSqrt2 is 1/√2, the tilt factor of the two G1 Adjacent planes.
var invSqrt2 = 1 / stdmath.Sqrt2

// squareSides builds the 4 rails of the unit square. Bottom (y=0) and top (y=1) are G1
// with tilted Adjacent planes whose feet-normals are known in closed form; right (x=1) and
// left (x=0) are G0-only. Traversed CCW so the interior is a consistent side.
func squareSides() [4]PlateSide {
	bottomN := math.V3(0, invSqrt2, invSqrt2) // plane through the y=0 rail, tilted 45°
	topN := math.V3(0, invSqrt2, -invSqrt2)   // plane through the y=1 rail, tilted the other way
	return [4]PlateSide{
		{Curve: fakeSegment{math.P3(0, 0, 0), math.P3(1, 0, 0)}, Order: 1,
			Adjacent: fakePlaneSurface{point: math.P3(0, 0, 0), normal: bottomN, du: math.V3(1, 0, 0), dv: math.V3(0, invSqrt2, -invSqrt2)}},
		{Curve: fakeSegment{math.P3(1, 0, 0), math.P3(1, 1, 0)}, Order: 0},
		{Curve: fakeSegment{math.P3(1, 1, 0), math.P3(0, 1, 0)}, Order: 1,
			Adjacent: fakePlaneSurface{point: math.P3(0, 1, 0), normal: topN, du: math.V3(1, 0, 0), dv: math.V3(0, invSqrt2, invSqrt2)}},
		{Curve: fakeSegment{math.P3(0, 1, 0), math.P3(0, 0, 0)}, Order: 0},
	}
}

func TestDiscretizeSidesCounts(t *testing.T) {
	cs, vals, err := DiscretizeSides(squareSides(), unitSquareDomain(), 5)
	if err != nil {
		t.Fatalf("DiscretizeSides: unexpected error: %v", err)
	}
	// 4 sides × 5 G0 rows = 20; 2 G1 sides × 5 samples × 2 deriv rows = 20; total 40.
	if len(cs) != 40 {
		t.Fatalf("constraint count = %d, want 40", len(cs))
	}
	for c := 0; c < 3; c++ {
		if len(vals[c]) != len(cs) {
			t.Fatalf("value field %d length = %d, want %d (one per constraint)", c, len(vals[c]), len(cs))
		}
	}
	g0, g1 := countOrders(cs)
	if g0 != 20 || g1 != 20 {
		t.Fatalf("G0 rows = %d (want 20), G1 rows = %d (want 20)", g0, g1)
	}
}

// countOrders splits the constraint set into G0 ({0,0}) and first-derivative row counts.
func countOrders(cs []PlateConstraint) (g0, g1 int) {
	for _, c := range cs {
		if c.Order == [2]int{0, 0} {
			g0++
			continue
		}
		g1++
	}
	return g0, g1
}

func TestDiscretizeSidesG0Values(t *testing.T) {
	cs, vals, err := DiscretizeSides(squareSides(), unitSquareDomain(), 5)
	if err != nil {
		t.Fatalf("DiscretizeSides: %v", err)
	}
	// The 3rd sample (k=2, t=0.5) of the bottom rail is world (0.5,0,0) → (u,v)=(0,−0.5).
	i := findRow(cs, 0, -0.5, [2]int{0, 0})
	if i < 0 {
		t.Fatalf("no G0 row at (u,v)=(0,-0.5); rows=%v", cs)
	}
	assertXYZ(t, vals, i, 0.5, 0, 0)
}

// findRow returns the index of the first constraint at (u,v) of the given order, or −1.
func findRow(cs []PlateConstraint, u, v float64, order [2]int) int {
	for i, c := range cs {
		if c.Order == order && stdmath.Abs(c.U-u) < 1e-12 && stdmath.Abs(c.V-v) < 1e-12 {
			return i
		}
	}
	return -1
}

func assertXYZ(t *testing.T, vals [3][]float64, i int, x, y, z float64) {
	t.Helper()
	if stdmath.Abs(vals[0][i]-x) > 1e-12 || stdmath.Abs(vals[1][i]-y) > 1e-12 || stdmath.Abs(vals[2][i]-z) > 1e-12 {
		t.Fatalf("row %d value = (%g,%g,%g), want (%g,%g,%g)", i, vals[0][i], vals[1][i], vals[2][i], x, y, z)
	}
}

// TestDiscretizeSidesG1Watertight is the watertight-critical assertion: at every G1 foot the
// prescribed axis partials S_u, S_v must span the Adjacent surface's tangent plane, i.e.
// S_u×S_v ∥ n̂. A wrong transverse DIRECTION passes an area gate but breaks the weld — this
// checks the property directly (kit §3 runtime witness).
func TestDiscretizeSidesG1Watertight(t *testing.T) {
	sides := squareSides()
	d := unitSquareDomain()
	cs, vals, err := DiscretizeSides(sides, d, 5)
	if err != nil {
		t.Fatalf("DiscretizeSides: %v", err)
	}
	res := ResolutionForSize(stdmath.Sqrt2)
	maxResidual := 0.0
	for _, side := range sides {
		if side.Order != 1 {
			continue
		}
		maxResidual = stdmath.Max(maxResidual, checkSideG1(t, side, d, cs, vals, res))
	}
	t.Logf("worst S_u×S_v ∥ n̂ residual across all G1 feet: %.3e (weld %.3e)", maxResidual, res.Weld())
}

// checkSideG1 walks one G1 side's feet, reconstructs (S_u,S_v) from the emitted rows, and
// asserts their cross product is parallel to the Adjacent normal at that foot. Returns the
// worst parallelism residual |unit(S_u×S_v)×n̂| seen.
func checkSideG1(t *testing.T, side PlateSide, d PlateDomain, cs []PlateConstraint, vals [3][]float64, res Resolution) float64 {
	t.Helper()
	lo, hi := side.Curve.Domain()
	worst := 0.0
	for k := 0; k < 5; k++ {
		foot := side.Curve.PointAt(lo + (hi-lo)*float64(k)/4)
		u, v := d.Project(foot)
		su := rowVector(cs, vals, u, v, [2]int{1, 0})
		sv := rowVector(cs, vals, u, v, [2]int{0, 1})
		fu, fv := side.Adjacent.ParamAt(foot)
		n := side.Adjacent.NormalAt(fu, fv)
		cross := unitOrZero(su.Cross(sv))
		resid := float64(cross.Cross(n).Length())
		if resid > res.Weld() {
			t.Fatalf("G1 foot %v: unit(S_u×S_v)=%v not ∥ n̂=%v (residual %.3e > weld %.3e)", foot, cross, n, resid, res.Weld())
		}
		worst = stdmath.Max(worst, resid)
	}
	return worst
}

// rowVector rebuilds the (x,y,z) target vector of the row at (u,v) of the given order.
func rowVector(cs []PlateConstraint, vals [3][]float64, u, v float64, order [2]int) math.Vector3 {
	i := findRow(cs, u, v, order)
	if i < 0 {
		return math.Vector3{}
	}
	return math.V3(vals[0][i], vals[1][i], vals[2][i])
}

// TestDiscretizeSidesG1ClosedForm pins the exact (S_u,S_v) of the bottom rail's midpoint foot
// against the hand-computed values in the P3 report, so a sign/rotation regression is caught
// numerically, not just by the parallelism witness.
func TestDiscretizeSidesG1ClosedForm(t *testing.T) {
	cs, vals, err := DiscretizeSides(squareSides(), unitSquareDomain(), 5)
	if err != nil {
		t.Fatalf("DiscretizeSides: %v", err)
	}
	su := rowVector(cs, vals, 0, -0.5, [2]int{1, 0})
	sv := rowVector(cs, vals, 0, -0.5, [2]int{0, 1})
	assertVec(t, "S_u", su, math.V3(1, 0, 0))
	assertVec(t, "S_v", sv, math.V3(0, invSqrt2, -invSqrt2))
}

func assertVec(t *testing.T, name string, got, want math.Vector3) {
	t.Helper()
	if !got.IsEqualTo(want, 1e-12) {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}

// TestDiscretizeSidesDegenerateStrip rejects a side that projects to a near-degenerate strip
// (a rail perpendicular to Ω collapses to a point in the domain), carrying the measured
// arc-length in the error.
func TestDiscretizeSidesDegenerateStrip(t *testing.T) {
	sides := squareSides()
	sides[1] = PlateSide{Curve: fakeSegment{math.P3(1, 0, 0), math.P3(1, 0, 1)}, Order: 0} // rises along +Z ⊥ Ω
	_, _, err := DiscretizeSides(sides, unitSquareDomain(), 5)
	if err == nil {
		t.Fatalf("expected degenerate-strip error, got nil")
	}
	if !strings.Contains(err.Error(), "degenerate strip") || !strings.Contains(err.Error(), "arc-length") {
		t.Fatalf("error should name the degenerate strip and its arc-length, got: %v", err)
	}
}

// TestDiscretizeSidesIllDefinedNormal rejects a G1 rail whose Adjacent surface has no
// well-defined normal at the foot (a zero normal → no tangent plane → G1 undefined).
func TestDiscretizeSidesIllDefinedNormal(t *testing.T) {
	sides := squareSides()
	sides[0].Adjacent = fakeDegenerateSurface{}
	_, _, err := DiscretizeSides(sides, unitSquareDomain(), 5)
	if err == nil {
		t.Fatalf("expected ill-defined-normal error, got nil")
	}
	if !strings.Contains(err.Error(), "normal degenerate") {
		t.Fatalf("error should name the degenerate normal, got: %v", err)
	}
}
