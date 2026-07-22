// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// TorusCylinderArc is a bounded segment of the TORUS ∩ CYLINDER section, evaluated ON the torus chart so it
// is EXACTLY on the torus arm (shared-edge identity, tessellator honesty) and on the capping cylinder to
// rounding — the far-runout section-curve sibling of SpiricArc (torus∩plane). Unlike the plane section,
// torus∩cylinder has no single-arccos u(v) form: at fixed tube angle v the on-cylinder constraint
//
//	g(u) = |E|² − (E·â₂)² − R₂² = 0,   E = P(u,v) − O₂
//
// is a degree-4 azimuth equation, so the section branch is followed by FEET-BRACKETED Newton continuation
// (far-runout-port-math.md §4): NewTorusCylinderArc solves g=0 across v∈[V0,V1] seeded from the previous
// v-slice (a stored (v,u) ladder), and PointAt(t) re-polishes u from the nearest ladder entry and returns
// Torus.PointAt(u,v). The feet clamp both ends so the continuation cannot leave the bracket; the fold guard
// (|g′(u)| ≤ 2ρ(v)·weld — a double azimuth root, the section grazing the u-sweep) honest-rejects rather than
// jumping a branch. It is the edge a torus arm's cylinder-capping far-runout trim stores.
type TorusCylinderArc struct {
	Torus  Torus
	Cyl    Cylinder
	V0, V1 float64 // tube-angle range; t∈[0,1] maps to v = V0 + t·(V1−V0)

	ladder []torusCylSample // solved (v,u) continuation from V0 to V1, seeds for PointAt's re-polish
	weld   float64          // point-space polish tolerance (res.Weld()·scale)
}

// torusCylSample is one solved slice of the continuation ladder: the on-cylinder azimuth u at tube angle v.
type torusCylSample struct{ v, u float64 }

// torusCylLadderSteps is the number of continuation slices between the feet — dense enough that the seeded
// Newton at each slice stays on-branch (|Δu| < π/2), coarse enough to stay cheap; PointAt re-polishes so the
// ladder is only a seed provider, not the returned accuracy.
const torusCylLadderSteps = 128

// NewTorusCylinderArc builds the section arc from the two feet's exact chart coords (u0,v0)→(u1,v1): it runs
// the feet-bracketed Newton continuation and returns false on a fold or a branch jump anywhere between the
// feet (honest-reject, do-no-harm). weld is the point-space polish/fold tolerance. Example:
//
//	arc, ok := geom.NewTorusCylinderArc(torus, cyl, v0, u0, v1, u1, res.Weld()*scale)
func NewTorusCylinderArc(t Torus, cyl Cylinder, v0, u0, v1, u1, weld float64) (TorusCylinderArc, bool) {
	ladder, ok := buildTorusCylLadder(t, cyl, v0, u0, v1, u1, weld)
	if !ok {
		return TorusCylinderArc{}, false
	}
	return TorusCylinderArc{Torus: t, Cyl: cyl, V0: v0, V1: v1, ladder: ladder, weld: weld}, true
}

// buildTorusCylLadder sweeps v from v0 to v1, Newton-solving the on-cylinder u at each slice seeded from the
// previous one (from u0 at v0). Declines on a fold (g′→0) or a branch jump (|Δu|>π/2 between adjacent slices,
// which the linear v-map cannot cross without leaving the u(v)-graph). u1 is the second foot's azimuth; the
// endpoint match is certified by the caller (torusCylinderTrim) via PointAt(1) vs foot1.
func buildTorusCylLadder(t Torus, cyl Cylinder, v0, u0, v1, u1, weld float64) ([]torusCylSample, bool) {
	ladder := make([]torusCylSample, 0, torusCylLadderSteps+1)
	u := u0
	for i := 0; i <= torusCylLadderSteps; i++ {
		v := v0 + (v1-v0)*float64(i)/float64(torusCylLadderSteps)
		un, ok := solveTorusCylU(t, cyl, v, u, weld)
		if !ok {
			return nil, false // fold or non-convergence between the feet: the trim is not a single u(v)-graph
		}
		if i > 0 && stdmath.Abs(un-u) > stdmath.Pi/2 {
			return nil, false // branch jump (|Δu|>π/2): the continuation left the section branch
		}
		ladder = append(ladder, torusCylSample{v: v, u: un})
		u = un
	}
	_ = u1
	return ladder, true
}

