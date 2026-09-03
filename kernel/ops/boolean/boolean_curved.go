// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
)

// The CURVED half of the boolean: the analytic recognizers and the guards that decide whether to
// trust one (split out of boolean.go for #2215).
//
// curvedExactPaths is an ordered first-fit list of 26 recognizers whose try-order is load-bearing.
// The ground rules forbid that shape — "dispatch is a classification that selects exactly one
// path" — and #3397 replaces it; ADR-0056 "What this deletes" names the recognizers a completed
// reconstruction retires. It is gathered in one file so that work has one place to land.
//
// The guards below it are why a wrong analytic result does not ship: a recognizer's body must be a
// valid solid AND land inside the Requicha volume bracket, or the operation falls through.

// curvedExactBoolean tries each exact analytic curved-boolean path in turn, returning the first that
// applies (M2, ADR-0027 §M2). Each keeps the operands' exact analytic surfaces instead of the triangle-soup
// CSG fallback; one that does not apply returns ok=false so booleanGeneral moves on. Every path belongs to
// one of three boolean KINDs (ADR-0045), distinguished by the DIMENSIONALITY of the operands' contact:
//
//	TRANSVERSAL CROSSING (contact = a 1-D curve; the general SSI → (u,v)-arrangement → classify → stitch
//	pipeline, or a curved solid trimmed by a convex tool's planar half-spaces, #1403/#1476):
//	  - a curved solid ∩/− a convex planar tool or prism (a box; cylinder − box) → composed half-space cuts (#1334);
//	  - two crossing cylinders ∩/−/∪ (rod band + fat-wall lens caps) (#1335);
//	  - a cone crossing a cylinder, and a cone crossing a fatter cone, ∩/−/∪ (#1335);
//	  - two EQUAL-radius perpendicular cylinders ∩/−/∪ (the Steinmetz bicylinder, imprint split at its pinches) (#1403);
//	  - a thin rod ending inside a fatter solid ∩/−/∪ (a partial penetration: plug, blind hole, one-sided stub) (#1335);
//	  - a rod COAXIAL with a ball, ending inside it ∩/−/∪ (the ball stud, its blind spherical bore, its plug) —
//	    transversal, but the contact is one PLANAR circle and a sphere is not a rim-bounded band, so the split is
//	    by construction and no arrangement runs; OCCT special-cases the same pair in closed form (#2036).
//
//	CURVED-ON-PLANAR (contact = one closed conic STRICTLY INSIDE a planar face, added as an inner loop; no
//	SSI arrangement — the periodic (u,v) machinery does not apply to a flat bounded face, ADR-0045):
//	  - drilling a clean through-hole in a slab with a straight cylinder (box − cylinder) (#1336);
//	  - the union of a cylinder seated flush on a planar face (a boss/spigot) → seat-face hole + wall + cap (#1336).
//
//	DEGENERATE OVERLAP (contact = a 2-D region of COINCIDENT surfaces; a simplification, not an SSI handler,
//	ADR-0045):
//	  - the union of two coaxial equal-radius cylinders that overlap/abut → one taller cylinder (#1336).
func curvedExactBoolean(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	for _, exact := range curvedExactPaths {
		if body, ok := exact(op, target, tool, rec); ok {
			return body, true
		}
	}
	// General per-face dispatch (ADR-0058): when the curved faces are clear of the other operand,
	// the exact planar pipeline splits the planar faces and the curved ones pass through whole —
	// an EXACT analytic result where no bespoke recognizer applied, tried before the mesh rescue.
	if body, ok := mixedPassThroughBoolean(op, target, tool, rec); ok {
		return body, true
	}
	// Last resort (ADR-0056 L5): no analytic recognizer applied — rebuild the join from the
	// exact mesh boolean's provenance on the operands' exact surfaces, instead of faceting.
	return reconstructedCurvedBoolean(op, target, tool, rec)
}

