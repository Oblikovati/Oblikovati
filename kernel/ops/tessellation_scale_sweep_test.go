// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/math"
)

// The #1610 scale sweep: the same drilled plate must tessellate watertight from 10 µm
// (1e-3 cm — watch-scale) to 10 m (1e3 cm) extents. The fixed 1e-6/1e-5 weld grids this
// pins against merged DISTINCT vertices on the µm part (collapsed geometry, masked
// cracks) and the shared trim-grid tolerance left T-junctions in the metric direction;
// both now derive from the model's own resolution (ADR-0042).
func TestTessellationWatertightAcrossScales(t *testing.T) {
	t.Parallel()
	for _, s := range []float64{1e-4, 1e-3, 1, 1e3, 1e4} {
		t.Run(fmt.Sprintf("scale=%g", s), func(t *testing.T) {
			for _, gq := range gateQualities() {
				mesh := tessellatedDrilledPlate(t, s, gq.q)
				if free := weldedFreeEdgeCount(mesh); free != 0 {
					t.Errorf("scale %g at %s quality: drilled plate has %d free (unpaired) mesh edges — not watertight",
						s, gq.name, free)
				}
				// Volume against the analytic slab − πr²h: over-merged (collapsed) vertices
				// would pass a free-edge count but wreck the enclosed volume, so this is the
				// half of the check watertightness alone cannot see.
				want := (2 * 2 * 0.6 * s * s * s) - stdmath.Pi*(0.3*s)*(0.3*s)*(0.6*s)
				got := meshGeometryProperties(mesh).Volume
				if rel := stdmath.Abs(got-want) / want; rel > 0.01 {
					t.Errorf("scale %g at %s quality: mesh volume %g, want ≈%g (rel err %g)", s, gq.name, got, want, rel)
				}
			}
		})
	}
}

// tessellatedDrilledPlate drills a through-hole in a slab at scale s and returns the
// merged body mesh — a planar multi-hole cap + curved wall, the CDT/conformance path.
func tessellatedDrilledPlate(t *testing.T, s float64, q Quality) *Mesh {
	t.Helper()
	slab, err := brep.SolidBlock(math.P3(-1*math.Scalar(s), -1*math.Scalar(s), 0), math.P3(1*math.Scalar(s), 1*math.Scalar(s), 0.6*math.Scalar(s)), "slab")
	if err != nil {
		t.Fatalf("SolidBlock(%g): %v", s, err)
	}
	rod, err := brep.SolidCylinder(math.P3(0, 0, -0.1*math.Scalar(s)), math.V3(0, 0, 1), 0.3*s, 0.8*s)
	if err != nil {
		t.Fatalf("SolidCylinder(%g): %v", s, err)
	}
	drilled, err := Boolean(Cut, slab, rod)
	if err != nil {
		t.Fatalf("Boolean cut(%g): %v", s, err)
	}
	mesh, _ := TessellateBody(drilled, q)
	return mesh
}
