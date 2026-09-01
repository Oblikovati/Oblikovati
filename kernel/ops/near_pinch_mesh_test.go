// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/math"
)

// TestNearPinchSnapMeshWatertight pins the tessellation-correctness gate — the mesh the user actually sees.
// Two crossing cylinders whose radii differ by less than the snap ceiling (#1780) intersect to a bicylinder
// whose TESSELLATED mesh is watertight (0 free edges): the snap produces a clean closed surface, where the
// SAME input took the non-manifold ~6300-face faceted fallback before #1780. This is the mesh-level companion
// to the B-rep-level Validate checks in boolean_crossing_cylinder_test.go.
func TestNearPinchSnapMeshWatertight(t *testing.T) {
	t.Parallel()
	const r = 3.0
	bx, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), r, 12)
	bz, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), r, 12)
	dr := 0.5 * geom.ResolutionForBox(bx.RangeBox().Union(bz.RangeBox())).Stitch() // within the snap ceiling

	cx, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), r, 12)
	cz, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), r+dr, 12)
	res, err := Boolean(Intersect, cx, cz)
	if err != nil {
		t.Fatalf("Boolean(Intersect near-equal): %v", err)
	}
	for _, gq := range gateQualities() {
		m, _ := tessellate.TessellateBody(res, gq.q)
		if free := freeEdgeCount(m); free != 0 {
			t.Errorf("%s quality: snapped near-equal bicylinder meshed with %d free edges; want 0 (watertight surface)",
				gq.name, free)
		}
	}
}
