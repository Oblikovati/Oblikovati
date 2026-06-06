// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/math"
	"oblikovati/model/sketch"
)

// squareOn builds a sketch on the given plane with a side×side square at offset (dx,dx).
func squareOn(plane sketch.Plane, side, dx float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(plane)
	c0 := s.Points().Add(math.P2(math.Scalar(dx), math.Scalar(dx)))
	c1 := s.Points().Add(math.P2(math.Scalar(dx+side), math.Scalar(dx)))
	c2 := s.Points().Add(math.P2(math.Scalar(dx+side), math.Scalar(dx+side)))
	c3 := s.Points().Add(math.P2(math.Scalar(dx), math.Scalar(dx+side)))
	s.Lines().Add(c0, c1)
	s.Lines().Add(c1, c2)
	s.Lines().Add(c2, c3)
	s.Lines().Add(c3, c0)
	return s
}

// embossedBlock builds a 10×10×2 block (vol 200) and returns the engine for an emboss test.
func embossedBlock(t *testing.T) *PartFeatures {
	t.Helper()
	fs := NewPartFeatures(nil, nil)
	NewExtrudeFeatures(fs).AddByDistanceExtent(squareSketch(10), 0, ops.NewBody, func() float64 { return 2 })
	return fs
}

// A raised emboss adds a 4×4×1 boss on the top face → block volume + 16.
func TestEmbossRaisesMaterial(t *testing.T) {
	fs := embossedBlock(t)
	es := squareOn(planeAtZ(2), 4, 3) // 4×4 square centred-ish on the z=2 top
	emb := NewEmbossFeatures(fs).Add(es, []int{0}, func() float64 { return 1 }, false, 0)
	fs.Recompute()
	if !emb.Health().OK() {
		t.Fatalf("emboss went sick: %+v", emb.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("embossed body not valid: %+v", r)
	}
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; stdmath.Abs(v-216) > 1e-6 {
		t.Errorf("raised emboss volume = %g, want 216 (200 + 4×4×1)", v)
	}
}

// An engraved emboss cuts a 4×4×1 pocket into the top face → block volume − 16.
func TestEmbossEngravesMaterial(t *testing.T) {
	fs := embossedBlock(t)
	es := squareOn(planeAtZ(2), 4, 3)
	emb := NewEmbossFeatures(fs).Add(es, []int{0}, func() float64 { return 1 }, true, 0) // engrave
	fs.Recompute()
	if !emb.Health().OK() {
		t.Fatalf("engrave went sick: %+v", emb.Health())
	}
	if v := ops.BodyGeometryProperties(fs.Result()[0], ops.DefaultQuality()).Volume; stdmath.Abs(v-184) > 1e-6 {
		t.Errorf("engraved volume = %g, want 184 (200 − 4×4×1)", v)
	}
}

func TestEmbossNeedsDepthAndProfile(t *testing.T) {
	fs := embossedBlock(t)
	es := squareOn(planeAtZ(2), 4, 3)
	emb := NewEmbossFeatures(fs).Add(es, nil, func() float64 { return 1 }, false, 0) // no profile
	fs.Recompute()
	if emb.Health().OK() {
		t.Error("emboss with no profile should be sick")
	}
}
