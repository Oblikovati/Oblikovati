// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// Edge-anchored rolling-ball canal stations against NUMERIC hosts (a B-spline wall, or two).
// For each anchor on the picked edge — the guide curve — the ball centre is solved in the
// SECTION PLANE through the anchor normal to the guide tangent, at distance r from both
// hosts, with the two tangency feet found by warm-started local point inversion. This is
// OCCT's constant-radius blend semantics ported faithfully: the section condition (nplan =
// normalized guide tangent at ptgui) is BlendFunc_ConstRad.cxx (OCCT
// src/ModelingAlgorithms/TKFillet/BlendFunc/), and the station-to-station warm continuation
// plays Blend_Walking's role (BRepBlend_Walking.hxx, Blend generic in TKGeomAlgo/Blend).
//
// The 2×2 Newton uses the envelope-theorem Jacobian: ∇dist-to-surface is exactly the unit
// foot→point direction (the foot's own motion is tangent to the host and drops out at first
// order), so each Jacobian row is a unit foot normal projected into the section plane. Valid
// wherever the foot is unique — i.e. away from the host's evolute, which the ball-clearance
// probe (canalBallClearance) enforces per station.

// CanalMarchHost is one roll host of the edge-anchored canal march. Side is the frozen sign
// of (C−foot)·N_host at the tangency foot: 0 lets the march derive it from station 0 and
// FREEZE it (a later flip is a fold/branch jump and aborts — the branch invariant). PeriodU
// > 0 marks a host closed in u (a swept pipe wall): foot u-parameters live on the universal
// cover (lifted, never clamped), so a march across the seam is invisible to the inversion.
type CanalMarchHost struct {
	Surf    Surface
	Side    float64
	PeriodU float64
}

// CanalFoot is one tangency foot: its (lifted) surface parameters and position.
type CanalFoot struct {
	U, V float64
	P    math.Point3
}

// CanalEdgeStation is one solved cross-section station of the canal.
type CanalEdgeStation struct {
	Anchor math.Point3
	Center math.Point3
	FootA  CanalFoot
	FootB  CanalFoot
}

// CanalEdgeAnchor is one section anchor on the guide: the guide point and its unit tangent
// (OCCT's ptgui / nplan pair — the section plane is {X : (X−P)·T = 0}).
type CanalEdgeAnchor struct {
	P math.Point3
	T math.Vector3
}

// canalMarchMaxNewton caps the damped per-station Newton. Quadratic convergence from the
// warm seed needs ~4; the cap only bounds a diverging degenerate station before it aborts.
const canalMarchMaxNewton = 30

// canalMarchHostState carries one host's frozen side and the warm inversion seed across
// stations — the continuation state Blend_Walking keeps in its previous solution point.
type canalMarchHostState struct {
	host CanalMarchHost
	side float64
	prev CanalFoot
	warm bool
}

// MarchCanalEdgeStations solves one canal station per anchor, seeded at c0 (the caller's
// topology-decided two-plane closed form for station 0) and continued warm station to
// station. weld is the model-relative station tolerance: every accepted station has both
// feet at |dist−r| ≤ weld (the same bound LoftCanalStations re-asserts). It errors — never
// fabricates a station — on Newton divergence, a side-sign flip, near-tangent hosts, or a
// ball-clearance (evolute) violation, naming the offending station and values.
func MarchCanalEdgeStations(hostA, hostB CanalMarchHost, r float64, anchors []CanalEdgeAnchor, c0 math.Point3, weld float64) ([]CanalEdgeStation, error) {
	return MarchCanalEdgeStationsSeeded(hostA, hostB, r, anchors, 0, c0, weld)
}

