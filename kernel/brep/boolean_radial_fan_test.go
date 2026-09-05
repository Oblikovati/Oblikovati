// SPDX-License-Identifier: GPL-2.0-only

package brep

import "testing"

// The radial fan's connector is the LOOP, not the face. A face whose boundary passes through one vertex
// on two of its loops is pinched there, and the two loops are two separate fans on that face — exactly
// as two faces kissing at a point are two fans on the body. Joining them by face identity welded a
// torus's figure-eight pinch onto a single vertex, and the body came out with an odd Euler
// characteristic that the validity gate rightly refuses (ADR-0061).
func TestGroupFansSeparatesTwoLoopsOfOneFace(t *testing.T) {
	t.Parallel()
	// The figure-eight pinch: lobe A bounds lid A and the torus's OUTER loop; lobe B bounds lid B and
	// the torus's HOLE loop. Only the torus face (2) is common, and it uses them on different loops.
	groups := []edgeGroup{
		{pair: [2]int{0, 0}, uses: []loopEdgeUse{{face: 0, ring: 0}, {face: 2, ring: 0}}},
		{pair: [2]int{0, 0}, uses: []loopEdgeUse{{face: 1, ring: 0}, {face: 2, ring: 1}}},
	}
	if fans := groupFans(groups, []int{0, 1}); len(fans) != 2 {
		t.Errorf("a pinch on two loops of one face gave %d fan(s), want 2 (one vertex each)", len(fans))
	}
}

// The ordinary vertex is unchanged: three edges meeting where three faces do, each face's single loop
// using two of them, is ONE fan whatever the connector — a refinement must never split a manifold
// vertex.
func TestGroupFansKeepsAManifoldVertexWhole(t *testing.T) {
	t.Parallel()
	groups := []edgeGroup{
		{pair: [2]int{0, 1}, uses: []loopEdgeUse{{face: 0, ring: 0}, {face: 1, ring: 0}}},
		{pair: [2]int{0, 2}, uses: []loopEdgeUse{{face: 1, ring: 0}, {face: 2, ring: 0}}},
		{pair: [2]int{0, 3}, uses: []loopEdgeUse{{face: 2, ring: 0}, {face: 0, ring: 0}}},
	}
	if fans := groupFans(groups, []int{0, 1, 2}); len(fans) != 1 {
		t.Errorf("a manifold corner gave %d fans, want 1", len(fans))
	}
}

// Two loops of one face are still unioned wherever another face genuinely joins their groups, so the
// refinement never splits a vertex the geometry holds together.
func TestGroupFansStillUnionsThroughAnotherFace(t *testing.T) {
	t.Parallel()
	groups := []edgeGroup{
		{pair: [2]int{0, 0}, uses: []loopEdgeUse{{face: 0, ring: 0}, {face: 2, ring: 0}}},
		{pair: [2]int{0, 0}, uses: []loopEdgeUse{{face: 0, ring: 1}, {face: 3, ring: 0}}},
		{pair: [2]int{0, 4}, uses: []loopEdgeUse{{face: 2, ring: 0}, {face: 3, ring: 0}}},
	}
	if fans := groupFans(groups, []int{0, 1, 2}); len(fans) != 1 {
		t.Errorf("groups joined through a third face gave %d fans, want 1", len(fans))
	}
}
