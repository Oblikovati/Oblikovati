// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
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
	for _, f := range body.Faces() {
		p, ok := FaceInteriorPoint(f)
		if !ok {
			continue
		}
		if !pointKeptBy(op, target, tool, p, tol) {
			return false
		}
	}
	return true
}

// pointKeptBy applies the membership rule to one boundary point. A point on BOTH operands'
// boundaries is kept by every operation — that is a coincident-face contact, where the shared
// boundary survives once — so it is accepted before the per-op split.
func pointKeptBy(op PartFeatureOperation, target, tool *topo.Body, p math.Point3, tol float64) bool {
	if op != Join && op != Cut && op != Intersect {
		return true // an operation with no membership rule to check
	}
	onTarget, onTool := onBodyBoundary(target, p, tol), onBodyBoundary(tool, p, tol)
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
func targetBoundaryKept(op PartFeatureOperation, tool *topo.Body, p math.Point3) bool {
	if op == Intersect {
		return brep.PointInside(tool, p)
	}
	return !brep.PointInside(tool, p)
}

// toolBoundaryKept applies the rule to a point on the TOOL's boundary alone: a cut or an
// intersection keeps what the target contains (the carved wall, the shared lens), a join keeps the
// part outside it.
func toolBoundaryKept(op PartFeatureOperation, target *topo.Body, p math.Point3) bool {
	if op == Join {
		return !brep.PointInside(target, p)
	}
	return brep.PointInside(target, p)
}

// onBodyBoundary reports whether p lies on any of b's trimmed faces, pruning by face range box
// first so the scan is near-linear in the faces the point can actually touch.
func onBodyBoundary(b *topo.Body, p math.Point3, tol float64) bool {
	for _, f := range b.Faces() {
		if boxReaches(f.RangeBox(), p, tol) && brep.PointOnFace(f, p, tol) {
			return true
		}
	}
	return false
}

// boxReaches reports whether p is inside box grown by tol — the broad-phase filter that keeps the
// boundary scan off faces the point cannot possibly touch. A face on an unbounded surface still has
// a bounded range box, so this never rejects a reachable face.
func boxReaches(box math.Box, p math.Point3, tol float64) bool {
	t := math.Scalar(tol)
	return box.Min.X-t <= p.X && p.X <= box.Max.X+t &&
		box.Min.Y-t <= p.Y && p.Y <= box.Max.Y+t &&
		box.Min.Z-t <= p.Z && p.Z <= box.Max.Z+t
}
