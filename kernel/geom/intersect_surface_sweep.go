// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// ONE step of the SSI march: propose, correct, accept or halve (split out of
// intersect_surface_trace.go for #2214).
//
// The sweep owns the step size. It plans a step from the local curvature, and on a rejected
// correction halves and retries rather than accepting a point off the intersection. Closure — the
// march arriving back at its start — is decided here too, because it is a property of the sweep's
// own history, not of the curve it produced.

// ssiStepVerdict classifies one predict-correct attempt.
type ssiStepVerdict int

const (
	ssiStepAccepted ssiStepVerdict = iota
	ssiStepRejected
	ssiStepExited
	ssiStepClosed
)

// ssiSweep is the mutable state of one directional march: current point and heading, adaptive step,
// accumulated arc length, the last accepted point (for the discrete curvature), and the tangent of the
// first accepted step (the closure gate's direction reference — the seed's analytic tangent may sit
// within tolerance of a pinch where nb×no is unreliable).
type ssiSweep struct {
	tr           ssiTracer
	p, start     math.Point3
	pPrev        math.Point3
	havePrev     bool
	dir          math.Vector3
	startTan     math.Vector3
	haveStartTan bool
	h, arc       float64
	align        float64 // |nb·no| at the last corrected point (tangency proximity)
	rejectsAtMin int
	pts          []math.Point3
}

// attemptStep predicts one step of h along the heading, corrects onto the curve, and runs the
// acceptance gates in order: corrector convergence, forward progress (a corrector landing behind the
// heading has jumped branches or folded), corrector displacement, tangent turn, window exit, closure.
func (sw *ssiSweep) attemptStep() (math.Point3, math.Vector3, ssiStepVerdict) {
	pred := sw.p.TranslateBy(sw.dir.Scale(math.Scalar(sw.h)))
	if !sw.tr.charge() {
		return pred, math.Vector3{}, ssiStepExited // corrector budget spent: stop marching, decline above
	}
	pc, nbc, noc, ok := correctToBothSurfaces(sw.tr.base, sw.tr.other, pred, sw.tr.tol)
	if !ok {
		return pc, math.Vector3{}, ssiStepRejected
	}
	moved, err := math.UnitVector3FromVector(sw.p.VectorTo(pc))
	if err != nil || float64(moved.AsVector().Dot(sw.dir)) <= 0 {
		return pc, math.Vector3{}, ssiStepRejected // no progress, or backtracked
	}
	if d := float64(pred.DistanceTo(pc)); d > ssiDisplacementGate*sw.h || d > ssiDisplacementSagGate*sw.tr.eps {
		return pc, math.Vector3{}, ssiStepRejected // corrector left the prediction / sag over budget
	}
	tc := orient(nbc.Cross(noc), moved.AsVector())
	sw.align = stdmath.Abs(float64(nbc.Dot(noc)))
	if float64(tc.Dot(sw.dir)) < ssiTurnGateCos {
		return pc, tc, ssiStepRejected // turned harder than any same-branch step can (#1598)
	}
	if !inWindow(sw.tr.base, pc, sw.tr.g) {
		return pc, tc, ssiStepExited
	}
	if sw.closes(pc, tc) {
		return pc, tc, ssiStepClosed
	}
	return pc, tc, ssiStepAccepted
}

// closes reports whether pc closes the loop: enough arc marched, back within ¾ of the CURRENT step of
// the start, AND heading the same way as the sweep's first accepted step — position alone falsely
// closes at a pinch, where the other branch passes through the start point at a crossing angle (#1404).
func (sw *ssiSweep) closes(pc math.Point3, tc math.Vector3) bool {
	if !sw.haveStartTan || sw.arc <= ssiCloseArcSteps*sw.h {
		return false
	}
	if float64(sw.start.DistanceTo(pc)) >= sw.h*ssiLoopCloseSteps {
		return false
	}
	return float64(tc.Dot(sw.startTan)) >= ssiCloseTangentCos
}

// halve rejects the pending step: shrink h toward the floor, and once AT the floor allow a bounded
// number of retries before the sweep gives up (returning the curve traced so far, open).
func (sw *ssiSweep) halve() bool {
	if sw.h > sw.tr.hMin {
		sw.h = stdmath.Max(sw.h/2, sw.tr.hMin)
		return true
	}
	sw.rejectsAtMin++
	return sw.rejectsAtMin <= ssiRejectRetriesAtMin
}

