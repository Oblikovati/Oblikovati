// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"slices"

	"oblikovati.org/math"
)

// Surface–surface intersection by a predictor–corrector continuation tracer (Oblikovati#1319). The
// old path contoured a signed-distance field on a fixed 96×96 grid (marching squares), which silently
// loses sub-grid loops, never sees a tangential contact (the field does not change sign), and pairs
// saddle branches arbitrarily. This tracer instead marches the joint zero of the two surfaces: from
// each seed it corrects onto the curve (project onto the intersection of the two tangent planes) and
// steps along nb×no, the curve tangent, with adaptive direction — so loops close, branches stay
// connected, and an isolated tangency (parallel normals) is reported as a point rather than dropped.
//
// A curve that PASSES THROUGH a tangency or pinch — where two intersection branches cross and the
// surface normals momentarily align, e.g. the two ellipses of an equal-radius Steinmetz crossing — is
// traced straight through the singularity rather than stopping at it (Oblikovati#1404): the corrector
// damps to a steepest-descent step where the tangent-plane intersection is ill-conditioned, so the
// march re-acquires the curve on the far side and the loop closes instead of being dropped open.

const (
	ssiCorrectIters     = 40   // max corrector iterations per point
	ssiMaxStepsPerCurve = 6000 // guard against a non-closing march
	ssiMaxCurves        = 256  // guard against pathological seeding
	// ssiTangencyCos is the |nb·no| above which the surfaces count as tangent (normals (anti)parallel).
	// It must be loose enough to cover the FUZZY neighbourhood of a tangency, where the normals are
	// nearly — but not exactly — parallel: too strict a value (e.g. 1−1e-10) lets the corrector treat a
	// point a hair off the contact as a real crossing and emit a spurious tiny curve there. 1−1e-5
	// corresponds to crossings shallower than ~0.18°, which are tangencies for all practical purposes.
	ssiTangencyCos = 1 - 1e-5 // tol:angular — surface-normal alignment cosine at a tangency
	// ssiToleranceFraction and ssiStepFraction scale the on-curve tolerance and the march step to the
	// base patch's 3D extent (so both are model-relative, ADR-0042). 1e-7 is the stated acceptance
	// tolerance reachable by the NURBS Gauss–Newton projection; 4e-3 keeps a full curve to a few
	// thousand points while resolving curvature.
	ssiToleranceFraction = 1e-7 // tol:numeric — dimensionless fraction of the patch 3D extent (model-relative)
	// ssiStepFraction is the INITIAL march step h₀ (the pre-#1598 fixed step, now only a
	// bootstrap guess): the curvature controller below adapts h from there each accepted
	// step, and the acceptance gates halve it out of high-curvature trouble at the first
	// prediction if the seed sits in a tight region.
	ssiStepFraction = 4e-3 // tol:numeric — dimensionless fraction of the patch 3D extent (model-relative)
	// Curvature-based step control (#1598, audit A2; Bajaj et al. 1988). The chord of a
	// step h on a curve of curvature κ sags ≈ κh²/8; bounding the sag by the chordal
	// budget ε gives h = 2√(2ε/κ). ε is the DOWNSTREAM deflection budget (what the
	// splitter/tessellator absorbs, ADR-0042 model-relative) — three decades looser than
	// the on-curve tolerance. h is clamped to [min,max] fractions of the extent: the max
	// keeps a straight line from crawling, the min (200× the on-curve tol) is what lets
	// loops far smaller than the old fixed step be traced at all.
	ssiChordFraction   = 1e-4 // tol:numeric — chordal sag budget ε as a fraction of the extent
	ssiMinStepFraction = 2e-5 // tol:numeric — h floor (200× on-curve tol; below it corrector jitter dominates)
	// The h ceiling equals the pre-#1598 fixed step: the controller only ever REFINES
	// below the historical spacing, never coarsens past it — the curved-boolean
	// split/stitch consumers are tuned to chords of at most that length, and letting
	// h grow 5× beyond it silently corrupted near-tangent unions (their long low-κ
	// flanks coarsened while the loops themselves stayed correct). Raising the ceiling
	// is a follow-up for when the consumers derive their gates from the polyline
	// itself rather than the historical step (#1403 pipeline work).
	ssiMaxStepFraction = ssiStepFraction
	// Per-step controller limits: grow at most ×1.4 per accepted step (≈√2 — doubles h in
	// two steps on a straightening curve), halve on rejection; at the h floor allow two
	// retries before terminating the sweep OPEN (an honest partial curve beats marching
	// onto the wrong branch).
	ssiStepGrow           = 1.4
	ssiRejectRetriesAtMin = 2
	// Step acceptance gates (the OCCT IntWalk_PWalking StatusDeflection pattern in object
	// space). A legitimate same-branch step turns by ≈ κh = 2√(2εκ) ≤ ~40° at the h floor,
	// so 45° is the tightest turn gate that never rejects a legal step — while a
	// branch-jump at a pinch turns by the full crossing angle or reverses outright. The
	// corrector displacement on a legal step is ≈ the sag ≈ ε ≪ h, so half a step is a
	// generous displacement bound that only a jump or a curvature under-estimate trips.
	ssiTurnGateCos = 0.70710678118654752 // cos 45° — max tangent turn per accepted step
	// The corrector displacement on a same-branch step is ≈ κh²/2 = 4× the chord sag, so
	// gating it at 4ε enforces the sag budget DIRECTLY — including on the very first
	// bootstrap steps, before any curvature estimate exists. The ½h cap stays as the
	// coarse branch-jump bound. The step law plans with a 0.8 safety factor so a
	// κ-planned step sits at ≈2.6ε, comfortably inside the gate.
	ssiDisplacementGate    = 0.5 // max corrector displacement as a fraction of h
	ssiDisplacementSagGate = 4.0 // max corrector displacement as a multiple of ε
	ssiStepSafety          = 0.8 // fraction of the exact sag-limited step the law plans
	// Loop closure requires position AND direction (#1404): within ¾ of the CURRENT step
	// of the start, with the tangent aligned to the first accepted step's tangent within
	// 30° — at a pinch the other branch crosses at a steeper angle (90° for equal-radius
	// Steinmetz) and is rejected, so the march keeps going instead of falsely closing.
	// The gate opens only after 3 current-steps of accumulated arc length, so a loop
	// traced at the h floor can still close while the first few steps cannot self-close.
	ssiCloseTangentCos = 0.86602540378443865 // cos 30° — closure tangent alignment
	// ssiTangencyBrakeCos opens the near-tangency brake band: once the surface normals
	// align within ~8° the march is entering a pinch/tangency neighbourhood and the step
	// caps at half the bootstrap (see accept), regardless of the discrete curvature.
	ssiTangencyBrakeCos = 0.99 // tol:angular
	ssiCloseArcSteps    = 3.0  // min accumulated arc length before closing, in current steps
	// ssiMaxArcExtents caps a sweep by ACCUMULATED ARC LENGTH (in extents): the old
	// fixed-step count cap alone would let a pathological h-floor march spin thousands of
	// tiny steps; a real intersection curve on the patch cannot exceed a few extents.
	ssiMaxArcExtents = 64.0
	// Step multiples used while marching: a seed within ssiDedupSteps of an existing curve is a
	// duplicate of it; the loop closes when the march returns within ssiLoopCloseSteps of its start;
	// a tangency seed may sit up to ssiTangencyGapSteps from the contact (the strict normal test, not
	// this gate, is the real discriminator). ssiDedupSteps pairs with the SEGMENT-distance dedup in
	// nearAnyCurve (#1597): a duplicate seed corrects onto the already-traced curve and sits within
	// chord sag of its polyline (≤ step²·κ/8, ≪ a step), while a genuinely distinct neighbouring loop
	// keeps its full 3D separation — so the radius must cover sag, not a whole step, or two
	// near-tangent circles closer than a step apart collapse into one.
	ssiDedupSteps       = 0.25
	ssiLoopCloseSteps   = 0.75
	ssiTangencyGapSteps = 5.0
	// ssiDescentGain damps the steepest-descent corrector step taken where the two surface normals are
	// near-parallel (a tangency/pinch), so the corrector converges onto the curve through that singular
	// neighbourhood instead of diverging — see correctorStep (#1404). A full (gain 1) Newton-on-the-gaps
	// step overshoots when the normals coincide; half-stepping is unconditionally stable and still
	// monotone, costing only a few extra iterations through the (rare, localised) near-parallel band.
	ssiDescentGain = 0.5 // tol:numeric — under-relaxation of the near-tangency descent step
)

