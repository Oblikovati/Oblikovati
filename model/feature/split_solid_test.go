// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/subd"
	"github.com/Oblikovati/oblikovati/kernel/topo"
)

// midPlaneAt builds work geometry with an XY work plane offset to z, recomputed and ready.
func midPlaneAt(z float64) (*WorkGeometry, *WorkPlane) {
	g := NewWorkGeometry()
	g.Recompute(nil)
	wp := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return z })
	g.Recompute(nil)
	return g, wp
}

func boxBody() *topo.Body { return subd.ToBody(subd.Box(4, 4, 4), "box") }

// A solid split by a mid work plane divides the part into two valid solids (each half-volume).
func TestSplitSolidFeatureDividesBox(t *testing.T) {
	_, wp := midPlaneAt(2)
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(boxBody())
	split := NewModifyFeatures(fs).AddSplitSolid(wp, SplitBoth)
	fs.Recompute()

	if !split.Health().OK() {
		t.Fatalf("split went sick: %+v", split.Health())
	}
	pieces := fs.Result()
	if len(pieces) != 2 {
		t.Fatalf("split result = %d bodies, want 2", len(pieces))
	}
	for _, p := range pieces {
		if v := ops.BodyGeometryProperties(p, ops.DefaultQuality()).Volume; stdmath.Abs(v-32) > 1e-6 {
			t.Errorf("piece volume = %g, want 32", v)
		}
	}
}

// Trim Solid (keep one side) leaves a single body — the kept half.
func TestSplitSolidFeatureTrimsOneSide(t *testing.T) {
	_, wp := midPlaneAt(2)
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(boxBody())
	NewModifyFeatures(fs).AddSplitSolid(wp, SplitNegative) // keep below z=2
	fs.Recompute()

	pieces := fs.Result()
	if len(pieces) != 1 {
		t.Fatalf("trim result = %d bodies, want 1 (kept side)", len(pieces))
	}
	if v := ops.BodyGeometryProperties(pieces[0], ops.DefaultQuality()).Volume; stdmath.Abs(v-32) > 1e-6 {
		t.Errorf("kept volume = %g, want 32", v)
	}
}

// The split's plane reference and kept side round-trip through the recipe.
func TestSplitSolidRoundTrip(t *testing.T) {
	g, wp := midPlaneAt(2)
	fs := NewPartFeatures(nil, nil)
	NewModifyFeatures(fs).AddSplitSolid(wp, SplitPositive)

	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, g); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	def := fresh.Item(0).Definition().(*SplitSolidFeature).Definition()
	if def.Keep != SplitPositive || def.Plane == nil || def.Plane.Key() != wp.Key() {
		t.Errorf("restored split = keep %d plane %v, want positive + the z=2 plane", def.Keep, def.Plane)
	}
}
