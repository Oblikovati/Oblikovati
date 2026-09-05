// SPDX-License-Identifier: GPL-2.0-only
package brep

import (
	"testing"

	"oblikovati.org/math"
)

// TestRadialSewSurfaceAgnostic proves the Weiler radial sew resolves a >2-use (tangent) edge from the
// injected per-face surface normals — the OCCT GetFaceDir contract — so it works for CURVED faces, not
// only the planar boolean's constant normals (ADR-0058). Four half-edge uses on one edge, two enter and
// two exit boundaries at distinct azimuths, must pair into two manifold dihedral groups.
func TestRadialSewSurfaceAgnostic(t *testing.T) {
	t.Parallel()
	uses := map[[2]int][]loopEdgeUse{{0, 1}: {
		{face: 0, reversed: true},  // enter, normal → +x  (interior +y)
		{face: 1, reversed: false}, // exit,  normal → +y  (interior −x)
		{face: 2, reversed: true},  // enter, normal → −x  (interior −y)
		{face: 3, reversed: false}, // exit,  normal → −y  (interior +x)
	}}
	// Curved-style faceDirAt: each face's outward normal supplied per-use (the value a surface would
	// return at the edge), NOT read off a planar builtFace.
	normals := []math.Vector3{math.V3(1, 0, 0), math.V3(0, 1, 0), math.V3(-1, 0, 0), math.V3(0, -1, 0)}
	dir := func(h loopEdgeUse, _ math.Point3) math.Vector3 { return normals[h.face] }

	axis := func() (math.Vector3, math.Point3) { return math.V3(0, 0, 1), math.P3(0, 0, 0.5) }
	groups := resolveEdgeUses(uses[[2]int{0, 1}], axis, dir)
	if len(groups) != 2 {
		t.Fatalf("the radial sew produced %d groups, want 2 manifold dihedrals", len(groups))
	}
	for gi, g := range groups {
		if len(g) != 2 {
			t.Errorf("group %d has %d uses, want 2 (one enter + one exit boundary)", gi, len(g))
		}
	}
}