// ssiTracer holds the immutable context for one surface↔surface trace: the two surfaces, the base's
// parameter window, and the model-relative tolerance and march step. Bundling it keeps the marching
// methods to a handful of varying arguments instead of threading eight of them through every call.
type ssiTracer struct {
	base, other Surface
	g           SurfaceGrid
	step, tol   float64 // step = the BOOTSTRAP h₀; the controller adapts from it (#1598)
	eps         float64 // chordal sag budget ε (model-relative)
	hMin, hMax  float64 // adaptive-step clamps
	arcCap      float64 // max accumulated arc length per sweep
	// spent counts corrector calls across the WHOLE trace, shared by every sweep (the tracer is copied
	// by value into each ssiSweep, so this has to be a pointer to accumulate). See ssiMaxCorrections.
	spent *int
}

// ssiMaxCorrections is a NON-TERMINATION GUARD on one trace's corrector calls, not a tuned threshold.
// A kernel operation may not run without bound, so a continuation that cannot resolve a pair has to
// stop; this is where it stops.
//
// ★ IT SHOULD NEVER FIRE ON A RESOLVABLE PAIR. If it does, that is a DEFECT TO INVESTIGATE, not a knob
// to turn up. Measured, every trace that resolves is far below it: the whole kernel/ops corpus — real
// filleted and boolean bodies — peaks at 921 corrections, and kernel/geom's deliberately hard cases
// peak at 44032 (torus∩plane 17334, near-tangent sphere/cylinder 1844, ripple 1910, near-pinch 1616,
// Steinmetz 1128, NURBS patch cut by a plane 299). The value here is that ceiling with an order of
// magnitude of headroom, so a body harder than anything in the corpora still resolves.
//
// ★ IT IS A COUNT, NEVER A CLOCK. A wall-clock deadline would make the verdict depend on machine load,
// and the kernel's output must be byte-identical across runs and platforms. A count is spent in a
// deterministic traversal order, so the same pair declines on every machine.
//
// ★ IT IS NOT A FIX FOR SLOWNESS, AND MUST NOT BE MISTAKEN FOR ONE. The guard is PER TRACE while the
// cost that motivated it is PER BODY: the U4 dual-host weld reaches the intersector on 17 face pairs,
// so any guard generous enough not to decline correct work still admits millions of corrections across
// one self-intersection scan. Each of those corrections is a NURBS point inversion that rebuilds a
// knot-span seed lattice and rescans it (geom.BSplineSurface.ParamAt → nearestSeed), which is the
// actual cost and is tracked as Oblikovati/Oblikovati#3490; the seed lattice itself is memoized
// per surface now, which removes the rebuild-and-rescan half of that term. Lowering this constant to make such a body
// finish would decline correct work instead of making the work cheaper, and is the wrong trade.
const ssiMaxCorrections = 1000000