// accept commits pc: record it, advance the heading, and re-plan h from the discrete curvature of the
// last three accepted points via h = 2√(2ε/κ), rate-limited to [½, ×1.4] per step and clamped to
// [hMin, hMax]. The one-step lag of the discrete estimate is covered by the acceptance gates.
func (sw *ssiSweep) accept(pc math.Point3, tc math.Vector3) {
	sw.rejectsAtMin = 0
	if !sw.haveStartTan {
		sw.startTan, sw.haveStartTan = tc, true
	}
	sw.arc += float64(sw.p.DistanceTo(pc))
	hPlan := sw.tr.hMax
	if kappa := sw.curvatureEstimate(pc, tc); kappa > 0 {
		hPlan = ssiStepSafety * 2 * stdmath.Sqrt(2*sw.tr.eps/kappa)
	}
	// Tangency brake: the ANALYTIC intersection-curve curvature carries a 1/sin²θ pole at
	// a tangency (θ = angle between the surface normals) that the discrete estimate — the
	// finite per-branch curvature — cannot see. Approaching a pinch (#1404), cap the step
	// at half the bootstrap so the delicate neighbourhood is sampled densely and the
	// pinch crossing is REPRESENTED by vertices, not merely straddled within the sag
	// budget by one long chord.
	if sw.align > ssiTangencyBrakeCos {
		hPlan = stdmath.Min(hPlan, sw.tr.step/2)
	}
	sw.h = clampStep(hPlan, sw.h, sw.tr.hMin, sw.tr.hMax)
	sw.pPrev, sw.havePrev = sw.p, true
	sw.p, sw.dir = pc, tc
	sw.pts = append(sw.pts, pc)
}

// curvatureEstimate is the controller's κ: the larger of the 3-point circumcircle estimate and
// the tangent turning rate Δθ/Δs over the accepted step. The turn rate is available from the very
// FIRST step (the circumcircle needs three points), so the bootstrap adapts one step sooner; both
// saturate near a pinch instead of diverging like the analytic intersection-curve curvature,
// driving h to the floor there — exactly the wanted slowdown.
func (sw *ssiSweep) curvatureEstimate(pc math.Point3, tc math.Vector3) float64 {
	kappa := sw.discreteCurvature(pc)
	ds := float64(sw.p.DistanceTo(pc))
	if ds <= 10*sw.tr.tol {
		return kappa
	}
	cosTurn := stdmath.Min(1, stdmath.Max(-1, float64(tc.Dot(sw.dir))))
	if turn := stdmath.Acos(cosTurn) / ds; turn > kappa {
		return turn
	}
	return kappa
}

// discreteCurvature is the circumcircle curvature of (pPrev, p, pc) — exact for circles, O(h²)
// otherwise, always finite (it saturates near a pinch instead of diverging like the analytic
// intersection-curve curvature, which drives h to the floor there — exactly the wanted slowdown).
// Spacings below 10·tol are corrector jitter, not geometry: keep the previous plan (return 0).
func (sw *ssiSweep) discreteCurvature(pc math.Point3) float64 {
	if !sw.havePrev {
		return 0
	}
	e1, e2 := sw.pPrev.VectorTo(sw.p), sw.p.VectorTo(pc)
	l1, l2 := float64(e1.Length()), float64(e2.Length())
	chord := float64(sw.pPrev.DistanceTo(pc))
	if l1 < 10*sw.tr.tol || l2 < 10*sw.tr.tol || chord == 0 {
		return 0
	}
	return 2 * float64(e1.Cross(e2).Length()) / (l1 * l2 * chord)
}

// clampStep rate-limits the controller (halve fast, grow ≤×1.4) and clamps to the global bounds.
func clampStep(plan, current, hMin, hMax float64) float64 {
	h := stdmath.Min(stdmath.Max(plan, current/2), current*ssiStepGrow)
	return stdmath.Min(stdmath.Max(h, hMin), hMax)
}

// curveTangent returns the unit curve tangent nb×no, reversed for the backward sweep; ok is false at a
// tangency (parallel normals, zero-length cross product).
func curveTangent(nb, no math.Vector3, forward bool) (math.Vector3, bool) {
	t, err := math.UnitVector3FromVector(nb.Cross(no))
	if err != nil {
		return math.Vector3{}, false
	}
	d := t.AsVector()
	if !forward {
		d = d.Scale(-1)
	}
	return d, true
}

// orient flips t to agree with the heading h (so the march does not reverse onto itself).
func orient(t, h math.Vector3) math.Vector3 {
	u, err := math.UnitVector3FromVector(t)
	if err != nil {
		return h
	}
	d := u.AsVector()
	if d.Dot(h) < 0 {
		d = d.Scale(-1)
	}
	return d
}
