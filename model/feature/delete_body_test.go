// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
)

// TestDeleteBodyDropsTheReferencedBody: with two separate bodies, a delete-body feature keyed by
// one body's reference key leaves exactly the other (#1078).
func TestDeleteBodyDropsTheReferencedBody(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	NewExtrudeFeatures(fs).AddByDistanceExtent(squareSketch(4), 0, ops.NewBody, func() float64 { return 5 })
	NewExtrudeFeatures(fs).AddByDistanceExtent(squareSketchAt(4, 10), 0, ops.NewBody, func() float64 { return 5 })
	fs.Recompute()
	if len(fs.Result()) != 2 {
		t.Fatalf("setup: got %d bodies, want 2", len(fs.Result()))
	}
	doomedKey := fs.Result()[0].ReferenceKey()
	keptKey := fs.Result()[1].ReferenceKey()

	pf := fs.AddDeleteBody(doomedKey)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("delete-body went sick: %s", pf.Health().Reason)
	}

	res := fs.Result()
	if len(res) != 1 {
		t.Fatalf("after delete = %d bodies, want 1", len(res))
	}
	if string(res[0].ReferenceKey()) != string(keptKey) {
		t.Error("delete-body removed the wrong body")
	}
}

// TestDeleteBodyMissingTargetGoesSick: a delete-body feature whose target key matches no running
// body goes Sick rather than silently deleting nothing.
func TestDeleteBodyMissingTargetGoesSick(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	NewExtrudeFeatures(fs).AddByDistanceExtent(squareSketch(4), 0, ops.NewBody, func() float64 { return 5 })
	fs.AddDeleteBody([]byte("no-such-body-key"))
	fs.Recompute()

	last := fs.Item(fs.Count() - 1)
	if last.Health().OK() {
		t.Error("delete-body with an unknown target key should go Sick")
	}
	if len(fs.Result()) != 1 {
		t.Errorf("a sick delete left %d bodies, want the 1 original untouched", len(fs.Result()))
	}
}

// TestDeleteBodySerializeRoundTrip pins the recipe round-trip: a delete-body feature survives
// MarshalRecipe → ApplyRecipe with its body key intact.
func TestDeleteBodySerializeRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	key := []byte{0x01, 0x02, 0x03, 0x04}
	fs.AddDeleteBody(key)

	data, err := fs.MarshalRecipe(emptySketches{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if len(data) != 1 || data[0].Kind != "delete-body" || data[0].DeleteBody == nil {
		t.Fatalf("marshaled = %+v, want one delete-body with payload", data)
	}

	restored := NewPartFeatures(nil, nil)
	if err := restored.ApplyRecipe(data, emptySketches{}, NewWorkGeometry()); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	dbf, ok := restored.Item(0).Definition().(*DeleteBodyFeature)
	if !ok {
		t.Fatalf("restored feature is %T, want *DeleteBodyFeature", restored.Item(0).Definition())
	}
	if string(dbf.Definition().BodyKey) != string(key) {
		t.Errorf("restored body key = %x, want %x", dbf.Definition().BodyKey, key)
	}
}