// charge records one corrector call and reports whether the trace may continue. A trace with no budget
// installed (the zero tracer some tests build) is never charged.
func (t ssiTracer) charge() bool {
	if t.spent == nil {
		return true
	}
	*t.spent++
	return *t.spent <= ssiMaxCorrections
}

// exhausted reports whether the trace has spent its corrector budget.
func (t ssiTracer) exhausted() bool { return t.spent != nil && *t.spent > ssiMaxCorrections }

// traceIntersectionCurves returns the intersection curve(s) of base and other as polylines whose every
// point lies on BOTH surfaces within tol. tol and the march step are model-relative (derived from the
// base's domain extent in 3D). Tangential contacts yield a single-point "curve" flagged by being a
// degenerate (1–2 point) polyline at a point where the normals are parallel.
func traceIntersectionCurves(base, other Surface, grid SurfaceGrid) ([][]math.Point3, bool) {
	g := resolveGrid(base, grid)
	if g.UMax <= g.UMin || g.VMax <= g.VMin {
		return nil, true
	}
	tr := newSSITracer(base, other, g)
	var curves [][]math.Point3
	for _, seed := range ssiSeeds(base, other, g) {
		if len(curves) >= ssiMaxCurves || tr.exhausted() {
			break
		}
		curves = tr.growFromSeed(curves, seed)
	}
	return curves, !tr.exhausted()
}

