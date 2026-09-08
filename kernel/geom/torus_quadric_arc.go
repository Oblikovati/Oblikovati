// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The TORUS side of the intersector's parametric×implicit split (ADR-0061 stage 5, second slice).
//
// A torus is quartic, so it has no implicit quadric of its own to substitute a ruling into and the
// ruled bucket cannot reach it. But the substitution runs the other way just as well. A point on the
// torus is
//
//	P(u, v) = C + ρ(v)·e(u) + r·sin v·â,   ρ(v) = R + r·cos v,   e(u) = cos u·ê₁ + sin u·ê₂
//
// which is AFFINE in e(u) — the same shape a ruled surface has in its ruling parameter, with the tube
// angle v as the station and the azimuth u as the unknown. Substituting it into a quadric leaves
//
//	A(v) + ρ(v)·(T(v)·e(u)) + (e·Me)·ρ(v)² = 0
//
// and when the quadric's quadratic form M is INVARIANT about the torus axis, e·Me is a constant α and
// the whole u-dependence is the single harmonic T(v)·e(u) = |T⊥|·cos(u − ψ). One harmonic has a closed
// form: u = ψ ± arccos(−A/(ρ|T⊥|)), two ordered roots wherever |A| ≤ ρ|T⊥|.
//
// So the family this reaches is every quadric whose M commutes with rotation about the torus axis: a
// SPHERE anywhere, and a cylinder or cone whose axis is PARALLEL to the torus axis. In CAD terms that
// is a ball meeting a ring and — much more often — an axial hole or boss drilled through one. A skew
// rod is not in it: its M is not axis-invariant, the u-dependence is a second harmonic, and the roots
// are a quartic in tan(u/2) rather than an arccos.
//
// The topology is then the same question the ruled form asks, and periodicRootWindows answers it once
// for both: the maximal v-spans where the two roots exist, with folds at the ends. [TorusQuadricArc]
// carries a branch over a full period, [TorusQuadricLoop] a folded window.

// torusHarmonic is the quadric's constraint on the torus at one tube angle, reduced to one harmonic in
// the azimuth: level + reach·cos(u − phase) = 0, so the two azimuths are phase ± arccos(−level/reach).
type torusHarmonic struct {
	level, reach, phase float64
}

// discriminant is reach² − level², positive exactly where the tube circle at this station crosses the
// quadric twice. It is the torus form's counterpart of the ruling quadratic's b² − 4ac, and it feeds
// the same shared window finder.
func (h torusHarmonic) discriminant() float64 { return h.reach*h.reach - h.level*h.level }

// root returns the upper or lower azimuth of the two the harmonic admits, ordered by their offset from
// the phase so that "upper" names the same branch at every station. It returns the fold azimuth itself
// — the phase — where the discriminant has fallen to zero and the two have merged, which is what lets a
// loop built on a window close exactly on its ends.
func (h torusHarmonic) root(upper bool) float64 {
	if h.reach == 0 {
		return stdmath.NaN() // no azimuth dependence at all: the section is whole circles, not two roots
	}
	arg := stdmath.Max(-1, stdmath.Min(1, -h.level/h.reach))
	if upper {
		return h.phase + stdmath.Acos(arg)
	}
	return h.phase - stdmath.Acos(arg)
}

// torusAxisFrame is the torus's own orthonormal frame: the axis and the two in-plane directions its
// azimuth sweeps.
func torusAxisFrame(t Torus) (axis, e1, e2 math.Vector3) {
	axis = t.AxisDir.AsVector()
	e1 = t.Ref.AsVector()
	return axis, e1, axis.Cross(e1)
}

// quadricIsAxisInvariant reports whether M acts the same way on every direction perpendicular to the
// axis — the condition that collapses the substitution to ONE harmonic. It is a test on the tensor, not
// on the surface's type: a sphere passes wherever it sits, a cylinder or cone passes exactly when its
// own axis is parallel to the torus's, and anything else fails. Gate on conditioning, not on type.
func quadricIsAxisInvariant(q Quadric, e1, e2 math.Vector3) (alpha float64, ok bool) {
	m11, m22, m12 := inPlaneTensorEntries(q, e1, e2)
	return m11, axisInvariantEntries(q, m11, m22, m12)
}

