// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// ribForFullRound builds a 4×10×5 rib and returns the engine plus the side/center face keys (the two
// vertical x faces and the top z=5 face).
func ribForFullRound(t *testing.T) (fs *PartFeatures, side1, center, side2 [][]byte) {
	t.Helper()
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 10}, {X: 0, Y: 10}},
		sketch.XYPlane(), span{near: 0, far: 5}, 0, "rib")
	for _, f := range box.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok {
			continue
		}
		n, o := pl.Normal(), pl.Origin
		switch {
		case n.X < -0.5:
			side1 = [][]byte{f.ReferenceKey()}
		case n.X > 0.5:
			side2 = [][]byte{f.ReferenceKey()}
		case n.Z > 0.5 && o.Z > 4:
			center = [][]byte{f.ReferenceKey()}
		}
	}
	fs = NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(box)
	return fs, side1, center, side2
}

// TestFullRoundRoundsRibTop: a full round between the two parallel side faces replaces the top with a
// half-round — a valid solid whose volume is the lower box plus the half-cylinder (the corners gone).
func TestFullRoundRoundsRibTop(t *testing.T) {
	fs, s1, ctr, s2 := ribForFullRound(t)
	pf := NewDressUpFeatures(fs).AddFullRoundFillet(s1, ctr, s2)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("full round went sick: %+v", pf.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("full-round body not a valid solid: %+v", r.Issues)
	}
	// rib 4×10×5 = 200; rounded = lower box 4×10×3 (120) + half-cylinder r2 len10 (≈62.8) ≈ 182.8.
	// The boolean facets the round, so the value lands a little under; bound it generously.
	v := ops.BodyGeometryProperties(res, ops.Quality{ChordTolerance: 1e-3}).Volume
	if v <= 178 || v >= 185 {
		t.Errorf("full-round volume = %g, want ≈ 182.8 (lower box + half cylinder, faceted)", v)
	}
	if v >= 200 {
		t.Error("full round should remove the top corners (volume < the original 200)")
	}
}

// TestFullRoundRejectsNonParallelSides: choosing a side and the top (not parallel) is a clean error.
func TestFullRoundRejectsNonParallelSides(t *testing.T) {
	fs, s1, ctr, _ := ribForFullRound(t)
	pf := NewDressUpFeatures(fs).AddFullRoundFillet(s1, ctr, ctr) // ctr (top) is not parallel to s1
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("full round with non-parallel sides should be sick")
	}
}

// TestFullRoundRejectsMissingFace: an unknown face key is a clean error.
func TestFullRoundRejectsMissingFace(t *testing.T) {
	fs, s1, ctr, _ := ribForFullRound(t)
	pf := NewDressUpFeatures(fs).AddFullRoundFillet(s1, ctr, [][]byte{[]byte("nope")})
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("full round with a missing side face should be sick")
	}
}
