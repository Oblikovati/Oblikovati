// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
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
func booleanGeneral(op PartFeatureOperation, target, tool *topo.Body, lin topo.Lineage, rec *diag.Recorder) (*topo.Body, error) {
	if body, ok := curvedExactBoolean(op, target, tool, rec); ok {
		// Provenance (ADR-0043): the curved stitch already named the new surface-intersection curves by
		// their generating face pair (curvedStitch → RelineageByFaceProvenance). Here, like the planar
		// path, restore the identity of original boundaries the curved cut passed through WHOLE — a
		// survivor keeps its own key, not a face-pair name. An exact analytic result (M2 #1334/#1335).
		body.InheritOriginalEdges(append(append([]*topo.Edge(nil), target.Edges()...), tool.Edges()...))
		return body, nil
	}
	bop, ok := toBrepOp(op)
	if !ok {
		return booleanCSG(op, target, tool, lin, rec)
	}
	body, err := brep.Boolean(bop, target, tool)
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
		if csg, cerr := booleanCSG(op, target, tool, lin, rec); cerr == nil && csg != nil && validBooleanSolid(csg) {
			return csg, nil
		}
	}
	return body, nil
}

// curvedExactBoolean tries each exact analytic curved-boolean path in turn, returning the first that
// applies (M2, ADR-0027 §M2). Each keeps the operands' exact analytic surfaces instead of the triangle-soup
// CSG fallback; one that does not apply returns ok=false so booleanGeneral moves on:
//   - a curved solid ∩ a convex planar tool (a box) → composed half-space cuts (#1334);
//   - a curved solid − a convex prism tunnelling through it (cylinder − box) (#1334);
//   - two crossing cylinders ∩ (rod band + fat-wall lens caps) (#1335);
//   - a cone crossing a cylinder ∩ (cone band + cylinder-wall lens caps) (#1335);
//   - a cone crossing a fatter cone ∩/−/∪ (rod-cone band + fat-cone lens caps; drill/stubs/join) (#1335);
//   - two EQUAL-radius perpendicular cylinders ∩/−/∪ (the Steinmetz bicylinder and its cut/union, fitted as crossing ellipses) (#1335);
//   - a thin rod ending inside a fatter cylinder ∩/−/∪ (a partial penetration: the plug, a blind hole, a one-sided stub) (#1335);
//   - drilling a fat cylinder with a crossing rod (fat − rod), and the two rod stubs of rod − fat (#1335);
//   - joining two crossing cylinders (fat ∪ rod: the fat with a rod stub each side) (#1335);
//   - drilling/joining a fat cylinder with a crossing CONE (the tapered tunnel/stubs of cone − fat / fat ∪ cone) (#1335);
//   - drilling a clean through-hole in an all-planar slab with a straight cylinder (a drilled plate: box − cylinder) (#1336);
//   - the union of two coaxial equal-radius cylinders that overlap/abut → one taller cylinder (a coplanar/tangent overlap) (#1336);
//   - the union of a cylinder seated flush on a planar face (a boss/spigot) → seat-face hole + outward wall + cap (a coplanar overlap) (#1336).
func curvedExactBoolean(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	for _, exact := range curvedExactPaths {
		if body, ok := exact(op, target, tool, rec); ok {
			return body, true
		}
	}
	return nil, false
}

// CurvedBoolean attempts the exact analytic curved boolean and reports whether it applied. It is SAFE to
// call on any operands — each path declines (ok=false) when it does not handle (op, target, tool), and none
// hangs (unlike the planar B-rep boolean, which loops on a full periodic curved face). The model layer uses
// it to combine a still-analytic primitive (a revolved torus, an extruded cylinder) by its curved faces
// before falling back to faceting the operands for the planar path (#129).
func CurvedBoolean(op PartFeatureOperation, target, tool *topo.Body) (*topo.Body, bool) {
	return curvedExactBoolean(op, target, tool, nil)
}

