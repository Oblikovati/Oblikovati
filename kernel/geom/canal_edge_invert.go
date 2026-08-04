// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// Warm-started LOCAL point inversion for the edge-anchored canal march. The march must
// never use the global closest-point search after station 0: on a closed host (a swept
// pipe) the global foot can leap to the antipodal sheet, which is a silent branch jump —
// warm-started local inversion is correctness, not an optimization. Parameters are LIFTED:
// a periodic u lives on the universal cover (evaluation wraps, the state does not), and an
// open host is evaluated unclamped so the march's end-overrun rides the B-spline's natural
// polynomial extension — the same prolong-then-trim OCCT's walking does past a restriction
// (BRepBlend_SurfRstLineBuilder.cxx).

// canalInvertMaxIter caps the damped Gauss–Newton foot refinement per inversion call.
const canalInvertMaxIter = 40

// canalInvertCosTol is the perpendicularity stop: the residual's cosine against both
// surface tangents (dimensionless, model-free) under which the foot is converged.
const canalInvertCosTol = 1e-12

// pointAtLifted evaluates the host at lifted parameters (periodic u wrapped into domain).
func (h CanalMarchHost) pointAtLifted(u, v float64) math.Point3 {
	return h.Surf.PointAt(h.wrapU(u), v)
}

// derivsAtLifted evaluates the host partials at lifted parameters.
func (h CanalMarchHost) derivsAtLifted(u, v float64) (du, dv math.Vector3) {
	return h.Surf.DerivativesAt(h.wrapU(u), v)
}

// normalAtLifted evaluates the host normal at lifted parameters.
func (h CanalMarchHost) normalAtLifted(u, v float64) math.Vector3 {
	return h.Surf.NormalAt(h.wrapU(u), v)
}

// NormalAtLifted is the exported lifted-parameter normal (periodic u wrapped into domain,
// open hosts evaluated unclamped on their natural extension) — the evaluation the
// certificate's reconstructed-centre needs at a foot SurfaceFootNear returned.
func (h CanalMarchHost) NormalAtLifted(u, v float64) math.Vector3 { return h.normalAtLifted(u, v) }

// wrapU maps a lifted u into the host's u-domain for evaluation; identity on an open host.
func (h CanalMarchHost) wrapU(u float64) float64 {
	if h.PeriodU <= 0 {
		return u
	}
	lo, _ := h.Surf.UDomain()
	w := stdmath.Mod(u-lo, h.PeriodU)
	if w < 0 {
		w += h.PeriodU
	}
	return lo + w
}

// invertWarm finds the tangency foot of q on the host: globally on the first call (station
// 0 — the seed centre is r from the host, so the global foot is unambiguous there), locally
// from the last foot afterwards. The result seeds the next call.
func (s *canalMarchHostState) invertWarm(q math.Point3) (CanalFoot, error) {
	f, err := s.peekInvert(q)
	if err != nil {
		return CanalFoot{}, err
	}
	s.prev, s.warm = f, true
	return f, nil
}

// peekInvert is invertWarm without advancing the warm seed — the line search probes
// candidate centres through it so a rejected candidate cannot poison the continuation.
func (s *canalMarchHostState) peekInvert(q math.Point3) (CanalFoot, error) {
	if !s.warm {
		u, v, p := ClosestPointOnSurface(s.host.Surf, q)
		return CanalFoot{U: u, V: v, P: p}, nil
	}
	return s.localFoot(q, s.prev)
}

// localFoot runs the damped, UNCLAMPED Gauss–Newton foot refinement from the given seed,
// wrapping a periodic u only at evaluation time (the lift). It errors on non-convergence
// and on a lifted jump past the branch guard — both are march-abort conditions, never
// silently absorbed.
func (s *canalMarchHostState) localFoot(q math.Point3, seed CanalFoot) (CanalFoot, error) {
	u, v := seed.U, seed.V
	for i := 0; i < canalInvertMaxIter; i++ {
		du, dv, r := s.footFrame(u, v, q)
		if footPerpendicular(du, dv, r) {
			return s.guardedFoot(u, v, seed)
		}
		su, sv, ok := gaussNewtonStep(du, dv, float64(du.Dot(r)), float64(dv.Dot(r)))
		if !ok {
			return CanalFoot{}, fmt.Errorf("degenerate tangent frame at lifted (%g, %g)", u, v)
		}
		nu, nv, moved := s.dampFootStep(q, u, v, su, sv, float64(r.LengthSquared()))
		if !moved {
			break // numeric floor: no step improves — the residual radius check arbitrates
		}
		u, v = nu, nv
	}
	// At the float floor the perpendicularity cosine can stall just above the tolerance while
	// the foot is already exact to machine noise; the caller's |dist−r| acceptance (weld-scaled)
	// is the arbiter, so a stalled-but-close foot is returned rather than refused (the same
	// break-on-no-improvement contract refineSurfaceParam ships).
	return s.guardedFoot(u, v, seed)
}

