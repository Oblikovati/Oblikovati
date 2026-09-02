// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/mesh"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The three set operations on PLANAR operands, and the containment relation they branch on (split
// out of boolean.go for #2215).
//
// classify decides the relation between two bodies once — disjoint, containing, or overlapping —
// and join/cut/intersect each read that one answer. Deciding an incidence once and reusing it is
// the point: two call sites recomputing the same containment could disagree.

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
	return allVerticesInside(inner, outer) && !query.BoundariesCross(outer, inner)
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