// mixedPassThroughBoolean runs brep's per-face-dispatch boolean on MIXED operands (ADR-0058): only
// when a curved face is present (an all-planar pair keeps its own guarded pipeline downstream,
// byte-for-byte), and only adopted as a valid solid — brep's conservative scope gate declines the
// rest (ErrUnsupportedMixedBoolean), falling through to the mesh reconstruction exactly as before.
func mixedPassThroughBoolean(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	if analyticFaceCount(target)+analyticFaceCount(tool) == 0 {
		return nil, false
	}
	bop, ok := toBrepOp(op)
	if !ok {
		return nil, false
	}
	body, err := brep.BooleanDiag(bop, target, tool, rec)
	if err != nil || body == nil || !Validate(body).ValidSolid() {
		return nil, false
	}
	return body, true
}

// CodeBooleanAnalyticVolumeReject marks a curved analytic boolean whose result fell OUTSIDE the
// Requicha two-sided volume bracket (#1601): the recognizer produced a valid body of materially
// wrong volume (kept the wrong lobe, removed too much), so it is rejected and the operation falls
// back to the guarded planar/CSG path instead of shipping the wrong analytic solid. A tracked
// defect — before this, the analytic short-circuit bypassed the volume guard the planar path gets.
const CodeBooleanAnalyticVolumeReject diag.Code = "boolean.analytic-volume-reject"

// curvedVolumeGuardFraction is the volume bracket's tolerance as a fraction of the larger operand's
// volume, for the case where a body's volume the analytic integrator DECLINES and the tessellation
// therefore measures: that number under-measures a curved body by ~0.6% (a cylinder) up to a bounded
// few percent, and the bracket has to absorb the deficit. It no longer applies when the volumes are
// exact (M48/C3 #3445: a 10% window demoted correct analytic results to facets because the MESHER,
// not the boolean, was wrong).
const curvedVolumeGuardFraction = 0.10 // tol:calibrated — bracket margin for a tessellated fallback volume

// curvedGuardBracketOverride, when non-nil, REPLACES the computed volume-bracket tolerance.
// Production leaves it nil; a test sets it to a large negative value so even a correct result reads
// as out of bracket, which drives the reject-and-diagnose path end to end without needing a
// recognizer bug to reproduce. It replaces rather than scales because the analytic bracket's own
// tolerance is a resolution cube — scaling that by −1 leaves a number far too small to force a
// violation, so the hook silently stopped working when the bracket became exact.
var curvedGuardBracketOverride *float64

// curvedExactGuarded returns an exact analytic curved boolean only when a path applies AND the
// result is CERTIFIED: every face passes Requicha's membership rule against the operands
// (certifyBooleanFaces), and the volume lands inside the Requicha two-sided bracket. The analytic
// short-circuit otherwise bypasses the guard the planar path gets, so a recognizer that
// mis-classifies to a valid body of materially wrong shape would ship silently. The per-face gate is
// the proof and the volume bracket the smoke test — a body can hold the right amount of material in
// the wrong place. When either rejects, this records a Defect and declines (ok=false) so
// booleanGeneral falls through to the guarded planar/CSG path. On acceptance it restores
// original-edge identity (ADR-0043) like the planar path.
func curvedExactGuarded(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	body, ok := curvedExactBoolean(op, target, tool, rec)
	if !ok {
		return nil, false
	}
	if !certifyBooleanFaces(op, target, tool, body) {
		rec.Recordf(CodeBooleanAnalyticFaceReject, diag.Defect,
			"curved %s analytic result has a face the operands do not account for: falling back to the guarded path", op)
		return nil, false
	}
	if !Validate(body).ValidSolid() {
		rec.Recordf(CodeBooleanAnalyticInvalid, diag.Defect,
			"curved %s analytic result is not a valid closed solid: falling back to the guarded path", op)
		return nil, false
	}
	tv, wv, bv := boolVolumes(target, tool, body)
	if volumeOutOfBracket(op, tv, wv, bv, curvedGuardTolerance(target, tool, tv, wv)) {
		rec.Recordf(CodeBooleanAnalyticVolumeReject, diag.Defect,
			"curved %s analytic result volume %g outside the Requicha bracket (V(A)=%g V(B)=%g): falling back to the guarded path",
			op, bv, tv, wv)
		return nil, false
	}
	body.InheritOriginalEdges(append(append([]*topo.Edge(nil), target.Edges()...), tool.Edges()...))
	return body, true
}