// growFromSeed extends the curve set with whatever one seed yields: a tangential CONTACT point, a
// newly marched curve, or nothing when the seed is a duplicate, fails to correct, or the trace has
// spent its corrector budget.
//
// A seed where the surfaces are close AND their normals are (anti)parallel is a tangential contact,
// not a transversal curve: it is refined to the contact point rather than run through the
// (ill-conditioned, singular) curve corrector there, which would otherwise emit a spurious
// near-tangency point and suppress the true one.
func (t ssiTracer) growFromSeed(curves [][]math.Point3, seed math.Point3) [][]math.Point3 {
	if nearTangency(t.base, t.other, seed, t.step) {
		if contact, isT := refineTangency(t.base, t.other, seed, t.tol); isT && !nearAnyCurve(curves, contact, t.step) {
			return append(curves, []math.Point3{contact})
		}
		return curves
	}
	if !t.charge() {
		return curves
	}
	pc, nb, no, ok := correctToBothSurfaces(t.base, t.other, seed, t.tol)
	if !ok || nearAnyCurve(curves, pc, t.step*ssiDedupSteps) {
		return curves // failed to correct, or lands on an already-traced curve (within a march step)
	}
	return append(curves, t.marchCurve(pc, nb, no))
}

// marchCurve traces one curve through pc by stepping forward then backward along the curve tangent,
// correcting each predicted point back onto both surfaces, until the curve closes into a loop or leaves
// the base's parameter window in both directions.
func (tr ssiTracer) marchCurve(pc math.Point3, nb, no math.Vector3) []math.Point3 {
	fwd, closed := tr.marchOneWay(pc, nb, no, true)
	if closed {
		return append([]math.Point3{pc}, fwd...)
	}
	bwd, _ := tr.marchOneWay(pc, nb, no, false)
	out := make([]math.Point3, 0, len(bwd)+1+len(fwd))
	for _, b := range slices.Backward(bwd) {
		out = append(out, b)
	}
	out = append(out, pc)
	return append(out, fwd...)
}

// marchOneWay steps from start in one direction (forward=along +nb×no, else −) and returns the points
// reached plus whether it closed back onto start (a loop). Each step is curvature-controlled and
// gate-checked (#1598): predict h along the tangent, correct onto the curve, then accept only if the
// corrector stayed near the prediction, made forward progress, and turned the tangent within the
// same-branch bound — otherwise reject and halve. The step then follows h = 2√(2ε/κ) from the discrete
// curvature of the last three accepted points, rate-limited so the controller cannot thrash.
func (tr ssiTracer) marchOneWay(start math.Point3, nb, no math.Vector3, forward bool) ([]math.Point3, bool) {
	dir, ok := curveTangent(nb, no, forward)
	if !ok {
		return nil, false
	}
	sw := ssiSweep{tr: tr, p: start, start: start, dir: dir, h: tr.step}
	for i := 0; i < ssiMaxStepsPerCurve && sw.arc < tr.arcCap; i++ {
		pc, tc, verdict := sw.attemptStep()
		switch verdict {
		case ssiStepRejected:
			if !sw.halve() {
				return sw.pts, false // stuck at the h floor: honest open curve (#1598)
			}
		case ssiStepExited:
			return append(sw.pts, pc), false // exited the base window: keep the boundary point
		case ssiStepClosed:
			return append(sw.pts, sw.start), true
		case ssiStepAccepted:
			sw.accept(pc, tc)
		}
	}
	return sw.pts, false
}

// newSSITracer derives the model-relative controller context from the base patch extent.
func newSSITracer(base, other Surface, g SurfaceGrid) ssiTracer {
	h0 := ssiStep(base, g)
	extent := h0 / ssiStepFraction
	return ssiTracer{
		base: base, other: other, g: g,
		step: h0, tol: ssiTolerance(base, g),
		eps:  ssiChordFraction * extent,
		hMin: ssiMinStepFraction * extent, hMax: ssiMaxStepFraction * extent,
		arcCap: ssiMaxArcExtents * extent,
		spent:  new(int),
	}
}

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

