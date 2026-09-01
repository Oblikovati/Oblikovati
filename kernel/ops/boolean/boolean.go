// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/topo"
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