// curvedExactPaths is the ordered list of exact analytic curved-boolean paths curvedExactBoolean tries; each
// returns ok=false when it does not apply to (op, target, tool). The recorder carries the SSI imprint's
// closure diagnostics (#1404) up to the boolean's caller; a path that takes no imprint ignores it.
var curvedExactPaths = []func(PartFeatureOperation, *topo.Body, *topo.Body, *diag.Recorder) (*topo.Body, bool){
	curvedConvexIntersect, curvedConvexSubtract,
	curvedCrossingIntersect, curvedSteinmetzIntersect, curvedConeCylinderIntersect, curvedConeConeIntersect,
	curvedPartialIntersect,
	curvedCylindricalHoleCut, curvedFlatSubtract, curvedPartialCut, curvedSteinmetzCut, curvedConeCylinderCut, curvedConeConeCut, curvedCrossingCut,
	curvedCoaxialJoin, curvedCylinderBossJoin, curvedPartialJoin, curvedConeCylinderJoin, curvedConeConeJoin, curvedCrossingJoin, curvedSteinmetzJoin,
}

func shouldFallbackBoolean(op PartFeatureOperation, target, tool, body *topo.Body) bool {
	if !validBooleanSolid(body) {
		return true
	}
	return invalidBooleanVolume(op, target, tool, body)
}

func validBooleanSolid(body *topo.Body) bool {
	r := Validate(body)
	return r.Valid && r.Closed && r.Manifold && body.IsSolid()
}

func invalidBooleanVolume(op PartFeatureOperation, target, tool, body *topo.Body) bool {
	// Model-relative volume tolerance (ADR-0042): scales with the operands' size³ so
	// the result-volume sanity check is faithful at any scale, not just ~cm parts.
	tol := ResolutionForBodies(target, tool).Volume()
	targetVol := BodyGeometryProperties(target, DefaultQuality()).Volume
	toolVol := BodyGeometryProperties(tool, DefaultQuality()).Volume
	bodyVol := BodyGeometryProperties(body, DefaultQuality()).Volume
	switch op {
	case Join:
		return bodyVol+tol < maxFloat(targetVol, toolVol)
	case Cut:
		return bodyVol > targetVol+tol
	case Intersect:
		return bodyVol > minFloat(targetVol, toolVol)+tol
	}
	return false
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// csgFallbackFaceLimit bounds the operand size for the invalid-result CSG fallback (see
// booleanGeneral): small degenerate cases (the V2 flush-bottom oblique penetration) recover
// cheaply, while a huge body's CSG attempt is costly and seldom valid.
const csgFallbackFaceLimit = 256

// toBrepOp maps the feature-level operation to the B-rep boolean operation (NewBody has no
// B-rep analogue — it is handled before this point).
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
	var result []tri
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

// allVerticesInside reports whether every vertex of inner lies strictly within outer. It
// tessellates outer ONCE and reuses that mesh for every vertex query (#1317) — previously each
// vertex re-tessellated the whole body, making boolean classification O(V·T).
func allVerticesInside(inner, outer *topo.Body) bool {
	verts := inner.Vertices()
	if len(verts) == 0 {
		return false
	}
	mesh, _ := TessellateBody(outer, DefaultQuality())
	for _, v := range verts {
		if !pointInMesh(mesh, v.Point()) {
			return false
		}
	}
	return true
}

// PointInsideBody reports whether p is inside a solid body, via the generalized winding number of
// the body's tessellation (see pointInMesh/windingNumber). This replaces the old single fixed-ray
// parity test (#1317), which miscounted whenever the ray grazed a shared edge or vertex of the mesh
// — ubiquitous on a closed surface — flipping the inside/outside result. The winding number
// integrates the entire boundary, so it has no such degeneracy and tolerates small mesh cracks.
func PointInsideBody(b *topo.Body, p math.Point3) bool {
	mesh, _ := TessellateBody(b, DefaultQuality())
	return pointInMesh(mesh, p)
}
