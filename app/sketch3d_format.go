// SPDX-License-Identifier: GPL-2.0-only

package app

// The Format panel's armed creation modes in a 3D sketch (#2039).
//
// The 2D side applies them from the recipe commit and from the two tools that bypass it. A 3D
// sketch has no recipe, and every 3D tool adds its geometry straight to the sketch in Commit —
// so arming Construction on the 3D Sketch tab marked nothing at all.
//
// Rather than call applyFormatModes from each of the eight 3D tools (the shape that already let
// two 2D tools fall out of step), the commit seam marks what the sketch held before the tool ran
// and formats exactly what it added. A new 3D tool is covered by construction.

// sketch3DCreationMark is what the active 3D sketch held before a tool committed.
type sketch3DCreationMark struct {
	entities   int
	dimensions int
	live       bool // false outside the 3D-sketch environment, where there is nothing to mark
}

// markSketch3DCreation snapshots the active 3D sketch's counts ahead of a tool commit.
func (s *Session) markSketch3DCreation() sketch3DCreationMark {
	sk := s.activeSketch3D
	if sk == nil {
		return sketch3DCreationMark{}
	}
	return sketch3DCreationMark{
		entities:   sk.EntityCount(),
		dimensions: sk.DimensionConstraints3D().Count(),
		live:       true,
	}
}

// applyFormatModes3D marks everything the commit added to the active 3D sketch per the armed
// creation modes: geometry per applyFormatModesTo (construction is the only mode registered on
// the 3D tab — a centerline axis and a hole-centre marker are planar concepts), and new
// dimensions as driven.
//
// Converting an EXISTING 3D dimension is not reachable: 3D dimensions have no viewport pick, so
// the selection can never hold one and ToggleDrivenDimension always takes its arm branch there.
func (s *Session) applyFormatModes3D(m sketch3DCreationMark) {
	sk := s.activeSketch3D
	if !m.live || sk == nil {
		return
	}
	ents := sk.Entities()
	for i := m.entities; i < len(ents); i++ {
		s.applyFormatModesTo(ents[i])
	}
	if !s.formatModes.drivenDim {
		return
	}
	dims := sk.DimensionConstraints3D().All()
	for i := m.dimensions; i < len(dims); i++ {
		dims[i].SetDriven(true)
	}
}
