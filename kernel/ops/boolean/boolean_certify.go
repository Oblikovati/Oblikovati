// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Per-face certification of a boolean result (M48/C3, Oblikovati/Oblikovati#3445–#3448).
//
// A whole-body volume comparison is a smoke test, never a proof: a result can hold the right amount
// of material in the wrong place, and a bracket wide enough to absorb a tessellation deficit is wide
// enough to pass a mis-recognized lobe. The proof is Requicha's membership rule applied FACE BY
// FACE — every face of A op B lies on ∂A or ∂B, and on the side the operation keeps:
//
//	A ∪ B: ∂A outside B, or ∂B outside A
//	A ∖ B: ∂A outside B, or ∂B inside A
//	A ∩ B: ∂A inside B,  or ∂B inside A
//
// Both the "on" test and the "inside" test read the analytic B-rep (brep.PointOnFace,
// brep.PointInside), so the gate is exact and Quality-independent — the ground rule that an oracle
// gating a result must be more exact than the result it gates.

// certifyBooleanFaces reports whether every face of body is one the operands can account for under
// op's membership rule. A face whose interior point cannot be found is SKIPPED rather than failed:
// the certificate refuses results it can disprove, and never rejects one merely because a probe was
// unavailable.
//
// Example: if !certifyBooleanFaces(Cut, target, tool, body) { /* fall back to the guarded path */ }
func certifyBooleanFaces(op PartFeatureOperation, target, tool, body *topo.Body) bool {
	if body == nil || target == nil || tool == nil {
		return false
	}
	tol := geom.ResolutionForBox(target.RangeBox().Union(tool.RangeBox())).Sew()
	ta, to := newBoundaryIndex(target), newBoundaryIndex(tool)
	for _, f := range body.Faces() {
		p, ok := query.FaceInteriorPoint(f)
		if !ok {
			continue
		}
		if !pointKeptBy(op, ta, to, p, tol) {
			return false
		}
	}
	return true
}

// boundaryIndex is one operand prepared for repeated "does this point lie on the boundary?"
// questions: its faces, with a box tree over their range boxes, so a probe only reaches the faces
// whose box covers it. It is built ONCE per boolean — scanning every face per probe made the gate
// cost O(result faces × operand faces) exact projections, which on a body of tens of thousands of
// faces costs far more than the boolean it is checking.
type boundaryIndex struct {
	body    *topo.Body
	faces   []*topo.Face
	tree    *geom.BoxTree
	unboxed []*topo.Face // faces with no range box: a BOUNDARY-LESS face has no vertices to build one
}

// newBoundaryIndex prepares b's faces for boundary probes. A face with an empty range box cannot be
// found by a tree query, so those are kept aside and always tested — that is the whole sphere a ball
// is made of, exactly the face a coaxial sphere/rod boolean has to certify.
func newBoundaryIndex(b *topo.Body) *boundaryIndex {
	bi := &boundaryIndex{body: b}
	var boxes []math.Box
	for _, f := range b.Faces() {
		if box := f.RangeBox(); !box.IsEmpty() {
			bi.faces = append(bi.faces, f)
			boxes = append(boxes, box)
			continue
		}
		bi.unboxed = append(bi.unboxed, f)
	}
	bi.tree = geom.NewBoxTree(boxes)
	return bi
}

// on reports whether p lies on any of the body's trimmed faces, within tol.
func (bi *boundaryIndex) on(p math.Point3, tol float64) bool {
	for _, f := range bi.unboxed {
		if brep.PointOnFace(f, p, tol) {
			return true
		}
	}
	found := false
	bi.tree.Query(pointReach(p, tol), func(i int) bool {
		found = brep.PointOnFace(bi.faces[i], p, tol)
		return found
	})
	return found
}

// inside reports whether p is strictly inside the body.
func (bi *boundaryIndex) inside(p math.Point3) bool { return brep.PointInside(bi.body, p) }

// pointReach is the query box for a probe: the point grown by the on-boundary tolerance.
func pointReach(p math.Point3, tol float64) math.Box {
	t := math.Scalar(tol)
	return math.NewBox(math.P3(p.X-t, p.Y-t, p.Z-t), math.P3(p.X+t, p.Y+t, p.Z+t))
}

// pointKeptBy applies the membership rule to one boundary point. A point on BOTH operands'
// boundaries is kept by every operation — that is a coincident-face contact, where the shared
// boundary survives once — so it is accepted before the per-op split.
func pointKeptBy(op PartFeatureOperation, target, tool *boundaryIndex, p math.Point3, tol float64) bool {
	if op != Join && op != Cut && op != Intersect {
		return true // an operation with no membership rule to check
	}
	onTarget, onTool := target.on(p, tol), tool.on(p, tol)
	if !onTarget && !onTool {
		return false // the face lies on neither operand: the result fabricated it
	}
	if onTarget && onTool {
		return true
	}
	if onTarget {
		return targetBoundaryKept(op, tool, p)
	}
	return toolBoundaryKept(op, target, p)
}

// targetBoundaryKept applies the rule to a point on the TARGET's boundary alone: a join or a cut
// keeps the part outside the tool, an intersection keeps only what the tool contains.
func targetBoundaryKept(op PartFeatureOperation, tool *boundaryIndex, p math.Point3) bool {
	if op == Intersect {
		return tool.inside(p)
	}
	return !tool.inside(p)
}

// toolBoundaryKept applies the rule to a point on the TOOL's boundary alone: a cut or an
// intersection keeps what the target contains (the carved wall, the shared lens), a join keeps the
// part outside it.
func toolBoundaryKept(op PartFeatureOperation, target *boundaryIndex, p math.Point3) bool {
	if op == Join {
		return !target.inside(p)
	}
	return target.inside(p)
}
