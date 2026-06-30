// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
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
	fs = NewPartFeatures(nil)
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

// TestFullRoundRejectsDegenerateSide: reusing the center face as a side cannot resolve a round — a
// clean error (the would-be sides share the center's plane, so there is no enclosed round).
func TestFullRoundRejectsDegenerateSide(t *testing.T) {
	fs, s1, ctr, _ := ribForFullRound(t)
	pf := NewDressUpFeatures(fs).AddFullRoundFillet(s1, ctr, ctr) // side2 == center
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("full round with the center reused as a side should be sick")
	}
}

// TestFullRoundConvergingSides: a trapezoidal rib (sides converge upward) full-rounds the narrow top
// between the two slanted sides (#694). The round is tangent to both sides with its apex on the
// original top plane — a valid solid whose volume DROPS (corners rounded off) and whose apex stays at
// the original top height (no upward bulge). The round is faceted (the boolean planarizes it).
func TestFullRoundConvergingSides(t *testing.T) {
	trap := []math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 3, Y: 2}, {X: 1, Y: 2}}
	rib := buildPrism(trap, sketch.XYPlane(), span{near: 0, far: 6}, 0, "rib")
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(rib)
	orig := ops.BodyGeometryProperties(rib, ops.DefaultQuality()).Volume

	top := faceKeyByNormal(t, rib, math.V3(0, 1, 0)) // narrow +Y top = center
	pf := NewDressUpFeatures(fs).AddFullRoundFillet(
		[][]byte{slantedSideKey(t, rib, -1)}, [][]byte{top}, [][]byte{slantedSideKey(t, rib, +1)})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("converging full round sick: %+v", pf.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("converging full round not a valid solid: %+v", r)
	}
	v := ops.BodyGeometryProperties(res, ops.Quality{ChordTolerance: 1e-3}).Volume
	if v >= orig {
		t.Errorf("converging full round volume = %g, want < %g (corners rounded off)", v, orig)
	}
	if maxY := maxVertexY(res); maxY > 2.0001 {
		t.Errorf("round apex maxY = %g, want ≈2 (no bulge above the original top)", maxY)
	}
}

// slantedSideKey returns the key of the trapezoid rib's slanted side whose normal has the given X sign.
func slantedSideKey(t *testing.T, b *topo.Body, sign float64) []byte {
	t.Helper()
	for _, f := range b.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok {
			continue
		}
		n := pl.Normal()
		if float64(n.Dot(math.V3(0, 1, 0))) > 0.2 && float64(n.X)*sign > 0.2 {
			return f.ReferenceKey()
		}
	}
	t.Fatalf("no slanted side with X sign %g", sign)
	return nil
}

// maxVertexY returns the highest vertex Y over all the body's faces.
func maxVertexY(b *topo.Body) float64 {
	hi := stdmath.Inf(-1)
	for _, f := range b.Faces() {
		for _, v := range f.Vertices() {
			hi = stdmath.Max(hi, float64(v.Point().Y))
		}
	}
	return hi
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