// MarchCanalEdgeStationsSeeded marches OUTWARD from anchors[seedIdx]: the seed station is
// solved from c0 first, then the run continues forward (seedIdx+1…) and backward
// (seedIdx−1…0) with warm continuation from the seed. An OPEN edge's march must be seeded
// ON the edge (never at a prolong tip, where the hosts' polynomial extension makes the
// global station-0 inversion and the two-plane seed unreliable) — the prolong spans are
// then reached as warm continuations of sound on-edge stations.
func MarchCanalEdgeStationsSeeded(hostA, hostB CanalMarchHost, r float64, anchors []CanalEdgeAnchor, seedIdx int, c0 math.Point3, weld float64) ([]CanalEdgeStation, error) {
	if len(anchors) < 2 || seedIdx < 0 || seedIdx >= len(anchors) {
		return nil, fmt.Errorf("MarchCanalEdgeStationsSeeded: %d anchors with seed %d, need >= 2 and an in-range seed", len(anchors), seedIdx)
	}
	if r <= 0 || weld <= 0 {
		return nil, fmt.Errorf("MarchCanalEdgeStationsSeeded: need r > 0 and weld > 0, got r=%g weld=%g", r, weld)
	}
	a := &canalMarchHostState{host: hostA, side: hostA.Side}
	b := &canalMarchHostState{host: hostB, side: hostB.Side}
	out := make([]CanalEdgeStation, len(anchors))
	st, err := solveCanalStation(a, b, r, anchors[seedIdx], c0, weld)
	if err != nil {
		return nil, fmt.Errorf("canal seed station %d (anchor %v): %w", seedIdx, anchors[seedIdx].P, err)
	}
	out[seedIdx] = st
	if err := marchCanalDirection(a, b, r, anchors, out, seedIdx, +1, weld); err != nil {
		return nil, err
	}
	ab, bb := restateAtSeed(a, hostA, st.FootA), restateAtSeed(b, hostB, st.FootB)
	if err := marchCanalDirection(ab, bb, r, anchors, out, seedIdx, -1, weld); err != nil {
		return nil, err
	}
	return out, nil
}

// restateAtSeed clones a host's continuation state back to the seed station (the forward
// run advanced the live state to the far end), keeping the frozen side.
func restateAtSeed(s *canalMarchHostState, host CanalMarchHost, seedFoot CanalFoot) *canalMarchHostState {
	return &canalMarchHostState{host: host, side: s.side, prev: seedFoot, warm: true}
}

// marchCanalDirection continues the march from the solved seed station in one direction
// (step ±1), each station's centre seeded by translating the previous one anchor-to-anchor
// (Blend_Walking's previous-point predictor).
func marchCanalDirection(a, b *canalMarchHostState, r float64, anchors []CanalEdgeAnchor, out []CanalEdgeStation, seedIdx, step int, weld float64) error {
	for k := seedIdx + step; k >= 0 && k < len(anchors); k += step {
		seed := out[k-step].Center.TranslateBy(anchors[k-step].P.VectorTo(anchors[k].P))
		st, err := solveCanalStation(a, b, r, anchors[k], seed, weld)
		if err != nil {
			return fmt.Errorf("canal station %d (anchor %v): %w", k, anchors[k].P, err)
		}
		out[k] = st
	}
	return nil
}

// solveCanalStation runs the damped 2×2 Newton for one station: C in the section plane with
// dist(C, A) = dist(C, B) = r, feet by warm local inversion. On convergence it asserts the
// frozen side signs and the ball clearance on both hosts.
func solveCanalStation(a, b *canalMarchHostState, r float64, an CanalEdgeAnchor, seed math.Point3, weld float64) (CanalEdgeStation, error) {
	p, q, err := sectionPlaneBasis(an.T)
	if err != nil {
		return CanalEdgeStation{}, err
	}
	c := projectIntoSectionPlane(seed, an, p, q)
	c, fa, fb, err := newtonCanalCentre(a, b, r, p, q, c, weld)
	if err != nil {
		return CanalEdgeStation{}, err
	}
	if err := acceptCanalStation(a, b, c, fa, fb, r, weld); err != nil {
		return CanalEdgeStation{}, err
	}
	a.prev, a.warm = fa, true
	b.prev, b.warm = fb, true
	return CanalEdgeStation{Anchor: an.P, Center: c, FootA: fa, FootB: fb}, nil
}

// sectionPlaneBasis returns an orthonormal in-plane basis (p, q) of the section plane with
// normal t (the guide tangent). Errors on a degenerate tangent.
func sectionPlaneBasis(t math.Vector3) (p, q math.Vector3, err error) {
	tn, uerr := math.UnitVector3FromVector(t)
	if uerr != nil {
		return math.Vector3{}, math.Vector3{}, fmt.Errorf("degenerate guide tangent %v", t)
	}
	pn, uerr := math.UnitVector3FromVector(perpendicularSeed(tn.AsVector()))
	if uerr != nil {
		return math.Vector3{}, math.Vector3{}, fmt.Errorf("no in-plane basis for tangent %v", t)
	}
	return pn.AsVector(), tn.AsVector().Cross(pn.AsVector()), nil
}

// perpendicularSeed is any vector not parallel to t, crossed into a perpendicular.
func perpendicularSeed(t math.Vector3) math.Vector3 {
	seed := math.V3(1, 0, 0)
	if stdmath.Abs(float64(t.X)) > 0.9 {
		seed = math.V3(0, 1, 0)
	}
	return t.Cross(seed)
}

