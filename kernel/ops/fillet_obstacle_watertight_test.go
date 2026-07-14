// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestFilletSlabColumnWatertight is the crux body-level gate for the mid-span obstacle slice (ADR-4,
// Task 6): a slab (z[-20,0]) carrying an elliptical column/tube (z[0,30]) whose footprint is a hole in
// the slab-top plane, filleted along the slab's front-top edge at r=6. The fillet's receded boundary
// (y=-7) dips into the column footprint ellipse (y∈[-10,10]), so without the obstacle rebuild the hole
// protrudes past the shrunken outer loop (HolesContained=false). With the rebuild — notched host,
// split obstacle wall, two wings and a corner-blend patch — the result must be a single, watertight,
// hole-contained solid. This drives the REAL fillet feature end to end, not the provider directly.
func TestFilletSlabColumnWatertight(t *testing.T) {
	body := slabWithColumn(t)
	res, ok, reason := filletFrontTopEdge(body, 6)
	if !ok {
		t.Fatalf("fillet failed: %s", reason)
	}
	rep := Validate(res)
	if !rep.HolesContained {
		t.Errorf("filleted slab+column must be hole-contained (no protrusion): %v", rep.Issues)
	}
	if !rep.Valid {
		t.Errorf("filleted slab+column must be a valid solid: %v", rep.Issues)
	}
	if !res.IsSolid() {
		t.Errorf("filleted slab+column must be a single closed solid")
	}
}

// slabWithColumn imports the slab-with-elliptical-column fixture (an elliptical tube rising from a
// hole in a box top — the canonical mid-span obstacle shape). Built via STEP import, mirroring the
// other ops fillet fixtures (importPrismCylBorder / importNotchedPrism), so the topology is a real
// imported B-rep, not a hand-welded stub.
func slabWithColumn(t *testing.T) *topo.Body {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "obstacle_slab_column.step"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	bodies, _, err := step.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	if err != nil || len(bodies) != 1 {
		t.Fatalf("import: %v (n=%d)", err, len(bodies))
	}
	return bodies[0]
}

// filletFrontTopEdge drives the real fillet feature over the slab's front-top edge (midpoint
// (0,-13,0), along +X) at radius r, returning the result body plus a health flag/reason.
func filletFrontTopEdge(body *topo.Body, r float64) (*topo.Body, bool, string) {
	edge := edgeAtMidpoint(body, math.P3(0, -13, 0))
	if edge == nil {
		return nil, false, "front-top edge (midpoint 0,-13,0) not found on the imported body"
	}
	res, err := FilletEdges(body, [][]byte{edge.ReferenceKey()}, r)
	if err != nil {
		return nil, false, err.Error()
	}
	return res, true, ""
}

// edgeAtMidpoint returns the body edge whose endpoint midpoint matches mid (within a mm-scale
// tolerance), or nil when none does.
func edgeAtMidpoint(b *topo.Body, mid math.Point3) *topo.Edge {
	for _, e := range b.Edges() {
		if e.StartVertex().Point().Midpoint(e.EndVertex().Point()).DistanceTo(mid) < 1e-3 {
			return e
		}
	}
	return nil
}
