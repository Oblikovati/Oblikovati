// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The folded half of the torus section (ADR-0061 stage 5), the exact counterpart of
// [RuledQuadricLoop] on the other parametric surface. Where the quadric reaches the tube over PART of
// its turn only — an axial drill through a ring reaches it around the ring's inner and outer flanks but
// not over its top and bottom — the two azimuths meet at the tube angles where the harmonic's
// discriminant vanishes, and the section is one closed loop out along the upper root and back along the
// lower. The cosine reparametrisation is the same trick and for the same reason: du/dv is infinite at a
// fold, and the station's speed vanishes there at exactly the rate that cancels it.

// TorusQuadricLoop is the EXACT closed intersection of a torus with an axis-invariant quadric over one
// tube-angle window [V0, V1], evaluated on the torus's own chart. The parameter t ∈ [0,1] runs one
// turn: the first half follows the upper azimuth from the fold at V0 to the fold at V1, the second half
// the lower azimuth back, so PointAt(0) == PointAt(1) exactly.
type TorusQuadricLoop struct {
	Torus  Torus   // the torus the loop is evaluated on
	Quad   Quadric // the implicit form of the other surface
	V0, V1 float64 // the two fold tube angles bounding the window, V0 < V1
}

// Kind reports the loop as a torus section: the same closed form as [TorusQuadricArc], over a window
// that closes on itself rather than on the torus's period.
func (l TorusQuadricLoop) Kind() CurveKind { return CurveTorusQuadric }

// Domain returns [0, 1].
func (l TorusQuadricLoop) Domain() (lo, hi float64) { return 0, 1 }

// mid and half are the tube-angle window's centre and half-width.
func (l TorusQuadricLoop) mid() float64  { return (l.V0 + l.V1) / 2 }
func (l TorusQuadricLoop) half() float64 { return (l.V1 - l.V0) / 2 }

// vAt maps one turn of s to the tube angle, by the cosine that makes the loop regular at its folds.
func (l TorusQuadricLoop) vAt(s float64) float64 { return l.mid() - l.half()*stdmath.Cos(s) }

// PointAt returns the point at t ∈ [0,1], on the azimuth the half-turn selects.
func (l TorusQuadricLoop) PointAt(t float64) math.Point3 {
	s := twoPi * t
	v := l.vAt(s)
	return l.Torus.PointAt(l.azimuthAt(s, v), v)
}

// azimuthAt is the azimuth of the branch s selects at tube angle v. At and just outside a fold the
// harmonic's root reader clamps to the phase, so the two halves meet at exactly the same point.
func (l TorusQuadricLoop) azimuthAt(s, v float64) float64 {
	h, _ := torusHarmonicAt(l.Torus, l.Quad, v)
	return h.root(upperHalf(s))
}

// TangentAt returns dP/dt by the same central difference [TorusQuadricArc.TangentAt] uses, in the
// LOOP's own parameter — which is where the cosine earns its keep. du/dv diverges at each fold, but
// du/ds does not, so differencing the composed azimuth u(s) rather than u(v) stays finite all the way
// round without a limit to special-case.
func (l TorusQuadricLoop) TangentAt(t float64) math.Vector3 {
	s := twoPi * t
	v := l.vAt(s)
	du, dv := l.Torus.DerivativesAt(l.azimuthAt(s, v), v)
	step := torusLoopDerivativeStep
	dvds := (l.vAt(s+step) - l.vAt(s-step)) / (2 * step)
	before := l.azimuthAt(s-step, l.vAt(s-step))
	after := l.azimuthAt(s+step, l.vAt(s+step))
	duds := shortestTurnDelta(before, after) / (2 * step)
	return dv.Scale(math.Scalar(dvds)).Add(du.Scale(math.Scalar(duds))).Scale(twoPi)
}

// torusLoopDerivativeStep is the central difference's step in the loop's own turn parameter. The
// composed azimuth is smooth in s through the folds, so a fixed fraction of a turn balances truncation
// against cancellation without knowing where in the window the query sits.
const torusLoopDerivativeStep = 1e-5 // tol:numeric — a fraction of the loop's own turn