// footFrame evaluates the tangent frame and residual q − S(u,v) at lifted parameters.
func (s *canalMarchHostState) footFrame(u, v float64, q math.Point3) (du, dv, r math.Vector3) {
	du, dv = s.host.derivsAtLifted(u, v)
	r = s.host.pointAtLifted(u, v).VectorTo(q)
	return du, dv, r
}

// footPerpendicular is the scale-free convergence test: the residual's cosine against both
// tangents under canalInvertCosTol (or a residual at numeric zero).
func footPerpendicular(du, dv, r math.Vector3) bool {
	rl := float64(r.Length())
	if rl == 0 {
		return true
	}
	return tangentCosine(float64(du.Dot(r)), float64(du.Length()), rl) < canalInvertCosTol &&
		tangentCosine(float64(dv.Dot(r)), float64(dv.Length()), rl) < canalInvertCosTol
}

// dampFootStep backtracks the Gauss–Newton step until it reduces the squared distance,
// stepping in LIFTED coordinates (no clamp — overrun and seam crossings are legitimate).
// moved=false means even the smallest step fails to improve: the numeric floor.
func (s *canalMarchHostState) dampFootStep(q math.Point3, u, v, su, sv, d2 float64) (nu, nv float64, moved bool) {
	alpha := 1.0
	for k := 0; k < 8; k++ {
		cu, cv := u+alpha*su, v+alpha*sv
		r := s.host.pointAtLifted(cu, cv).VectorTo(q)
		if float64(r.LengthSquared()) < d2 {
			return cu, cv, true
		}
		alpha *= 0.5
	}
	return u, v, false
}

// guardedFoot packages a converged foot, refusing a lifted parameter jump past the branch
// guard: more than a quarter period on a periodic direction (or half the domain span on an
// open one) between consecutive inversions is a sheet/branch jump, not a continuation.
func (s *canalMarchHostState) guardedFoot(u, v float64, seed CanalFoot) (CanalFoot, error) {
	if du := stdmath.Abs(u - seed.U); du > s.uJumpGuard() {
		return CanalFoot{}, fmt.Errorf("foot u jumped %g (guard %g): branch jump", du, s.uJumpGuard())
	}
	if dv := stdmath.Abs(v - seed.V); dv > s.vJumpGuard() {
		return CanalFoot{}, fmt.Errorf("foot v jumped %g (guard %g): branch jump", dv, s.vJumpGuard())
	}
	return CanalFoot{U: u, V: v, P: s.host.pointAtLifted(u, v)}, nil
}

// SurfaceFootNear is the one-shot exported form of the warm local inversion: the
// perpendicular foot of q on host near the seed (u0, v0), UNCLAMPED (a foot just past an
// open domain edge lands on the polynomial extension instead of piling on the boundary —
// the measurement a global, clamped closest-point cannot make) and periodic-aware.
//
//	u, v, p, err := geom.SurfaceFootNear(geom.CanalMarchHost{Surf: wall}, u0, v0, q)
func SurfaceFootNear(host CanalMarchHost, u0, v0 float64, q math.Point3) (u, v float64, p math.Point3, err error) {
	s := &canalMarchHostState{host: host, warm: true, prev: CanalFoot{U: u0, V: v0, P: host.pointAtLifted(u0, v0)}}
	f, ferr := s.localFoot(q, s.prev)
	if ferr != nil {
		return 0, 0, math.Point3{}, ferr
	}
	return f.U, f.V, f.P, nil
}

// uJumpGuard bounds one inversion's u-motion: a quarter period on a periodic host, half
// the u-domain span otherwise.
func (s *canalMarchHostState) uJumpGuard() float64 {
	if s.host.PeriodU > 0 {
		return s.host.PeriodU / 4
	}
	lo, hi := s.host.Surf.UDomain()
	return (hi - lo) / 2
}

// vJumpGuard bounds one inversion's v-motion: half the v-domain span.
func (s *canalMarchHostState) vJumpGuard() float64 {
	lo, hi := s.host.Surf.VDomain()
	return (hi - lo) / 2
}