// solveTorusCylU Newton-solves g(u)=|E|²−(E·â₂)²−R₂²=0 at fixed v, seeded from uSeed, with the derivative
// g′(u)=2E·∂P/∂u−2(E·â₂)(∂P/∂u·â₂). The fold guard rejects |g′(u)| ≤ 2ρ(v)·weld (ρ(v)=R′+r·cos v is the
// |∂P/∂u| scale) — a double azimuth root, the section grazing the u-sweep — returning false (honest-reject).
func solveTorusCylU(t Torus, cyl Cylinder, v, uSeed, weld float64) (float64, bool) {
	a2 := cyl.AxisDir.AsVector()
	rho := stdmath.Abs(t.MajorRadius + t.MinorRadius*stdmath.Cos(v))
	u := uSeed
	for iter := 0; iter < 40; iter++ {
		e := cyl.Origin.VectorTo(t.PointAt(u, v)) // E = P(u,v) − O₂
		ea := float64(e.Dot(a2))
		g := float64(e.LengthSquared()) - ea*ea - cyl.Radius*cyl.Radius
		du, _ := t.DerivativesAt(u, v) // ∂P/∂u
		gp := 2*float64(e.Dot(du)) - 2*ea*float64(du.Dot(a2))
		if stdmath.Abs(gp) <= 2*rho*weld {
			return 0, false // fold: |g′(u)| below the point-space floor
		}
		step := g / gp
		u -= step
		if stdmath.Abs(step)*rho <= weld {
			return u, true
		}
	}
	return 0, false // did not converge to the point-space floor in the iteration budget
}

// PointAt returns the point at t∈[0,1], evaluated on the torus at (u(v),v): v=V0+t(V1−V0), u re-polished from
// the nearest ladder entry so the point is EXACT on the torus and on the cylinder to weld.
func (a TorusCylinderArc) PointAt(t float64) math.Point3 {
	v := a.V0 + t*(a.V1-a.V0)
	return a.Torus.PointAt(a.polishU(v), v)
}

// polishU seeds u from the nearest ladder slice to v and re-solves the on-cylinder root there; on a fold at
// an interior re-polish it falls back to the seed (a ladder slice is itself a converged root).
func (a TorusCylinderArc) polishU(v float64) float64 {
	seed := a.nearestLadderU(v)
	if u, ok := solveTorusCylU(a.Torus, a.Cyl, v, seed, a.weld); ok {
		return u
	}
	return seed
}

// nearestLadderU returns the solved u of the ladder slice whose v is closest to the query v.
func (a TorusCylinderArc) nearestLadderU(v float64) float64 {
	best, bestD := a.ladder[0].u, stdmath.Abs(v-a.ladder[0].v)
	for _, s := range a.ladder[1:] {
		if d := stdmath.Abs(v - s.v); d < bestD {
			best, bestD = s.u, d
		}
	}
	return best
}

// TangentAt returns dP/dt = (V1−V0)·(du/dv·∂P/∂u + ∂P/∂v), du/dv from implicit differentiation of g=0
// (du/dv = −g_v/g_u). The chord-and-angle edge sampler drives off PointAt, so a large tangent near a fold is
// harmless.
func (a TorusCylinderArc) TangentAt(t float64) math.Vector3 {
	v := a.V0 + t*(a.V1-a.V0)
	u := a.polishU(v)
	du, dv := a.Torus.DerivativesAt(u, v)
	return du.Scale(math.Scalar(a.dUdV(u, v))).Add(dv).Scale(math.Scalar(a.V1 - a.V0))
}

// dUdV = −g_v/g_u where g_u=2E·∂P/∂u−2(E·â₂)(∂P/∂u·â₂) and g_v is the same with ∂P/∂v; zero when g_u vanishes
// (a fold, where the tangent is dominated by the ∂P/∂v term anyway).
func (a TorusCylinderArc) dUdV(u, v float64) float64 {
	a2 := a.Cyl.AxisDir.AsVector()
	e := a.Cyl.Origin.VectorTo(a.Torus.PointAt(u, v))
	ea := float64(e.Dot(a2))
	du, dv := a.Torus.DerivativesAt(u, v)
	gu := 2*float64(e.Dot(du)) - 2*ea*float64(du.Dot(a2))
	gv := 2*float64(e.Dot(dv)) - 2*ea*float64(dv.Dot(a2))
	if gu == 0 {
		return 0
	}
	return -gv / gu
}

// Domain returns [0, 1].
func (a TorusCylinderArc) Domain() (lo, hi float64) { return 0, 1 }
