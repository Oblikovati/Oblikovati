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

// genericNormal is a fully generic unit Adjacent normal — three DISTINCT, NONZERO components
// ((12,3,-4)/13, a clean unit vector via the 3-4-12-13 Pythagorean quadruple). This is the P3
// review fixture fix: squareSides' tilted normals (0,±1/√2,1/√2) have X=0 AND |Y|=|Z|, two
// symmetries that leave the normal (and hence the parallelism check against it) invariant
// under an axis swap or a Y↔Z transposition — hiding exactly those mutations. genericNormal has
// no such symmetry: swapping or transposing any pair of its components produces a DIFFERENT
// vector, not a scalar multiple of the original.
var genericNormal = math.V3(12, 3, -4).Scale(1.0 / 13.0)

// genericRailTangent is the along-rail world tangent paired with genericNormal: the matching
// leg of the same 3-4-12-13 quadruple, so tangent·normal=0 (the G1 precondition — the rail
// lies in Adjacent's tangent plane). Its domain (x,y) projection (3,4) is NOT axis-aligned
// (unlike squareSides' bottom/top rails, whose tangents are exactly (±1,0,0) — giving one of
// {d̂ᵘ,d̂ᵛ} the vacuous value 0, which hides an S_u↔S_v swap; see the along-rail identity check
// in checkAlongRailIdentity).
var genericRailTangent = math.V3(3, 4, 12)

// genericPlaneDU, genericPlaneDV complete (genericPlaneDU, genericPlaneDV, genericNormal) into
// a right-handed orthonormal frame (genericPlaneDU = unit(genericRailTangent); genericPlaneDV =
// genericNormal × genericPlaneDU) — only needed so fakePlaneSurface.ParamAt has a well-defined
// (non-degenerate) projection basis; NormalAt is constant, so these don't affect the G1 math.
var (
	genericPlaneDU = genericRailTangent.Scale(1.0 / 13.0)
	genericPlaneDV = math.V3(4, -12, 3).Scale(1.0 / 13.0)
)