// correctToBothSurfaces pulls p onto the intersection of base and other by repeatedly stepping it toward
// both surfaces (the standard SSI corrector). Returns the corrected point, the unit normals there, and
// whether it converged below tol. Where the normals are well separated each step is the tangent-plane
// intersection; through a tangency/pinch neighbourhood — where that step is ill-conditioned — it falls
// back to a damped descent step so the curve is traced THROUGH the singularity rather than dropped at it
// (correctorStep, #1404). ok is false only when no zero is reached within ssiCorrectIters (a genuine
// mid-air stall: the predicted point has no nearby crossing, e.g. a march walking off a NURBS patch).
func correctToBothSurfaces(base, other Surface, p math.Point3, tol float64) (math.Point3, math.Vector3, math.Vector3, bool) {
	var nb, no math.Vector3
	for range ssiCorrectIters {
		ub, vb, db := ProjectPointToSurface(base, p)
		uo, vo, do := ProjectPointToSurface(other, p)
		pb, po := base.PointAt(ub, vb), other.PointAt(uo, vo)
		nb, no = base.NormalAt(ub, vb), other.NormalAt(uo, vo)
		sb := float64(nb.Dot(pb.VectorTo(p))) // signed gap to base's tangent plane
		so := float64(no.Dot(po.VectorTo(p)))
		// Converge on the actual residual distances, NOT the tangent-plane gaps: where a (clamped)
		// projection lands on a NURBS patch boundary the gap reads ~0 while the point is still off the
		// surface, so a march that walks off the patch must fail here rather than accept a false zero.
		if db < tol && do < tol {
			return p, nb, no, true
		}
		p = p.TranslateBy(correctorStep(nb, no, sb, so))
	}
	return p, nb, no, false
}

// correctorStep returns one corrector displacement that drives the two signed tangent-plane gaps (sb, so)
// toward zero. Where the normals are well separated it is the exact tangent-plane intersection step (lands
// p on BOTH tangent planes). Where they are near-parallel — a tangency or pinch, the configuration #1404
// must survive — that 2×2 system is ill-conditioned and blows up, so it falls back to a damped
// steepest-descent step on ½(sb²+so²), whose gradient is sb·nb+so·no: unconditionally stable, still
// monotone, and so converges onto a shallow transversal crossing (a near-pinch) and onto a true contact
// (a pinch) instead of giving up and dropping the curve there.
func correctorStep(nb, no math.Vector3, sb, so float64) math.Vector3 {
	if a, b, ok := tangentPlaneSolve(nb, no, sb, so); ok {
		return nb.Scale(math.Scalar(a)).Add(no.Scale(math.Scalar(b)))
	}
	gradient := nb.Scale(math.Scalar(sb)).Add(no.Scale(math.Scalar(so)))
	return gradient.Scale(-ssiDescentGain)
}

// tangentPlaneSolve finds the step δ = a·nb + b·no that lands p on BOTH tangent planes:
// nb·δ = −sb, no·δ = −so. With c = nb·no this is a 2×2 system; ok is false when |1−c²| ≈ 0
// (parallel normals — the planes are coincident/parallel and the intersection line is undefined).
func tangentPlaneSolve(nb, no math.Vector3, sb, so float64) (a, b float64, ok bool) {
	c := float64(nb.Dot(no))
	den := 1 - c*c
	if stdmath.Abs(den) < 1-ssiTangencyCos {
		return 0, 0, false
	}
	b = (sb*c - so) / den
	a = -sb - b*c
	return a, b, true
}

// nearTangency reports whether the surfaces are close at p (within a march step) AND their normals are
// (anti)parallel there — the signature of a tangential contact, distinct from a transversal crossing
// (where the surfaces cross at a non-zero angle, so their normals are not parallel).
func nearTangency(base, other Surface, p math.Point3, step float64) bool {
	ub, vb, _ := ProjectPointToSurface(base, p)
	uo, vo, do := ProjectPointToSurface(other, p)
	// Gate generously (a seed can sit several steps from the contact) — the strict normal-parallelism
	// test (ssiTangencyCos = 1 − 1e-10) is the real discriminator: only a genuine tangency passes it,
	// a transversal crossing (even a shallow one) does not.
	if do > ssiTangencyGapSteps*step {
		return false
	}
	c := stdmath.Abs(float64(base.NormalAt(ub, vb).Dot(other.NormalAt(uo, vo))))
	return c > ssiTangencyCos
}

// refineTangency pinpoints a tangential contact near p by iterating toward the midpoint of the two
// surfaces' closest points: where the surfaces touch, both feet converge to the contact. It returns
// the contact point and whether it is a GENUINE tangency — the surfaces meet within tol there and
// their normals are (anti)parallel (so the corrector's stall was a real touch, not a mid-air stall).
func refineTangency(base, other Surface, p math.Point3, tol float64) (math.Point3, bool) {
	for range ssiCorrectIters {
		ub, vb, _ := ProjectPointToSurface(base, p)
		uo, vo, _ := ProjectPointToSurface(other, p)
		pb, po := base.PointAt(ub, vb), other.PointAt(uo, vo)
		gap := float64(pb.DistanceTo(po))
		p = pb.TranslateBy(pb.VectorTo(po).Scale(0.5)) // midpoint of the closest approach
		if gap < tol {
			c := stdmath.Abs(float64(base.NormalAt(ub, vb).Dot(other.NormalAt(uo, vo))))
			return p, c > ssiTangencyCos
		}
	}
	return p, false
}