// projectIntoSectionPlane drops the seed centre into the section plane {X: (X−P)·T = 0}.
func projectIntoSectionPlane(seed math.Point3, an CanalEdgeAnchor, p, q math.Vector3) math.Point3 {
	d := an.P.VectorTo(seed)
	return an.P.TranslateBy(p.Scale(d.Dot(p))).TranslateBy(q.Scale(d.Dot(q)))
}

// newtonCanalCentre is the damped 2×2 Newton on the in-plane centre: residuals are the two
// |C−foot|−r distances, Jacobian rows the unit foot normals projected into (p, q) — the
// envelope-theorem gradient of the distance-to-surface function.
func newtonCanalCentre(a, b *canalMarchHostState, r float64, p, q math.Vector3, c math.Point3, weld float64) (math.Point3, CanalFoot, CanalFoot, error) {
	tol := 0.25 * weld
	for range canalMarchMaxNewton {
		fa, fb, g1, g2, err := canalStationResiduals(a, b, c, r)
		if err != nil {
			return c, fa, fb, err
		}
		if stdmath.Abs(g1) <= tol && stdmath.Abs(g2) <= tol {
			return c, fa, fb, nil
		}
		dx, dy, err := solveCanalNewtonStep(c, fa, fb, p, q, g1, g2, r, weld)
		if err != nil {
			return c, fa, fb, err
		}
		c = dampCanalStep(a, b, c, r, p, q, dx, dy, g1, g2)
	}
	return c, CanalFoot{}, CanalFoot{}, fmt.Errorf("station Newton did not converge in %d iterations (r=%g)", canalMarchMaxNewton, r)
}

// canalStationResiduals inverts both feet at the current centre and returns the two
// distance residuals |C−foot|−r.
func canalStationResiduals(a, b *canalMarchHostState, c math.Point3, r float64) (fa, fb CanalFoot, g1, g2 float64, err error) {
	fa, err = a.invertWarm(c)
	if err != nil {
		return fa, fb, 0, 0, err
	}
	fb, err = b.invertWarm(c)
	if err != nil {
		return fa, fb, 0, 0, err
	}
	return fa, fb, float64(c.DistanceTo(fa.P)) - r, float64(c.DistanceTo(fb.P)) - r, nil
}

// solveCanalNewtonStep solves J·[dx dy]ᵀ = −g with J rows the unit foot normals projected
// into the section basis. A near-singular J means the two foot normals are (anti)parallel in
// the section plane — the tangent-host degeneracy — and errors with the offending sine.
func solveCanalNewtonStep(c math.Point3, fa, fb CanalFoot, p, q math.Vector3, g1, g2, r, weld float64) (dx, dy float64, err error) {
	na := c.VectorTo(fa.P).Scale(-1)
	nb := c.VectorTo(fb.P).Scale(-1)
	a11, a12 := unitDot(na, p), unitDot(na, q)
	a21, a22 := unitDot(nb, p), unitDot(nb, q)
	det := a11*a22 - a12*a21
	if stdmath.Abs(det) < canalTangentSineFloor(r, weld) {
		return 0, 0, fmt.Errorf("near-tangent hosts: section-plane normal wedge %.3e under floor %.3e", det, canalTangentSineFloor(r, weld))
	}
	return (-g1*a22 + g2*a12) / det, (-g2*a11 + g1*a21) / det, nil
}

// canalTangentSineFloor is the conditioning floor on the section-plane normal wedge (the
// Newton determinant): the model-relative angular band weld/r (the curvedTangencyBand
// pattern, ADR-0042) floored at 1e-6 so a huge radius on a small model cannot demand a
// wedge below numeric noise.
func canalTangentSineFloor(r, weld float64) float64 {
	return stdmath.Max(1e-6, weld/r)
}

// unitDot is (unit(v))·w for a possibly non-unit v; 0 for a degenerate v.
func unitDot(v, w math.Vector3) float64 {
	l := float64(v.Length())
	if l == 0 {
		return 0
	}
	return float64(v.Dot(w)) / l
}

// dampCanalStep applies the Newton step with halving line-search on the squared residual
// norm, keeping the centre in the section plane by stepping only along (p, q).
func dampCanalStep(a, b *canalMarchHostState, c math.Point3, r float64, p, q math.Vector3, dx, dy, g1, g2 float64) math.Point3 {
	best := g1*g1 + g2*g2
	alpha := 1.0
	for range 8 {
		cand := c.TranslateBy(p.Scale(math.Scalar(alpha * dx))).TranslateBy(q.Scale(math.Scalar(alpha * dy)))
		if canalResidualNorm(a, b, cand, r) < best {
			return cand
		}
		alpha *= 0.5
	}
	return c.TranslateBy(p.Scale(math.Scalar(dx))).TranslateBy(q.Scale(math.Scalar(dy)))
}

