// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops/internal/mesh"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// PartFeatureOperation is the boolean a feature applies to existing material — the
// PartFeatureOperationEnum.
type PartFeatureOperation uint8

const (
	// Join adds material (A ∪ B).
	Join PartFeatureOperation = iota
	// Cut removes the tool from the target (A − B).
	Cut
	// Intersect keeps only the common material (A ∩ B).
	Intersect
	// NewBody creates a separate body rather than combining.
	NewBody
	// Surface is NOT a boolean: it is Inventor's kSurfaceOperation. The feature builds an
	// open sheet (surface) body — walls only, uncapped, non-solid — added alongside existing
	// bodies with no boolean. Features must short-circuit on Surface before reaching Boolean;
	// it never maps to a B-rep boolean (see toBrepOp). #1858.
	Surface
)

// String returns a stable name for diagnostics.
func (op PartFeatureOperation) String() string {
	switch op {
	case Join:
		return "join"
	case Cut:
		return "cut"
	case Intersect:
		return "intersect"
	case Surface:
		return "surface"
	default:
		return "new-body"
	}
}

// Boolean combines target and tool under op. The non-overlapping topological cases —
// disjoint bodies and one fully containing the other — produce valid results without
// splitting faces. General intersecting booleans (face-face intersection, splitting and
// re-stitching) go through booleanGeneral: the exact planar B-rep boolean for
// planar-faceted operands, falling back to triangle-soup CSG when an operand has a curved
// face. Result-face lineage flows from the operands, so reference keys stay rebindable.
func Boolean(op PartFeatureOperation, target, tool *topo.Body) (*topo.Body, error) {
	return BooleanWithDiagnostics(op, target, tool, nil)
}

// CodeBooleanCSGFallback marks a boolean that abandoned the exact analytic/planar B-rep path for
// triangle-soup CSG — a tracked defect: the result keeps no analytic surfaces and can hide a
// volume-preserving topology error the volume guard misses (Oblikovati#1407).
const CodeBooleanCSGFallback diag.Code = "boolean.csg-fallback"

// CodeBooleanAnalyticFaceted marks a boolean whose analytic curved operand(s) were re-faceted
// into a planar B-rep because no exact curved path applied (#1601). Faceting is permanent — the
// analytic surface is unrecoverable and every downstream feature operates on facets — so the
// degradation must ride with the feature that caused it, not vanish.
const CodeBooleanAnalyticFaceted diag.Code = "boolean.analytic-faceted"

// BooleanWithDiagnostics is [Boolean] with a diagnostic [diag.Recorder] (pass nil to discard). Whenever
// the operation falls back from the exact analytic/planar path to triangle-soup CSG it records a Defect
// diagnostic naming the operation and operands, so callers and tests can SEE and count the fallback
// instead of silently shipping a faceted mesh (Oblikovati#1407).
func BooleanWithDiagnostics(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, error) {
	lin := topo.NewLineage(topo.Tok("boolean", op.String(), 0))
	if op == NewBody {
		return tool, nil
	}
	rel := classify(target, tool)
	switch op {
	case Join:
		return join(lin, target, tool, rel, rec)
	case Cut:
		return cut(lin, target, tool, rel, rec)
	default: // Intersect
		return intersect(lin, target, tool, rel, rec)
	}
}

// join combines target and tool under A ∪ B. When one body strictly contains the other (classify
// guarantees no boundary crossing — see strictlyContains, #1315), the union IS the outer body, so the
// inner shell is discarded rather than kept as a floating interior wall — the old MergeBodies path
// concatenated both shells and double-counted the volume (#1316). Disjoint bodies merge into one
// multi-lump body (each a separate valid shell); intersecting bodies go to the face-splitting boolean.
func join(lin topo.Lineage, target, tool *topo.Body, rel relation, rec *diag.Recorder) (*topo.Body, error) {
	switch rel {
	case targetContainsTool:
		return target, nil // tool lies inside target → union is target alone
	case toolContainsTarget:
		return tool, nil
	case intersecting:
		return booleanGeneral(Join, target, tool, lin, rec)
	default: // disjoint: two separate lumps form a valid multi-shell body
		return topo.MergeBodies(lin, true, target, tool), nil
	}
}

