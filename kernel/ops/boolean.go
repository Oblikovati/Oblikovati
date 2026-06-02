// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"github.com/Oblikovati/oblikovati/build"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
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

// Boolean combines target and tool under op. Phase A handles the non-overlapping
// topological cases — disjoint bodies and one fully containing the other — which
// produce valid results without splitting faces. General intersecting booleans
// (face-face intersection, splitting and re-stitching) are the hardest part of the
// kernel and are phased to Phase C, returning NotYetImplemented for now
// (architecture core/03). Result-face lineage flows from the operands, so reference
// keys stay rebindable.
func Boolean(op PartFeatureOperation, target, tool *topo.Body) (*topo.Body, error) {
	lin := topo.NewLineage(topo.Tok("boolean", op.String(), 0))
	if op == NewBody {
		return tool, nil
	}
	rel := classify(target, tool)
	switch op {
	case Join:
		if rel == intersecting {
			return nil, build.NotYetImplemented("PBI-082-join-intersecting")
		}
		return topo.MergeBodies(lin, true, target, tool), nil
	case Cut:
		return cut(lin, target, rel)
	default: // Intersect
		return intersect(lin, target, tool, rel)
	}
}

func cut(lin topo.Lineage, target *topo.Body, rel relation) (*topo.Body, error) {
	switch rel {
	case disjoint:
		return target, nil // tool removes nothing
	case toolContainsTarget:
		return topo.MergeBodies(lin, true), nil // target fully removed → empty
	default: // targetContainsTool (cavity) and intersecting need face splitting
		return nil, build.NotYetImplemented("PBI-082-cut-intersecting")
	}
}

func intersect(lin topo.Lineage, target, tool *topo.Body, rel relation) (*topo.Body, error) {
	switch rel {
	case disjoint:
		return topo.MergeBodies(lin, true), nil // nothing in common → empty
	case targetContainsTool:
		return tool, nil
	case toolContainsTarget:
		return target, nil
	default:
		return nil, build.NotYetImplemented("PBI-082-intersect-intersecting")
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

// classify determines the spatial relationship from bounding boxes and vertex
// containment. It recognizes the cases Phase A can combine without face splitting;
// anything else is reported as intersecting.
func classify(target, tool *topo.Body) relation {
	if !target.RangeBox().Intersects(tool.RangeBox()) {
		return disjoint
	}
	if allVerticesInside(tool, target) {
		return targetContainsTool
	}
	if allVerticesInside(target, tool) {
		return toolContainsTarget
	}
	return intersecting
}

// allVerticesInside reports whether every vertex of inner lies strictly within outer.
func allVerticesInside(inner, outer *topo.Body) bool {
	verts := inner.Vertices()
	if len(verts) == 0 {
		return false
	}
	for _, v := range verts {
		if !PointInsideBody(outer, v.Point()) {
			return false
		}
	}
	return true
}

// PointInsideBody reports whether p is inside a solid body, by casting a ray from p
// and counting crossings of the body's tessellated faces (odd ⇒ inside). The ray
// direction is skewed off the axes to avoid degenerate grazing hits.
func PointInsideBody(b *topo.Body, p math.Point3) bool {
	mesh, _ := TessellateBody(b, DefaultQuality())
	dir := math.V3(0.5773, 0.5774, 0.5775) // near-diagonal, non-axis-aligned
	crossings := 0
	for t := 0; t+2 < len(mesh.Indices); t += 3 {
		a := mesh.Positions[mesh.Indices[t]]
		q := mesh.Positions[mesh.Indices[t+1]]
		r := mesh.Positions[mesh.Indices[t+2]]
		if rayHitsTriangle(p, dir, a, q, r) {
			crossings++
		}
	}
	return crossings%2 == 1
}

// rayHitsTriangle reports whether the ray from orig along dir hits triangle abc at a
// positive distance (Möller–Trumbore; shares rayTriangleDist with picking).
func rayHitsTriangle(orig math.Point3, dir math.Vector3, a, b, c math.Point3) bool {
	_, ok := rayTriangleDist(orig, dir, a, b, c)
	return ok
}
