// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
)

// A cut THROUGH a sphere's pole is the sphere chart's singular case: longitude names no direction at
// v = ±π/2, so the two halves of the section curve reach the pole on meridians half a turn apart. Until
// ADR-0061's pole anchoring the polyline carried one branch across the pole by continuity, which put a
// half-turn leap into the arrangement at v = −π/2; the leap crossed the chart's seam and the boundary
// walk followed it out along the seam, leaving two open edges. It failed for HALF of all cut
// orientations — whichever way round the kept side landed relative to the placed seam — so the corpus
// takes all four axis-aligned removals, not the one that happened to be sampled
// (Oblikovati/Oblikovati#1334).
func TestSphereCapCutThroughItsPoleClosesAtEveryOrientation(t *testing.T) {
	t.Parallel()
	const R = 5.0
	sphere, _ := brep.SolidSphere(math.P3(0, 0, 0), R, "s")
	capPlane, _ := geom.NewPlane(math.P3(0, 0, -3), math.V3(0, 0, 1)) // keep z ≤ −3: a cap holding the pole
	cap, err := brep.HalfSpaceCut(sphere, capPlane)
	if err != nil {
		t.Fatalf("cap cut: %v", err)
	}
	capVol := query.BodyGeometryProperties(cap, ops.DefaultQuality()).Volume

	for _, row := range []struct {
		name     string
		min, max math.Point3
	}{
		{"remove x ≥ 0", math.P3(0, -10, -10), math.P3(10, 10, 10)},
		{"remove x ≤ 0", math.P3(-10, -10, -10), math.P3(0, 10, 10)},
		{"remove y ≥ 0", math.P3(-10, 0, -10), math.P3(10, 10, 10)},
		{"remove y ≤ 0", math.P3(-10, -10, -10), math.P3(10, 0, 10)},
	} {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			tool, err := brep.SolidBlock(row.min, row.max, "tool")
			if err != nil {
				t.Fatalf("tool: %v", err)
			}
			half, err := brep.Boolean(brep.Difference, cap, tool)
			if err != nil {
				t.Fatalf("difference: %v", err)
			}
			if r := ops.Validate(half); !r.Valid || !r.Closed || !r.Manifold || !half.IsSolid() {
				t.Fatalf("%s: not a valid closed manifold solid: %+v", row.name, r)
			}
			got := query.BodyGeometryProperties(half, ops.DefaultQuality()).Volume
			want := capVol / 2 // the plane of the cut is a plane of symmetry of the cap
			if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
				t.Errorf("%s: half volume %.6f, want %.6f (cap/2) — rel %.4f > 2%%", row.name, got, want, rel)
			}
		})
	}
}
