// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
)

// TestFilletCrossG2BuildsValidSolid: a G2 cross-section fillet through the feature engine rounds a box
// edge into a watertight solid.
func TestFilletCrossG2BuildsValidSolid(t *testing.T) {
	t.Parallel()
	fs, keys := boxAndVerticalEdges(t)
	pf := NewDressUpFeatures(fs).AddFilletDef(&FilletDefinition{EdgeKeys: [][]byte{keys[0]}, Radius: angleConst(0.5), CornerType: types.FilletCornerMiter, CrossSection: FilletG2})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("G2 fillet sick: %+v", pf.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("G2 fillet not a valid solid: %+v", r)
	}
}

// TestFilletCrossConicFullness: a fuller conic (higher rho) leaves more material than a flatter one.
func TestFilletCrossConicFullness(t *testing.T) {
	t.Parallel()
	vol := func(rho float64) float64 {
		fs, keys := boxAndVerticalEdges(t)
		NewDressUpFeatures(fs).AddFilletDef(&FilletDefinition{EdgeKeys: [][]byte{keys[0]}, Radius: angleConst(0.5), CornerType: types.FilletCornerMiter, CrossSection: FilletConic, Rho: rho})
		fs.Recompute()
		return ops.BodyGeometryProperties(fs.Result()[0], ops.Quality{ChordTolerance: 1e-3}).Volume
	}
	if flat, full := vol(0.3), vol(0.7); full <= flat {
		t.Errorf("conic fullness not monotonic in rho: 0.3→%g, 0.7→%g", flat, full)
	}
}

// TestFilletCrossRoundTrip: the cross-section and rho survive a recipe save/restore.
func TestFilletCrossRoundTrip(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewDressUpFeatures(fs).AddFilletDef(&FilletDefinition{EdgeKeys: [][]byte{[]byte("edge/x")}, Radius: angleConst(0.5), CornerType: types.FilletCornerMiter, CrossSection: FilletConic, Rho: 0.65})
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	got := fresh.Item(0).Definition().(*FilletFeature).Definition()
	if got.CrossSection != FilletConic || got.Rho != 0.65 {
		t.Errorf("restored cross-section = %v rho %g, want conic(2) 0.65", got.CrossSection, got.Rho)
	}
}
