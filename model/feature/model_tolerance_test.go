// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/sketch"
)

// sampleToleranceDef builds a GD&T record set: a flatness frame on faceKey relative to datum A,
// and a datum label A on the same geometry.
func sampleToleranceDef(faceKey []byte) *ModelToleranceDefinition {
	return &ModelToleranceDefinition{
		Frames: []ToleranceFrame{{
			GeometryKey:    faceKey,
			Characteristic: types.CharacteristicFlatness,
			Value:          0.005, // 0.05 mm in cm
			Datums:         []string{"A"},
		}},
		Datums: []DatumLabel{{GeometryKey: faceKey, Label: "A"}},
	}
}

// TestModelTolerancePassesBodyThrough: the carrier annotates without changing the running body
// (#866 — metadata feature).
func TestModelTolerancePassesBodyThrough(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	addRect(sk, 0, 0, 2, 2)
	NewExtrudeFeatures(fs).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 2 })
	fs.Recompute()
	before := fs.Result()[0]
	faceKey := before.Faces()[0].ReferenceKey()

	mt := NewToleranceFeatures(fs).AddModelTolerance(sampleToleranceDef(faceKey))
	fs.Recompute()
	if !mt.Health().OK() {
		t.Fatalf("model tolerance sick: %+v", mt.Health())
	}
	after := fs.Result()
	if len(after) != 1 || after[0] != before {
		t.Errorf("model tolerance altered the running body: %d bodies (want 1, same pointer)", len(after))
	}
}

// TestModelToleranceRoundTrip: frames + datums survive the recipe codec (characteristic, value,
// datum labels, geometry keys).
func TestModelToleranceRoundTrip(t *testing.T) {
	faceKey := []byte("face-7")
	fs := NewPartFeatures(nil, nil)
	NewToleranceFeatures(fs).AddModelTolerance(sampleToleranceDef(faceKey))

	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	def := fresh.Item(0).Definition().(*ModelToleranceFeature).Definition()
	if len(def.Frames) != 1 || len(def.Datums) != 1 {
		t.Fatalf("restored = %d frames, %d datums; want 1, 1", len(def.Frames), len(def.Datums))
	}
	fr := def.Frames[0]
	if fr.Characteristic != types.CharacteristicFlatness || fr.Value != 0.005 ||
		len(fr.Datums) != 1 || fr.Datums[0] != "A" || string(fr.GeometryKey) != "face-7" {
		t.Errorf("restored frame = %+v, want flatness 0.005 datum A on face-7", fr)
	}
	if dl := def.Datums[0]; dl.Label != "A" || string(dl.GeometryKey) != "face-7" {
		t.Errorf("restored datum = %+v, want label A on face-7", dl)
	}
}

// TestModelToleranceUnknownCharacteristicRejected: a bad characteristic spelling errors on
// restore (no silent loss).
func TestModelToleranceUnknownCharacteristicRejected(t *testing.T) {
	d := &ModelToleranceData{Frames: []ToleranceFrameData{{Geometry: encodeKey([]byte("f0")), Characteristic: "bogus", Value: 1}}}
	if _, err := restoreModelTolerance(NewPartFeatures(nil, nil), d); err == nil {
		t.Error("expected an error for an unknown characteristic spelling")
	}
}