// axisInvarianceTol is how far a quadratic form may depart from acting the same way on every direction
// perpendicular to the torus axis and still count as invariant. It is RELATIVE to the tensor's own
// entries — a comparison of coefficients with coefficients — so it carries no model scale: the same
// pair written in metres and in millimetres classifies identically.
const axisInvarianceTol = 1e-12 // tol:numeric — relative departure of a tensor from axial symmetry

// torusHarmonicAt reduces the quadric's constraint on the torus at tube angle v. ok=false when M is not
// axis-invariant, in which case no single harmonic describes the station and this form does not apply.
func torusHarmonicAt(t Torus, q Quadric, v float64) (torusHarmonic, bool) {
	st := torusStationAt(t, q, v)
	if !st.invariant {
		return torusHarmonic{}, false
	}
	return st.harmonic(), true
}

// TorusQuadricArc is a bounded run of the EXACT intersection of a torus with an axis-invariant quadric,
// evaluated on the TORUS chart so the point is exactly on the torus and on the quadric to the harmonic
// solve's rounding. The parameter t ∈ [0,1] maps to the tube angle v = V0 + t·(V1−V0), and the point is
// Torus.PointAt(u, v) with u the Upper or lower azimuth the harmonic admits there.
//
// Example — the two seams an axial drill leaves in a ring, on the ring's chart:
//
//	curves, ok := geom.IntersectSurfacesAnalytic(ring, drill, res)
type TorusQuadricArc struct {
	Torus  Torus   // the torus the arc is evaluated on
	Quad   Quadric // the implicit form of the other surface
	Upper  bool    // which of the two azimuths this branch follows
	V0, V1 float64 // tube-angle range; t∈[0,1] maps to v = V0 + t·(V1−V0)
}

// Kind reports the arc as a torus section — the same closed form as [TorusQuadricLoop], over a range
// that closes on the torus's own period rather than on a fold.
func (a TorusQuadricArc) Kind() CurveKind { return CurveTorusQuadric }

// Domain returns [0, 1].
func (a TorusQuadricArc) Domain() (lo, hi float64) { return 0, 1 }

// vAt maps the curve parameter to the tube angle.
func (a TorusQuadricArc) vAt(t float64) float64 { return a.V0 + t*(a.V1-a.V0) }

// PointAt returns the point at t ∈ [0,1], evaluated on the torus at (u, v).
func (a TorusQuadricArc) PointAt(t float64) math.Point3 {
	v := a.vAt(t)
	h, _ := torusHarmonicAt(a.Torus, a.Quad, v)
	return a.Torus.PointAt(h.root(a.Upper), v)
}

// TangentAt returns dP/dt = (V1−V0)·(∂P/∂v + du/dv·∂P/∂u), with du/dv from a central difference of the
// harmonic's own root — the reduction is a composition of trigonometric evaluations whose analytic
// derivative would restate the same terms with no gain in accuracy, and the conditioning gate keeps a
// built arc away from the folds where du/dv diverges.
func (a TorusQuadricArc) TangentAt(t float64) math.Vector3 {
	v := a.vAt(t)
	du, dv := a.Torus.DerivativesAt(a.azimuthAt(v), v)
	step := torusArcDerivativeStep * stdmath.Abs(a.V1-a.V0)
	slope := shortestTurnDelta(a.azimuthAt(v-step), a.azimuthAt(v+step)) / (2 * step)
	return dv.Add(du.Scale(math.Scalar(slope))).Scale(math.Scalar(a.V1 - a.V0))
}

// azimuthAt is this branch's azimuth at tube angle v.
func (a TorusQuadricArc) azimuthAt(v float64) float64 {
	h, _ := torusHarmonicAt(a.Torus, a.Quad, v)
	return h.root(a.Upper)
}

// torusArcDerivativeStep is the central difference's step as a fraction of the arc's own tube-angle
// span, chosen so the truncation error (∝ step²) and the cancellation error (∝ ε/step) are both far
// below the weld at double precision.
const torusArcDerivativeStep = 1e-5 // tol:numeric — a fraction of the arc's parameter span