// inWindow reports whether pc's base parameters lie inside the grid window (with a one-step margin so a
// point exactly on the boundary still counts).
func inWindow(base Surface, pc math.Point3, g SurfaceGrid) bool {
	u, v, _ := ProjectPointToSurface(base, pc)
	mu := (g.UMax - g.UMin) * 1e-9 // tol:parametric — fraction of the u-window
	mv := (g.VMax - g.VMin) * 1e-9 // tol:parametric — fraction of the v-window
	return u >= g.UMin-mu && u <= g.UMax+mu && v >= g.VMin-mv && v <= g.VMax+mv
}

// nearAnyCurve reports whether p is within tol of an already-traced curve, measured to the polyline
// SEGMENTS (a single-point tangency marker degenerates to its point). Vertex distance conflated a
// duplicate seed (on the same curve, within chord sag of its polyline) with a distinct neighbouring
// loop as soon as the loops sat closer together than the dedup radius — e.g. the two near-tangent
// circles where a sphere barely clears a cylinder (#1597).
func nearAnyCurve(curves [][]math.Point3, p math.Point3, tol float64) bool {
	for _, c := range curves {
		if len(c) == 1 && float64(p.DistanceTo(c[0])) < tol {
			return true
		}
		for i := 0; i+1 < len(c); i++ {
			if DistancePointToSegment(LineSegment{StartPoint: c[i], EndPoint: c[i+1]}, p) < tol {
				return true
			}
		}
	}
	return false
}

// ssiTolerance is the model-relative on-curve tolerance: 1e-7 of the base's 3D extent (the stated
// acceptance tolerance, reachable by the NURBS Gauss–Newton projection — a 1e-9 target is not).
func ssiTolerance(base Surface, g SurfaceGrid) float64 {
	return ssiToleranceFraction * ssiExtent(base, g)
}

// ssiStep is the nominal march step: a small fraction of the base's 3D extent, so a full curve is a
// few thousand points at most while fine enough to resolve curvature.
func ssiStep(base Surface, g SurfaceGrid) float64 {
	return ssiStepFraction * ssiExtent(base, g)
}

// ssiExtentSamples is the per-axis lattice for the extent estimate. Five samples put nodes at the
// quarter points of a full-period window, so a closed direction contributes the four cardinal points
// of every circular section — its true girth — to the bounding box (#1597).
const ssiExtentSamples = 5

// ssiExtent estimates the base patch's 3D size over the grid window — the diagonal of the bounding
// box of a sampled parameter lattice — used to scale the tolerance and step. The previous
// corner-to-corner estimate collapsed wherever the window is periodic: on a torus both corners map to
// the SAME 3D point (measured 1.5e-14 for R=50/r=10, true ≈171), and on a cylinder they differ by the
// axial height only, missing the girth entirely — which drove step and tolerance toward 0 and pushed
// every trace onto the marching-squares fallback silently (#1597). OCCT sizes its walking step from
// surface bounding boxes the same way (IntWalk_PWalking via Bnd_Box/adaptor resolution).
func ssiExtent(base Surface, g SurfaceGrid) float64 {
	d := float64(sampledPatchBox(base, g).Diagonal().Length())
	if d <= 0 {
		return 1
	}
	return d
}

// sampledPatchBox is the axis-aligned box of base evaluated on an ssiExtentSamples² lattice over the
// grid window (corners included).
func sampledPatchBox(base Surface, g SurfaceGrid) math.Box {
	n := ssiExtentSamples - 1
	pts := make([]math.Point3, 0, ssiExtentSamples*ssiExtentSamples)
	for i := 0; i <= n; i++ {
		u := g.UMin + (g.UMax-g.UMin)*float64(i)/float64(n)
		for j := 0; j <= n; j++ {
			v := g.VMin + (g.VMax-g.VMin)*float64(j)/float64(n)
			pts = append(pts, base.PointAt(u, v))
		}
	}
	return math.BoxFromPoints(pts...)
}