// canalResidualNorm is the squared distance-residual norm at a candidate centre, +Inf when
// an inversion fails (so the line search backs off instead of stepping into a failure).
func canalResidualNorm(a, b *canalMarchHostState, c math.Point3, r float64) float64 {
	fa, err := a.peekInvert(c)
	if err != nil {
		return stdmath.Inf(1)
	}
	fb, err := b.peekInvert(c)
	if err != nil {
		return stdmath.Inf(1)
	}
	g1 := float64(c.DistanceTo(fa.P)) - r
	g2 := float64(c.DistanceTo(fb.P)) - r
	return g1*g1 + g2*g2
}

// acceptCanalStation runs the per-station invariants: feet at radius (within weld), frozen
// side signs, and the ball-clearance (evolute) probe on both hosts.
func acceptCanalStation(a, b *canalMarchHostState, c math.Point3, fa, fb CanalFoot, r, weld float64) error {
	if d := stdmath.Abs(float64(c.DistanceTo(fa.P)) - r); d > weld {
		return fmt.Errorf("foot A %g off radius %g (weld %g)", d, r, weld)
	}
	if d := stdmath.Abs(float64(c.DistanceTo(fb.P)) - r); d > weld {
		return fmt.Errorf("foot B %g off radius %g (weld %g)", d, r, weld)
	}
	if err := a.assertSide(c, fa); err != nil {
		return fmt.Errorf("host A: %w", err)
	}
	if err := b.assertSide(c, fb); err != nil {
		return fmt.Errorf("host B: %w", err)
	}
	if err := canalBallClearance(a.host, c, fa, r); err != nil {
		return fmt.Errorf("host A: %w", err)
	}
	if err := canalBallClearance(b.host, c, fb, r); err != nil {
		return fmt.Errorf("host B: %w", err)
	}
	return nil
}

// assertSide freezes the host's tangency side at station 0 and asserts it thereafter: a
// flip means the march crossed a fold or jumped branch (the Blend_Walking branch invariant).
func (s *canalMarchHostState) assertSide(c math.Point3, f CanalFoot) error {
	n := s.host.normalAtLifted(f.U, f.V)
	side := stdmath.Copysign(1, float64(f.P.VectorTo(c).Dot(n)))
	if s.side == 0 {
		s.side = side
		return nil
	}
	if side != s.side {
		return fmt.Errorf("tangency side flipped (%+g → %+g): fold or branch jump", s.side, side)
	}
	return nil
}

// canalBallClearanceSpanFrac and canalBallClearancePenetration parameterize the evolute
// probe: host samples at ±0.3r/±0.6r around the foot must stay outside the ball up to a
// 1e-6·r penetration allowance. At those spans a curvature violation κ > 1/r penetrates
// quadratically (depth ≈ s²(κ−1/r)/2, i.e. ~1e-2·r at 5% violation) — far above the
// allowance — while a healthy tangency clears it by the same margin. The probe is anchored
// on REQUEST geometry (the host surface itself), not on the march's own output.
const (
	canalBallClearanceSpanFrac    = 0.3
	canalBallClearancePenetration = 1e-6
)

// canalBallClearance rejects a station whose ball locally penetrates its host beyond the
// tangency foot — the r-past-the-evolute failure a constant-radius fillet must refuse.
func canalBallClearance(h CanalMarchHost, c math.Point3, f CanalFoot, r float64) error {
	du, dv := h.derivsAtLifted(f.U, f.V)
	for _, span := range []float64{canalBallClearanceSpanFrac, 2 * canalBallClearanceSpanFrac} {
		for _, dir := range [4][2]float64{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			su, sv := paramSpanFor(du, dv, span*r, dir)
			p := h.pointAtLifted(f.U+su, f.V+sv)
			if d := float64(c.DistanceTo(p)); d < r*(1-canalBallClearancePenetration) {
				return fmt.Errorf("ball of radius %g penetrates host near foot (sample dist %g): radius past local curvature", r, d)
			}
		}
	}
	return nil
}

// paramSpanFor converts a surface-space probe span into a parameter-space step along one
// param direction, guarding a degenerate tangent (zero step probes the foot itself: inert).
func paramSpanFor(du, dv math.Vector3, span float64, dir [2]float64) (su, sv float64) {
	lu, lv := float64(du.Length()), float64(dv.Length())
	if dir[0] != 0 && lu > 0 {
		su = dir[0] * span / lu
	}
	if dir[1] != 0 && lv > 0 {
		sv = dir[1] * span / lv
	}
	return su, sv
}
