// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"errors"
	"fmt"

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
	if err != nil {
		return nil, err
	}
	if body != nil && !Validate(body).ValidSolid() {
		return nil, unmodelledBoolean(op, target, tool, errInvalidExactResult)
	}
	return body, nil
}

// errInvalidExactResult is the cause an exact result that fails Validate declines with. It used to be
// the door to the mesh-arrangement rescue, which returned a watertight but FACETED body in its place
// (ADR-0061 stage 6 closed that door: a body that fails its own post-condition is an error, never a
// return value, and no engine stands behind it any more).
var errInvalidExactResult = errors.New("the exact result is not a valid closed solid")

// booleanGeneralExact runs the exact boolean: the analytic curved paths first, then the exact per-face
// B-rep boolean. A configuration neither models is REFUSED by name (ADR-0061 stage 6) — there is no
// triangle-soup CSG and no mesh-arrangement rescue behind it any more, so a caller that reaches this
// error quarantines the feature instead of shipping a faceted body that looks solid.
func booleanGeneralExact(op PartFeatureOperation, target, tool *topo.Body, lin topo.Lineage, rec *diag.Recorder) (*topo.Body, error) {
	if body, ok := curvedExactGuarded(op, target, tool, rec); ok {
		return body, nil
	}
	bop, ok := toBrepOp(op)
	if !ok {
		return nil, unmodelledBoolean(op, target, tool, errNotABoolean)
	}
	body, err := brep.BooleanDiag(bop, target, tool, rec)
	if err != nil {
		return nil, unmodelledBoolean(op, target, tool, err)
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
	if shouldFallbackBoolean(op, target, tool, body) {
		return nil, unmodelledBoolean(op, target, tool, errFailedAcceptance)
	}
	return body, nil
}

// errNotABoolean is the cause for an op that is no boolean at all (NewBody, Surface); features
// short-circuit on both before reaching here, so it names a caller error rather than a geometry one.
var errNotABoolean = errors.New("the operation is not a boolean")

// errFailedAcceptance is the cause an exact result that misses its own acceptance gate declines with:
// invalid, or a face the operation should not have kept, or a volume outside the Requicha bracket.
var errFailedAcceptance = errors.New("the exact result failed its own acceptance gate")

// unmodelledBoolean is the boolean's NAMED decline: no exact path models this pair. It carries the
// operation, both operands' face counts and the underlying cause, so a sick feature says what was
// refused instead of reporting a bare failure (ADR-0061 stage 6).
func unmodelledBoolean(op PartFeatureOperation, target, tool *topo.Body, cause error) error {
	return fmt.Errorf("%w: %s of a %d-face target and a %d-face tool: %w",
		ErrUnmodelledBoolean, op, len(target.Faces()), len(tool.Faces()), cause)
}

// ErrUnmodelledBoolean is the boolean's refusal: the operands meet in a configuration no exact path
// models. It is a REFUSAL at classification, never a wrong result and never a faceted stand-in — the
// ground rule the CSG and mesh-arrangement fallbacks broke for as long as they stood behind it.
var ErrUnmodelledBoolean = errors.New("boolean: no exact path models this contact configuration")