// CodeBooleanAnalyticInvalid marks a curved analytic boolean whose result is not a valid closed solid.
// Validate is the post-condition of every public kernel operation, and this entry had none: the inner
// paths that DO validate (mixedPassThroughBoolean) covered most of the surface, so a recognizer that
// returned a torn body shipped it to whichever caller did not re-check — which, once the public
// CurvedBoolean entries took this guarded path, is none of them (ADR-0061). A tracked degradation: the
// analytic result is refused and the operation falls to the guarded planar path.
const CodeBooleanAnalyticInvalid diag.Code = "boolean.analytic-invalid"

// CodeBooleanAnalyticFaceReject marks a curved analytic boolean whose result carried a face the
// operands cannot account for under the operation's membership rule — a face on neither operand's
// boundary, or on the side the operation removes. Unlike a volume miss this localizes the fault to
// a face, and it catches a wrong lobe that happens to have the right volume.
const CodeBooleanAnalyticFaceReject diag.Code = "boolean.analytic-face-reject"

// curvedGuardTolerance is the volume bracket's slack: the model-relative resolution cube when both
// operands integrate analytically (the volumes are exact, so nothing wider is justified), widened to
// curvedVolumeGuardFraction of the larger operand when either fell back to the tessellation, whose
// chord deficit the bracket must then absorb.
func curvedGuardTolerance(target, tool *topo.Body, tv, wv float64) float64 {
	if curvedGuardBracketOverride != nil {
		return *curvedGuardBracketOverride
	}
	if analyticVolumesExact(target, tool) {
		return ResolutionForBodies(target, tool).Volume()
	}
	return curvedVolumeGuardFraction * max(tv, wv)
}

// analyticVolumesExact reports whether both operands integrate over their analytic B-rep, so their
// volumes carry no tessellation deficit for the bracket to absorb.
func analyticVolumesExact(target, tool *topo.Body) bool {
	if _, ok := query.AnalyticGeometryProperties(target); !ok {
		return false
	}
	_, ok := query.AnalyticGeometryProperties(tool)
	return ok
}

// CurvedBoolean attempts the exact analytic curved boolean and reports whether it applied. It is SAFE to
// call on any operands — each path declines (ok=false) when it does not handle (op, target, tool), and none
// hangs (unlike the planar B-rep boolean, which loops on a full periodic curved face). The model layer uses
// it to combine a still-analytic primitive (a revolved torus, an extruded cylinder) by its curved faces
// before falling back to faceting the operands for the planar path (#129).
//
// It is the GUARDED entry (curvedExactGuarded), the same one booleanGeneralExact takes: a result must pass
// the per-face membership certificate and the Requicha volume bracket, or this declines. The public entry
// used to call curvedExactBoolean directly, so a recognizer that over-matched to a valid body of materially
// wrong shape shipped to the feature layer uncertified while the identical call inside the kernel was
// certified. Only the feature layer's face-COUNT gate stood between a wrong result and the model — which is
// why widening that gate to a classification (ADR-0061) had to close this seam first: one operation, one
// certification, whoever calls it.
func CurvedBoolean(op PartFeatureOperation, target, tool *topo.Body) (*topo.Body, bool) {
	return curvedExactGuarded(op, target, tool, nil)
}

// CurvedBooleanWithDiagnostics is [CurvedBoolean] with a diagnostic recorder (nil to discard):
// the exact paths record imprint-quality diagnostics (#1404) and their internal fallbacks, so a
// feature-level caller carries the kernel's quality signal instead of dropping it (#1601).
func CurvedBooleanWithDiagnostics(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	return curvedExactGuarded(op, target, tool, rec)
}

