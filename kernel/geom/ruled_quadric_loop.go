// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The FOLDED half of the ruled∩quadric section (ADR-0061 stage 5). [RuledQuadricArc] follows one
// ordered root of the ruling quadratic across the base's whole azimuth, which is the section's shape
// only while the ruling meets the quadric at EVERY azimuth — a rod driven right through a wall. A ball
// sitting off a cylinder's axis is the other shape: the ruling meets the sphere over an azimuth WINDOW
// and misses it outside, so the section is one closed loop that runs out along the upper root and back
// along the lower, the two meeting where the discriminant vanishes. Those meeting points are FOLDS: the
// branch label "upper" stops being defined there, and dv/du is infinite.
//
// A fold is a defect of the (u, v) GRAPH, not of the curve — the section is perfectly smooth in space,
// tangent to the ruling as it turns. So the loop is parametrised to be regular through it: the azimuth
// runs u(s) = m − w·cos s over one turn of s, whose speed vanishes at each fold exactly as fast as
// dv/du diverges (both like the square root of the distance to the fold), leaving dP/ds finite. That is
// the whole trick, and it is why this is a second curve type rather than a flag on the arc: the arc's
// parameter IS the azimuth, and no reparametrisation of it can be regular at a fold.

// RuledQuadricLoop is the EXACT closed intersection of a straight-ruled surface with an implicit
// quadric over one azimuth window [U0, U1], evaluated on the ruled surface's own chart. The parameter
// t ∈ [0,1] runs one turn: the first half follows the upper root from the fold at U0 to the fold at U1,
// the second half the lower root back, so PointAt(0) == PointAt(1) exactly.
//
// Example — the closed seam where a ball crosses a rod off its axis, on the rod's chart:
//
//	curves, ok := geom.IntersectSurfacesAnalytic(rodCylinder, ball, res)
type RuledQuadricLoop struct {
	Base   Surface // the straight-ruled surface the loop is evaluated on
	Quad   Quadric // the implicit form of the other surface
	U0, U1 float64 // the two fold azimuths bounding the window, U0 < U1
}

// Kind reports the loop as a ruled-quadric section: it is the same closed form as [RuledQuadricArc],
// over a window that closes on itself rather than on the base's period.
func (l RuledQuadricLoop) Kind() CurveKind { return CurveRuledQuadric }

// Domain returns [0, 1].
func (l RuledQuadricLoop) Domain() (lo, hi float64) { return 0, 1 }

// mid and half are the azimuth window's centre and half-width.
func (l RuledQuadricLoop) mid() float64  { return (l.U0 + l.U1) / 2 }
func (l RuledQuadricLoop) half() float64 { return (l.U1 - l.U0) / 2 }

// uAt maps one turn of s to the azimuth. The cosine is what makes the loop regular at its folds: it
// approaches each end quadratically in s, and the branch separation grows like the square root of that
// distance, so the two rates cancel.
func (l RuledQuadricLoop) uAt(s float64) float64 { return l.mid() - l.half()*stdmath.Cos(s) }

// upperHalf reports whether s is on the outbound (upper-root) half of the turn.
func upperHalf(s float64) bool { return s <= stdmath.Pi }

// PointAt returns the point at t ∈ [0,1], evaluated on the base surface at (u, v) with v the root the
// half-turn selects.
func (l RuledQuadricLoop) PointAt(t float64) math.Point3 {
	s := twoPi * t
	u := l.uAt(s)
	return l.Base.PointAt(u, l.coeffsAt(u).foldRoot(upperHalf(s)))
}

// TangentAt returns dP/dt = 2π·(du/ds·∂P/∂u + dv/ds·∂P/∂v). dv/ds splits into a regular part and the
// branch term ±(D′/4a)·(du/ds)/√Δ, whose two factors both vanish at a fold; branchRatio evaluates that
// quotient by its limit there, so the tangent stays finite and non-zero all the way round.
func (l RuledQuadricLoop) TangentAt(t float64) math.Vector3 {
	s := twoPi * t
	u := l.uAt(s)
	co := l.coeffsAt(u)
	v := co.foldRoot(upperHalf(s))
	duds := l.half() * stdmath.Sin(s)
	dvds := duds*co.regularDvDu(v) + l.branchRatio(s)*co.discriminantSlope()/(4*co.a)
	du, dv := l.Base.DerivativesAt(u, v)
	return du.Scale(math.Scalar(duds)).Add(dv.Scale(math.Scalar(dvds))).Scale(twoPi)
}

// coeffsAt is the ruling quadratic at azimuth u.
func (l RuledQuadricLoop) coeffsAt(u float64) ruledQuadricCoeffs {
	return l.Quad.alongRuling(straightRulingAt(l.Base, u))
}

// foldParamGuard is how close in s the tangent may be evaluated to a fold before √Δ is read by its
// limit instead of by dividing. Δ = b²−4ac cancels to zero AT a fold, so within a whisker of one its
// square root has lost most of its digits while the limit — which reads Δ′ at the fold itself, where
// nothing cancels — is exact. It is a distance in the curve's OWN parameter, not a length, so it does
// not scale with the model.
const foldParamGuard = 1e-3 // tol:numeric — parametric distance to a fold where √Δ loses its digits

// branchRatio returns ±(du/ds)/√Δ — the only factor of dv/ds that is singular at a fold, and the one
// place the loop's regularity is actually spent. Away from a fold it is the quotient itself, signed so
// that the outbound and return halves both walk the window forwards. Within foldParamGuard of one it is
// the limit √(2w/|Δ′|), read from Δ′ at that fold.
func (l RuledQuadricLoop) branchRatio(s float64) float64 {
	if d, uFold := l.nearestFold(s); d < foldParamGuard {
		slope := stdmath.Abs(l.coeffsAt(uFold).discriminantSlope())
		if slope <= 0 {
			return 0 // a double fold: the window closes on a tangency the conditioning gate refuses
		}
		return stdmath.Sqrt(2 * l.half() / slope)
	}
	co := l.coeffsAt(l.uAt(s))
	disc := co.discriminant()
	if disc <= 0 {
		return 0
	}
	return branchSign(s) * l.half() * stdmath.Sin(s) / stdmath.Sqrt(disc)
}

// branchSign is +1 on the outbound half and −1 on the return, which is exactly the sign that makes
// branchSign(s)·sin(s) non-negative over the whole turn — the two halves walk the window the same way.
func branchSign(s float64) float64 {
	if upperHalf(s) {
		return 1
	}
	return -1
}

// nearestFold returns the parametric distance from s to the nearer fold and that fold's azimuth. The
// folds sit at s = 0 (or 2π) and s = π.
func (l RuledQuadricLoop) nearestFold(s float64) (float64, float64) {
	if d := stdmath.Min(s, twoPi-s); d < stdmath.Abs(s-stdmath.Pi) {
		return d, l.U0
	}
	return stdmath.Abs(s - stdmath.Pi), l.U1
}
