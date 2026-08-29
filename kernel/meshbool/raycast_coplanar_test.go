// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import "testing"

// TestSegmentCoplanarOutsideIsMiss pins the #2247 classification fix: a ray endpoint coplanar with a
// triangle's PLANE but OUTSIDE the triangle is a clean miss, not a degeneracy. Before the fix, any
// coplanar endpoint was flagged degenerate, so classifying a point that sat at a feature height (a cut
// floor at the same z as a distant coplanar step floor) rejected EVERY ray direction and insideExact
// fell through to a wrong "outside", dropping a cut face and tearing chained booleans.
func TestSegmentCoplanarOutsideIsMiss(t *testing.T) {
	// A triangle in the z=1 plane spanning x in [3,4].
	a, b, c := pt([3]float64{3, 0, 1}), pt([3]float64{4, 0, 1}), pt([3]float64{3, 2, 1})

	// p is coplanar (z=1) but well outside the triangle (x=2); the ray to a far point above must MISS
	// it cleanly, not report a degeneracy.
	if crosses, degen := segmentPiercesTriExact(pt([3]float64{2, 1, 1}), pt([3]float64{2, 1, 3}), a, b, c); crosses || degen {
		t.Errorf("coplanar-outside start: got crosses=%v degenerate=%v, want clean miss (false,false)", crosses, degen)
	}
	// A point coplanar AND inside the triangle is genuinely on the surface — still a degeneracy.
	if _, degen := segmentPiercesTriExact(pt([3]float64{3, 1, 1}), pt([3]float64{3, 1, 3}), a, b, c); !degen {
		t.Errorf("coplanar-inside start: want degenerate=true (the point lies on the triangle)")
	}
	// A normal transversal crossing through the triangle INTERIOR is unaffected.
	if crosses, degen := segmentPiercesTriExact(pt([3]float64{3.5, 0.4, 0}), pt([3]float64{3.5, 0.4, 2}), a, b, c); !crosses || degen {
		t.Errorf("transversal pierce: got crosses=%v degenerate=%v, want (true,false)", crosses, degen)
	}
}

// TestInsideExactInteriorAtFeaturePlane is the insideExact-level regression: a point strictly inside a
// notched solid but sitting at the z of a distant coplanar face must classify as INSIDE.
func TestInsideExactInteriorAtFeaturePlane(t *testing.T) {
	// An L-shaped solid: the [0,4]x[0,2]x[0,2] box with the [3,4]x[0,2]x[1,2] corner removed, whose
	// step floor lies in the z=1 plane. Built as a soup by BooleanTagged difference (the engine under
	// test's own union of the two boxes is exercised elsewhere).
	box := boxMesh([3]float64{0, 0, 0}, [3]float64{4, 2, 2})
	notch := boxMesh([3]float64{3, 0, 1}, [3]float64{4, 2, 3})
	lshape := Boolean(box, notch, Difference)
	grid := newFaceGrid(lshape)
	// (2,1,1) is deep inside the full part, at the step floor's z. It must read as inside.
	if !insideExact(pt([3]float64{2, 1, 1}), lshape, grid) {
		t.Error("interior point at a coplanar-feature height classified as OUTSIDE (the #2247 raycast bug)")
	}
}
