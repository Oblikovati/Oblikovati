// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	stdmath "math"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/sketch"
)

// These tests cover the profileSeeds / profileSeed path: an external author selects the extruded/
// revolved region by an interior seed point (resolved by containment on the solved sketch) instead
// of ProfileIndex, which it cannot predict. The seed must pick the region that CONTAINS it — not
// the default index 0 — so the seed and the index disagree by construction.

// addRectAt draws a closed w×h rectangle with its lower-left corner at (x0,y0) — one region.
func addRectAt(sk *sketch.Sketch, x0, y0, w, h float64) {
	at := func(x, y float64) *sketch.Point { return sk.Points().Add(math.P2(x, y)) }
	corners := []*sketch.Point{at(x0, y0), at(x0+w, y0), at(x0+w, y0+h), at(x0, y0+h)}
	for i, c := range corners {
		sk.Lines().Add(c, corners[(i+1)%len(corners)])
	}
}

// twoRegionPart builds a part whose first sketch holds two DISJOINT rectangles: a big 4×3 (area 12)
// at the origin and a small 1×1 (area 1) offset to the right. Region index order is a DCEL artifact,
// so an author must select by containment, not index.
func twoRegionPart(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	d, err := s.Workspace().Add(doc.Part, "seed.obk", true)
	if err != nil {
		t.Fatalf("add document: %v", err)
	}
	def := compdef.NewPartComponentDefinition()
	d.SetContent(def)
	sk := def.Sketches().Add(sketch.XYPlane())
	addRectAt(sk, 0, 0, 4, 3) // big region, area 12
	addRectAt(sk, 6, 0, 1, 1) // small region, area 1, offset so the two are disjoint
	def.Recompute()
	return s
}

// TestExtrudeProfileSeedSelectsContainingRegion: a seed inside the small region extrudes ONLY that
// region (area 1 × 1 cm = 1 cm³), proving the host resolves the seed by containment rather than
// extruding the default region 0.
func TestExtrudeProfileSeedSelectsContainingRegion(t *testing.T) {
	s := twoRegionPart(t)
	if _, err := apply(t, s, "extrude",
		`{"sketchIndex":0,"distance":"10 mm","profileSeeds":[[6.5,0.5]]}`); err != nil {
		t.Fatalf("extrude by profileSeeds: %v", err)
	}
	if v := bodyVolume(t, s); stdmath.Abs(v-1.0) > 1e-6 {
		t.Errorf("seed-selected extrude volume = %v, want 1 (the small 1×1 region × 1 cm)", v)
	}
}

// TestExtrudeProfileSeedBeatsIndex: with the seed inside the big region but profileIndex left at 0,
// the seed wins — a second seed pointing at the big region yields area 12 × 1 = 12 cm³.
func TestExtrudeProfileSeedBeatsIndex(t *testing.T) {
	s := twoRegionPart(t)
	if _, err := apply(t, s, "extrude",
		`{"sketchIndex":0,"profileIndex":0,"distance":"10 mm","profileSeeds":[[2,1.5]]}`); err != nil {
		t.Fatalf("extrude by profileSeeds: %v", err)
	}
	if v := bodyVolume(t, s); stdmath.Abs(v-12.0) > 1e-6 {
		t.Errorf("seed-selected extrude volume = %v, want 12 (the big 4×3 region × 1 cm)", v)
	}
}
