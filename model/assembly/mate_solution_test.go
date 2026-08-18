// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// TestMateSolutionTypes the four mate solutions shape the plane-mate residuals (#1971): the three
// directed senses pin the two normals parallel and hold the offset (two rotational + one gap), while
// no-solution leaves the sense — and the parallelism it would imply — unconstrained, holding ONLY the
// offset. alignResiduals is sense-blind (it pins A's normal onto B's axis, either way), so opposed,
// aligned and undirected share one residual set; undirected names that the solver may keep whichever
// sense the parts already hold, so a drag never forces a flip.
func TestMateSolutionTypes(t *testing.T) {
	a := PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))  // +Z
	b := PlanePrimitive(math.P3(0, 0, 5), unit(t, 0, 0, -1)) // −Z, 5 apart (an opposed mate)
	id := math.Identity4()

	for _, sol := range []types.MateConstraintSolutionType{
		types.MateSolutionOpposed, types.MateSolutionAligned, types.MateSolutionUndirected,
	} {
		r := planeMateResiduals(a, b, id, id, 5, sol)
		if len(r) != 3 {
			t.Errorf("solution %v: %d residuals, want 3 (two rotational + one gap)", sol, len(r))
		}
		if stdmath.Abs(r[0]) > 1e-9 || stdmath.Abs(r[1]) > 1e-9 {
			t.Errorf("solution %v: parallel planes left a rotational residual %v", sol, r[:2])
		}
	}

	// No-solution holds only the offset gap — the normals need not even be parallel.
	if r := planeMateResiduals(a, b, id, id, 5, types.MateSolutionNoSolution); len(r) != 1 {
		t.Fatalf("noSolution: %d residuals, want 1 (gap only)", len(r))
	}

	// Undirected must not force a flip: on parts whose normals already point the SAME way, its
	// rotational residuals still vanish (the mate is satisfied in the current orientation).
	c := PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1)) // +Z
	d := PlanePrimitive(math.P3(0, 0, 5), unit(t, 0, 0, 1)) // +Z (same sense)
	und := planeMateResiduals(c, d, id, id, 5, types.MateSolutionUndirected)
	if stdmath.Abs(und[0]) > 1e-9 || stdmath.Abs(und[1]) > 1e-9 {
		t.Errorf("undirected forced a rotation on same-sense normals: %v", und[:2])
	}
}
