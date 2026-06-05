// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// A twisted loft has warped (non-planar) ruled side faces. sweptSolid triangulates those so the
// body stays planar-faceted; without it the planar boolean's imprint segments land offset from
// the true edges and a partial-penetration union goes non-manifold (the deformed fan blade).
// This pins the fix: a box hub UNION a twisted-loft blade (which pokes part-way into the hub)
// must be a valid manifold solid at every twist — even a 0.001 rad twist used to break it.
func TestTwistedLoftUnionStaysManifold(t *testing.T) {
	xy := sketch.XYPlane()
	square := []math.Point2{{X: -0.675, Y: -0.675}, {X: 0.675, Y: -0.675}, {X: 0.675, Y: 0.675}, {X: -0.675, Y: 0.675}}
	root := []math.Point3{{X: 0.6, Y: -0.08, Z: 0}, {X: 2.35, Y: -0.08, Z: 0}, {X: 2.35, Y: 0.08, Z: 0}, {X: 0.6, Y: 0.08, Z: 0}}

	for _, twist := range []float64{0, 0.001, 0.05, 0.3} {
		cx, cy := 1.475, 0.0
		c, s := stdmath.Cos(twist), stdmath.Sin(twist)
		tip := make([]math.Point3, 4)
		for i, p := range root {
			dx, dy := p.X-cx, p.Y-cy
			tip[i] = math.Point3{X: cx + dx*c - dy*s, Y: cy + dx*s + dy*c, Z: 1.4}
		}
		blade, err := sweptSolid([][]math.Point3{root, tip}, false, "blade")
		if err != nil {
			t.Fatalf("twist=%.3f sweptSolid: %v", twist, err)
		}
		hub := buildPrism(square, xy, span{near: 0, far: 1.5}, 0, "hub")
		res, err := ops.Boolean(ops.Join, hub, blade)
		if err != nil {
			t.Fatalf("twist=%.3f join: %v", twist, err)
		}
		if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
			t.Errorf("twist=%.3f union not a valid solid: manifold=%v closed=%v orient=%v issues=%v",
				twist, r.Manifold, r.Closed, r.OrientationOK, r.Issues[:min(4, len(r.Issues))])
		}
	}
}
