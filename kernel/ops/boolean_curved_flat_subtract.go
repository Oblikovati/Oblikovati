// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// Exact curved − convex-planar subtract bounded by a single plane (M2 Phase-1 follow-up,
// Oblikovati/Oblikovati#1372). Cut is not a half-space composition, but De Morgan gives
// target − box = target ∩ ¬box = ∪ᵢ (target ∩ Hᵢ⁺) over the box faces' OUTER half-spaces. When the
// box's removal from the curved target is bounded by exactly one of its faces (a flat milled on a
// cone or cylinder side: the other faces clear the target), all but one of those pieces are empty, so
// the subtract is a single brep.HalfSpaceCut keeping the target outside that face — an exact curved
// B-rep, the cone/cylinder surface preserved. A removal bounded by several box planes (a corner
// notch) leaves more than one non-empty piece and would need a curved union, so it defers to CSG.

// curvedFlatSubtract returns target − box when the removal is bounded by a single box face, or
// ok=false to defer. Only Cut, a curved target, and a convex all-planar tool map here.
func curvedFlatSubtract(op PartFeatureOperation, target, tool *topo.Body) (*topo.Body, bool) {
	if op != Cut || !hasCurvedFace(target) || hasCurvedFace(tool) {
		return nil, false
	}
	planes, ok := convexFacePlanes(tool) // each oriented along its OUTWARD normal
	if !ok {
		return nil, false
	}
	var pieces []*topo.Body
	for _, pl := range planes {
		inward, err := geom.NewPlane(pl.Origin, pl.NormalAt(0, 0).Scale(-1)) // flip outward → inward
		if err != nil {
			return nil, false
		}
		piece, err := brep.HalfSpaceCut(target, inward) // keeps Hᵢ⁺ = the target outside this face
		if err != nil {
			return nil, false // an unhandled cut: defer the whole subtract to CSG
		}
		if len(piece.Faces()) > 0 {
			pieces = append(pieces, piece)
		}
	}
	if len(pieces) != 1 || !validBooleanSolid(pieces[0]) {
		return nil, false // 0 pieces (box ⊇ target) or several (a multi-plane notch needing a union): defer
	}
	return pieces[0], true
}