// booleanGeneral runs the general (face-splitting) boolean: the planar B-rep boolean first
// — it is sound under chaining and yields a low-face-count solid — falling back to the
// triangle-soup BSP CSG only when an operand has a non-planar face the B-rep path can't take
// (a cylinder, cone, etc.). A nil B-rep result is a (valid) empty body.
// booleanGeneral runs the general boolean and, as a last resort, rescues a torn or
// invalid result with the exact mesh-arrangement engine (Oblikovati#2084). The
// analytic/planar/CSG path (booleanGeneralExact) runs first and its result is kept
// whenever it is a valid closed solid — so every case those paths already handle is
// untouched and their analytic B-rep (curved faces, edge provenance) is preserved.
// Only when they leave an INVALID or torn result — the near-tangent grazing seam the
// planar imprint cannot stitch — does this fall to the mesh-arrangement engine,
// adopting it only if IT is a valid solid. That result is FACETED (#2153), so it is a
// rescue for a case that otherwise ships broken, never a preference.
func booleanGeneral(op PartFeatureOperation, target, tool *topo.Body, lin topo.Lineage, rec *diag.Recorder) (*topo.Body, error) {
	body, err := booleanGeneralExact(op, target, tool, lin, rec)
	if err == nil && body != nil && Validate(body).ValidSolid() {
		return body, nil // reconstruction now lives inside curvedExactBoolean (ADR-0056 L5), so a
		// valid primary here is already the analytic reconstruction where one applied
	}
	if mesh := meshArrangementFallback(op, target, tool, rec); mesh != nil {
		return mesh, nil
	}
	return body, err
}

// CodeBooleanMeshArrangementFallback marks a boolean the analytic/planar/CSG path left
// invalid and the exact mesh-arrangement engine (ADR-0052) recovered. The recovered
// solid is exact-volume and watertight but FACETED (#2153) — its curved faces are
// planar approximations — so this is a tracked degradation like the CSG fallback: a
// valid faceted solid in place of a torn one, not the preferred analytic result.
const CodeBooleanMeshArrangementFallback diag.Code = "boolean.mesh-arrangement-fallback"

// meshArrangementFallback rescues an invalid boolean result with the exact
// mesh-arrangement engine, returning a valid faceted solid or nil when it does not
// apply or does not recover. It is gated on the same modest operand size as the CSG
// fallback: the engine tessellates both operands and runs an exact arrangement,
// expensive on a large body and rarely the way to recover one.
func meshArrangementFallback(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) *topo.Body {
	mop, ok := toMeshboolOp(op)
	if !ok || len(target.Faces())+len(tool.Faces()) > csgFallbackFaceLimit {
		return nil
	}
	body := booleanViaMeshbool(target, tool, mop, DefaultQuality(), "boolean")
	// Require a NON-EMPTY valid solid. An empty body vacuously passes Validate().ValidSolid()
	// (no faces to violate manifoldness), but the fallback only reaches here because the
	// primary path left a non-empty torn result — so real geometry is expected. An empty
	// mesh result is not the tear it exists to rescue, so decline rather than adopt it.
	if len(body.Faces()) == 0 || !Validate(body).ValidSolid() {
		return nil
	}
	rec.Recordf(CodeBooleanMeshArrangementFallback, diag.Defect,
		"%s: analytic/planar/CSG boolean left an invalid result; recovered with the exact mesh-arrangement engine (faceted, #2153)", op)
	return body
}

// booleanGeneralExact runs the analytic/planar/CSG boolean (no mesh-arrangement
// fallback): the exact analytic curved paths first, then the planar B-rep boolean,
// falling to triangle-soup CSG when an operand has a non-planar face the B-rep path
// cannot take. booleanGeneral wraps it with the mesh-arrangement rescue.
func booleanGeneralExact(op PartFeatureOperation, target, tool *topo.Body, lin topo.Lineage, rec *diag.Recorder) (*topo.Body, error) {
	if body, ok := curvedExactGuarded(op, target, tool, rec); ok {
		return body, nil
	}
	bop, ok := toBrepOp(op)
	if !ok {
		return booleanCSG(op, target, tool, lin, rec)
	}
	body, err := brep.BooleanDiag(bop, target, tool, rec)
	if err != nil {
		return booleanCSG(op, target, tool, lin, rec) // non-planar operand → triangle CSG
	}
	if body == nil {
		return topo.MergeBodies(lin, true), nil
	}
	// Provenance (ADR-0043): the planar boolean names intersection edges by their crossing faces but
	// falls surviving ORIGINAL boundaries back to a build-order ordinal (brep:edge#N). Restore those
	// boundaries' original identity so a reference to an edge the operation passed through whole — a
	// box edge a chamfer/combine/hole left untouched — survives an upstream edit.
	body.InheritOriginalEdges(append(append([]*topo.Edge(nil), target.Edges()...), tool.Edges()...))
	// Guard: the exact planar boolean can still produce an INVALID (non-manifold) result on a
	// degenerate arrangement — notably a coplanar flush face combined with an oblique partial
	// penetration (the "V2" oblique-into-corner-with-flush-bottom case), where the coplanar
	// seam doesn't stitch. Validate is cheap (edge-use counts, no tessellation); only when it
	// fails do we fall back to the robust triangle CSG, and only adopt that if IT is valid —
	// so every case the planar path already handles is untouched. The fallback is gated on a
	// modest operand size: triangle CSG on a large body is expensive and rarely recovers it,
	// so above the limit we keep the (fast) planar result rather than pay a big CSG attempt.
	if shouldFallbackBoolean(op, target, tool, body) && len(target.Faces())+len(tool.Faces()) <= csgFallbackFaceLimit {
		// Adopt the CSG fallback only when it is a genuine closed manifold solid — NOT merely
		// Validate().Valid, which does not require Closed. T-junction removal now always closes
		// the cage (#1336), so this is belt-and-braces: never replace a sound planar result with
		// an open one.
		if csg, cerr := booleanCSG(op, target, tool, lin, rec); cerr == nil && csg != nil && Validate(csg).ValidSolid() {
			return csg, nil
		}
	}
	return body, nil
}

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
	if _, ok := AnalyticGeometryProperties(target); !ok {
		return false
	}
	_, ok := AnalyticGeometryProperties(tool)
	return ok
}