// curvedExactPaths is the ordered list of exact analytic curved-boolean paths curvedExactBoolean tries; each
// returns ok=false when it does not apply to (op, target, tool). The recorder carries the SSI imprint's
// closure diagnostics (#1404) up to the boolean's caller; a path that takes no imprint ignores it. The paths
// are grouped by op (the try-order within an op is load-bearing; do not reorder across a pair that two paths
// could both accept) and tagged with their boolean KIND (ADR-0045): [T] transversal crossing, [P] curved-on-
// planar, [D] degenerate overlap.
var curvedExactPaths = []func(PartFeatureOperation, *topo.Body, *topo.Body, *diag.Recorder) (*topo.Body, bool){
	// Intersect — all [T] transversal (curved∩convex-planar half-space, then ruled crossings).
	curvedConvexIntersect, curvedConvexSubtract,
	curvedRuledCrossingIntersect, curvedSteinmetzIntersect,
	curvedPartialIntersect, curvedBallRodIntersect,
	// Cut — [P] the drill through-hole and the edge scallop (curved-on-planar), the rest [T] transversal.
	curvedCylindricalHoleCut, curvedEdgeScallopCut, curvedFlatSubtract, curvedPartialCut, curvedSteinmetzCut, curvedRuledCrossingCut, curvedCapCrossCut, curvedRimCrossCut, curvedTwoCapCrossCut, curvedConeCapCrossCut, curvedPartialRimCut, curvedPartialRimCornerCut, curvedBallRodCut,
	// Join — [D] coaxial (degenerate overlap), [P] boss interior then straddling (curved-on-planar), the rest [T] transversal.
	curvedCoaxialJoin, curvedCylinderBossJoin, curvedPartialBossJoin, curvedPartialJoin, curvedRuledCrossingJoin, curvedSteinmetzJoin, curvedBallRodJoin,
}

// shouldFallbackBoolean decides whether a result must be abandoned for the next path. Validity comes
// first (an invalid body is never a result), then the per-face membership certificate — the proof
// that the faces are the ones this operation keeps — and only then the whole-body volume bracket,
// which is a cheap smoke test for what the per-face gate could not probe (M48/C3 #3446/#3447).
func shouldFallbackBoolean(op PartFeatureOperation, target, tool, body *topo.Body) bool {
	if !Validate(body).ValidSolid() {
		return true
	}
	if !certifyBooleanFaces(op, target, tool, body) {
		return true
	}
	return invalidBooleanVolume(op, target, tool, body)
}

func invalidBooleanVolume(op PartFeatureOperation, target, tool, body *topo.Body) bool {
	// Model-relative volume tolerance (ADR-0042): scales with the operands' size³ so
	// the result-volume sanity check is faithful at any scale, not just ~cm parts. The
	// planar path's arithmetic is exact-plane, so a tight resolution-cube tol is right.
	tv, wv, bv := boolVolumes(target, tool, body)
	return volumeOutOfBracket(op, tv, wv, bv, ResolutionForBodies(target, tool).Volume())
}

// boolVolumes measures the target, tool and result volumes for the acceptance bracket. All three go
// through query.BodyGeometryProperties, which integrates the ANALYTIC B-rep (M48/C3 #3448), so the bracket
// no longer compares three tessellations whose chord deficits it had to be widened to absorb. The
// quality is shared and only reaches a body the analytic path declines, where having all three
// measured the same way still lets their deficits partly cancel.
func boolVolumes(target, tool, body *topo.Body) (targetVol, toolVol, bodyVol float64) {
	q := DefaultQuality()
	return query.BodyGeometryProperties(target, q).Volume,
		query.BodyGeometryProperties(tool, q).Volume,
		query.BodyGeometryProperties(body, q).Volume
}

// volumeOutOfBracket reports whether a result volume falls outside the Requicha two-sided
// bracket for op with tolerance tol (#1601): V(A∪B) ∈ [max(V(A),V(B)), V(A)+V(B)] and
// V(A∖B) ∈ [V(A)−V(B), V(A)]. The old one-sided checks let a join that fabricated material or a
// cut that removed too much ship silently. Intersect keeps only its upper bound V(A∩B) ≤
// min(V(A),V(B)): its Requicha lower bound needs V(A∪B) — a second boolean, too expensive for a
// guard — and is trivially ≥ 0 without it. Shared by the planar guard (a tight model-relative
// tol) and the curved analytic guard (a deficit-dominating relative tol, curvedExactGuarded).
func volumeOutOfBracket(op PartFeatureOperation, targetVol, toolVol, bodyVol, tol float64) bool {
	switch op {
	case Join:
		return bodyVol+tol < max(targetVol, toolVol) || bodyVol > targetVol+toolVol+tol
	case Cut:
		return bodyVol > targetVol+tol || bodyVol+tol < targetVol-toolVol
	case Intersect:
		return bodyVol > min(targetVol, toolVol)+tol
	}
	return false
}