// genericCornerSides builds a synthetic 4-rail corner whose SINGLE G1 side carries the fully
// generic genericNormal/genericRailTangent pair above. Used by the hardened watertight and
// along-rail-identity tests (P3 review) so a wrong transverse direction, a collapsed tangent
// frame, an S_u↔S_v swap, or a Y↔Z value transposition all show up as a genuine, non-symmetric
// residual — none of the four mutations leaves this fixture invariant. The other 3 sides are
// plain G0 rails; DiscretizeSides treats each side independently, so they need not close a
// planar loop the way squareSides' do.
func genericCornerSides() [4]PlateSide {
	origin := math.P3(0, 0, 0)
	tip := origin.TranslateBy(genericRailTangent) // (3,4,12): distinct, nonzero X/Y/Z
	return [4]PlateSide{
		{Curve: fakeSegment{origin, tip}, Order: 1,
			Adjacent: fakePlaneSurface{point: origin, normal: genericNormal, du: genericPlaneDU, dv: genericPlaneDV}},
		{Curve: fakeSegment{tip, math.P3(6, 0, 0)}, Order: 0},
		{Curve: fakeSegment{math.P3(6, 0, 0), math.P3(0, 6, 0)}, Order: 0},
		{Curve: fakeSegment{math.P3(0, 6, 0), origin}, Order: 0},
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
	for c := range 3 {
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
	for k := range 5 {
		foot := side.Curve.PointAt(lo + (hi-lo)*float64(k)/4)
		u, v := d.Project(foot)
		su := rowVector(cs, vals, u, v, [2]int{1, 0})
		sv := rowVector(cs, vals, u, v, [2]int{0, 1})
		raw := su.Cross(sv)
		// Non-degeneracy guard (P3 review): a collapsed tangent frame (τ=0, or τ ∥ A from a
		// dropped n̂×Â cross) makes unitOrZero(raw)=0, and 0×n̂=0 ≤ weld passes the parallelism
		// check below VACUOUSLY. Must fail loudly here, before that check would short-circuit.
		if raw.Length() <= res.Weld() {
			t.Fatalf("G1 foot %v: |S_u×S_v|=%.3e <= weld %.3e; tangent frame is degenerate (collapsed or τ ∥ A)",
				foot, raw.Length(), res.Weld())
		}
		fu, fv := side.Adjacent.ParamAt(foot)
		n := side.Adjacent.NormalAt(fu, fv)
		cross := unitOrZero(raw)
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

// checkAlongRailIdentity verifies the kit §3 along-rail identity — moving along the domain
// direction d̂ reproduces the rail's own world tangent: d̂ᵘ·S_u + d̂ᵛ·S_v = A = tangent/ρ. This
// holds EXACTLY for the correct recipe regardless of how {S_u,S_v} decompose {A,τ} (the
// decomposition is a rotation of an orthonormal frame, and d̂ᵘ²+d̂ᵛ²=1 cancels the cross terms)
// — but an S_u↔S_v swap breaks it whenever the rail is NOT axis-aligned in the domain (d̂ᵘ,d̂ᵛ
// both nonzero), which genericRailTangent's (3,4) domain projection guarantees (squareSides'
// rails are exactly axis-aligned, d̂ᵛ=0 there, which would hide this exact swap).
func checkAlongRailIdentity(t *testing.T, side PlateSide, d PlateDomain, cs []PlateConstraint, vals [3][]float64, ts []float64, res Resolution) {
	t.Helper()
	for _, tp := range ts {
		foot := side.Curve.PointAt(tp)
		tangent := side.Curve.TangentAt(tp)
		du, dv := tangent.Dot(d.U), tangent.Dot(d.V)
		rho := stdmath.Hypot(du, dv)
		dhu, dhv := du/rho, dv/rho
		u, v := d.Project(foot)
		su := rowVector(cs, vals, u, v, [2]int{1, 0})
		sv := rowVector(cs, vals, u, v, [2]int{0, 1})
		reconstructed := su.Scale(dhu).Add(sv.Scale(dhv))
		wantA := tangent.Scale(1 / rho)
		if resid := reconstructed.Sub(wantA).Length(); resid > res.Weld() {
			t.Fatalf("G1 foot %v: d̂ᵘS_u+d̂ᵛS_v=%v, want rail tangent A=%v "+
				"(along-rail identity broken, residual %.3e > weld %.3e)", foot, reconstructed, wantA, resid, res.Weld())
		}
	}
}

// genericCornerRes is the model-relative Resolution for genericCornerSides — sized off its
// largest coordinate (genericRailTangent's hypotenuse, 13, the 3-4-12-13 quadruple).
func genericCornerRes() Resolution { return ResolutionForSize(13.0) }

// TestDiscretizeSidesG1WatertightGenericNormal is the P3-review-hardened counterpart of
// TestDiscretizeSidesG1Watertight: it runs the SAME parallelism witness (via checkSideG1, which
// now also carries the non-degeneracy guard) plus the along-rail identity over genericCornerSides
// — a fixture with NO axis/component symmetry — so each of the 4 reviewer mutations produces its
// own independent failure here, not just in the single hand-pinned TestDiscretizeSidesG1ClosedForm.
func TestDiscretizeSidesG1WatertightGenericNormal(t *testing.T) {
	sides := genericCornerSides()
	d := unitSquareDomain()
	cs, vals, err := DiscretizeSides(sides, d, 5)
	if err != nil {
		t.Fatalf("DiscretizeSides: %v", err)
	}
	res := genericCornerRes()
	g1 := sides[0]
	checkSideG1(t, g1, d, cs, vals, res)
	checkAlongRailIdentity(t, g1, d, cs, vals, curveParams(g1.Curve, 5), res)
}

// TestDiscretizeSidesG0ValuesGeneric is the P3-review-hardened counterpart of
// TestDiscretizeSidesG0Values: the probed foot is genericCornerSides' rail tip (3,4,12) —
// three DISTINCT, NONZERO coordinates (unlike the z=0-plane squareSides fixture, where every
// G0 value has Z=0) — so a Y↔Z (or any) coordinate transposition in the value columns changes
// the asserted numbers instead of leaving a degenerate 0 column unchanged.
func TestDiscretizeSidesG0ValuesGeneric(t *testing.T) {
	sides := genericCornerSides()
	d := unitSquareDomain()
	cs, vals, err := DiscretizeSides(sides, d, 5)
	if err != nil {
		t.Fatalf("DiscretizeSides: %v", err)
	}
	tip := sides[0].Curve.PointAt(1) // (3,4,12): the shared endpoint of side 0 and side 1
	u, v := d.Project(tip)
	i := findRow(cs, u, v, [2]int{0, 0})
	if i < 0 {
		t.Fatalf("no G0 row at tip %v (u,v)=(%g,%g); rows=%v", tip, u, v, cs)
	}
	assertXYZ(t, vals, i, float64(tip.X), float64(tip.Y), float64(tip.Z))
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