// CurvedBoolean attempts the exact analytic curved boolean and reports whether it applied. It is SAFE to
// call on any operands — each path declines (ok=false) when it does not handle (op, target, tool), and none
// hangs (unlike the planar B-rep boolean, which loops on a full periodic curved face). The model layer uses
// it to combine a still-analytic primitive (a revolved torus, an extruded cylinder) by its curved faces
// before falling back to faceting the operands for the planar path (#129).
func CurvedBoolean(op PartFeatureOperation, target, tool *topo.Body) (*topo.Body, bool) {
	return curvedExactBoolean(op, target, tool, nil)
}

// CurvedBooleanWithDiagnostics is [CurvedBoolean] with a diagnostic recorder (nil to discard):
// the exact paths record imprint-quality diagnostics (#1404) and their internal fallbacks, so a
// feature-level caller carries the kernel's quality signal instead of dropping it (#1601).
func CurvedBooleanWithDiagnostics(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	return curvedExactBoolean(op, target, tool, rec)
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
// through BodyGeometryProperties, which integrates the ANALYTIC B-rep (M48/C3 #3448), so the bracket
// no longer compares three tessellations whose chord deficits it had to be widened to absorb. The
// quality is shared and only reaches a body the analytic path declines, where having all three
// measured the same way still lets their deficits partly cancel.
func boolVolumes(target, tool, body *topo.Body) (targetVol, toolVol, bodyVol float64) {
	q := DefaultQuality()
	return BodyGeometryProperties(target, q).Volume,
		BodyGeometryProperties(tool, q).Volume,
		BodyGeometryProperties(body, q).Volume
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

// csgFallbackFaceLimit bounds the operand size for the invalid-result CSG fallback (see
// booleanGeneral): small degenerate cases (the V2 flush-bottom oblique penetration) recover
// cheaply, while a huge body's CSG attempt is costly and seldom valid.
const csgFallbackFaceLimit = 256

// toBrepOp maps the feature-level operation to the B-rep boolean operation (NewBody and
// Surface have no B-rep analogue — both are handled before this point).
func toBrepOp(op PartFeatureOperation) (brep.Op, bool) {
	switch op {
	case Join:
		return brep.Union, true
	case Cut:
		return brep.Difference, true
	case Intersect:
		return brep.Intersection, true
	default:
		return 0, false
	}
}

// booleanCSG runs the general intersecting boolean via the BSP-tree CSG over the
// operands' triangles, welding the kept triangles back into a watertight solid
// (PBI-171). An empty result (e.g. an intersection that turns out disjoint) yields an
// empty body, which the caller drops.
//
// This is the LAST-RESORT fallback, not the primary curved path (M2 §M2, #1336): booleanGeneral reaches
// it only after every exact analytic path (curvedExactPaths) and the exact planar B-rep boolean have
// declined. The supported curved booleans keep their analytic surfaces and never land here — the
// TestCurvedBooleansStayExact guard pins that. CSG remains for the unsupported long tail (arbitrary
// freeform/NURBS overlaps), where a faceted-but-watertight result still beats failing.
func booleanCSG(op PartFeatureOperation, target, tool *topo.Body, lin topo.Lineage, rec *diag.Recorder) (*topo.Body, error) {
	rec.Recordf(CodeBooleanCSGFallback, diag.Defect,
		"%s fell back to triangle-soup CSG (target %d faces, tool %d faces): no exact analytic/planar path",
		op, len(target.Faces()), len(tool.Faces()))
	a, b := bodyTriangles(target), bodyTriangles(tool)
	// One model-relative on-plane tolerance for the BSP, scaled to the larger operand
	// (ADR-0042) so a sub-µm boolean classifies coplanarity correctly.
	planeTol := ResolutionForBodies(target, tool).Plane()
	var result []mesh.Tri
	switch op {
	case Join:
		result = csgUnion(a, b, planeTol)
	case Cut:
		result = csgSubtract(a, b, planeTol)
	default:
		result = csgIntersect(a, b, planeTol)
	}
	if body := trianglesToBody(result, "boolean-"+op.String()); body != nil {
		return body, nil
	}
	return topo.MergeBodies(lin, true), nil
}

func cut(lin topo.Lineage, target, tool *topo.Body, rel relation, rec *diag.Recorder) (*topo.Body, error) {
	switch rel {
	case disjoint:
		return target, nil // tool removes nothing
	case toolContainsTarget:
		return topo.MergeBodies(lin, true), nil // target fully removed → empty
	default: // targetContainsTool (a void/cavity) and intersecting need face splitting
		return booleanGeneral(Cut, target, tool, lin, rec)
	}
}

func intersect(lin topo.Lineage, target, tool *topo.Body, rel relation, rec *diag.Recorder) (*topo.Body, error) {
	switch rel {
	case disjoint:
		return topo.MergeBodies(lin, true), nil // nothing in common → empty
	case targetContainsTool:
		return tool, nil
	case toolContainsTarget:
		return target, nil
	default:
		return booleanGeneral(Intersect, target, tool, lin, rec)
	}
}

// relation classifies how two bodies sit relative to each other.
type relation uint8

const (
	disjoint relation = iota
	targetContainsTool
	toolContainsTarget
	intersecting
)

// classify determines the spatial relationship from bounding boxes and containment. It recognizes
// the cases that combine without face splitting (disjoint, or one body strictly containing the
// other); anything else is reported as intersecting and routed to the general face-splitting boolean.
func classify(target, tool *topo.Body) relation {
	if !target.RangeBox().Intersects(tool.RangeBox()) {
		return disjoint
	}
	if strictlyContains(target, tool) {
		return targetContainsTool
	}
	if strictlyContains(tool, target) {
		return toolContainsTarget
	}
	return intersecting
}

// strictlyContains reports whether outer fully contains inner: every vertex of inner lies inside
// outer AND the two boundaries do not cross. The boundary-crossing condition is essential — vertex
// containment alone is unsound for non-convex bodies, where inner's surface can pass back out through
// outer between inner's vertices (#1315). Without it such a pair skips face splitting and returns a
// silently wrong solid. The crossing test runs only after the (cheap) vertex test passes.
func strictlyContains(outer, inner *topo.Body) bool {
	return allVerticesInside(inner, outer) && !boundariesCross(outer, inner)
}

// allVerticesInside reports whether every vertex of inner lies strictly within outer, analytically
// (M48/C3 #3422). It prepares outer ONCE as a brep.InsideQuery and reuses that for every vertex query
// — the analytic analog of tessellating outer once and reusing the mesh (#1317) — so classification
// no longer reads a tessellation. The prepared query dispatches by representation exactly as
// PointInsideBody does (all-planar → generalized winding, curved → nearest-crossing ray), so the
// vertex verdicts match the retired mesh oracle on every consistently-oriented body.
func allVerticesInside(inner, outer *topo.Body) bool {
	verts := inner.Vertices()
	if len(verts) == 0 {
		return false
	}
	inside := brep.NewInsideQuery(outer)
	for _, v := range verts {
		if !inside.Inside(v.Point()) {
			return false
		}
	}
	return true
}

// PointInsideBody reports whether p is strictly inside a solid body, analytically (M48/C3 #3426/#3427):
// it delegates to brep.PointInside, which dispatches by representation — an all-planar body to the
// solid-angle-flux generalized winding number (Jacobson) and a curved body to the nearest-crossing ray
// classifier (the classical solid classifier), neither of which reads a tessellation. It replaced the
// mesh winding oracle once that classifier gained near-surface crispness (the nearest-crossing side
// test is decisive a hair off a face, where the smooth winding reads ≈½) and import-robust orientation
// (orientFaceSigns derives each face's outward sign from its loop geometry, not the stored Reversed flag).
func PointInsideBody(b *topo.Body, p math.Point3) bool {
	return brep.PointInside(b, p)
}
