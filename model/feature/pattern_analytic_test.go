// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
	"oblikovati/model/sketch"
)

// discThenCutPatterned models the wheel core: a Ø60 disc, one Ø6 bolt-hole cut, then a 5-up
// circular pattern of that cut. The disc and bolt are analytic cylinders; the pattern boolean must
// re-facet the curved tool (#129) or it explodes (the raw periodic-cylinder face used to blow the
// body up to tens of thousands of edges and hang).
func discThenCutPatterned(t *testing.T) *topo.Body {
	t.Helper()
	fs := NewPartFeatures(nil, nil)
	disc := sketch.NewSketches().Add(sketch.XYPlane())
	disc.Circles().AddByCenterRadius(math.P2(0, 0), 30)
	NewExtrudeFeatures(fs).AddByDistanceExtent(disc, 0, ops.NewBody, func() float64 { return 10 })

	hole := sketch.NewSketches().Add(sketch.XYPlane())
	hole.Circles().AddByCenterRadius(math.P2(20, 0), 3)
	cut := NewExtrudeFeatures(fs).AddExtrude(hole, []int{0}, ops.Cut, Extent{Type: DistanceExtent, Distance: func() float64 { return 10 }}, 0)

	NewPatternFeatures(fs).AddCircular([]ID{cut.ID()}, func() int { return 5 }, func() float64 { return 2 * 3.141592653589793 },
		math.P3(0, 0, 0), math.V3(0, 0, 1))

	fs.Recompute()
	bodies := fs.Result()
	if len(bodies) != 1 {
		t.Fatalf("disc+cut = %d bodies, want 1", len(bodies))
	}
	return bodies[0]
}

// TestAnalyticPatternedCutDoesNotExplode is the #129 regression for the curved-tool pattern blow-up:
// the patterned bolt cut must re-facet the analytic bolt before each boolean, leaving ONE valid solid
// of bounded size with five holes removed — not a tens-of-thousands-of-edges (multi-minute) explosion
// from feeding a raw periodic cylinder to the planar boolean. (Disc-level reference-key stability vs a
// direct faceted extrude is pinned separately by TestPlanarizedDiscMatchesFacetedExtrude.)
func TestAnalyticPatternedCutDoesNotExplode(t *testing.T) {
	body := discThenCutPatterned(t)
	if vr := ops.Validate(body); !vr.Valid || !body.IsSolid() {
		t.Fatalf("patterned cut is not a valid solid: %+v", vr.Issues)
	}
	if n := len(body.Edges()); n > 2000 {
		t.Fatalf("patterned cut exploded to %d edges (curved tool not re-faceted before the boolean?)", n)
	}
	disc := stdmath.Pi * 30 * 30 * 10
	holes := 5 * stdmath.Pi * 3 * 3 * 10 // five Ø6 through-holes
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(got, disc-holes) > 0.03 {
		t.Fatalf("patterned-cut volume = %g, want ≈%g (disc − 5 holes)", got, disc-holes)
	}
}
