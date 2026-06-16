// SPDX-License-Identifier: GPL-2.0-only

package clientgraphics

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestMapperFromWire converts a wire color mapper and validates a malformed one (M16-F05 #641).
func TestMapperFromWire(t *testing.T) {
	m, err := MapperFromWire(wire.GraphicsColorMapper{Values: []float64{0, 1}, Colors: []float32{0, 0, 0, 1, 1, 1, 1, 1}})
	if err != nil || m == nil || len(m.Values) != 2 {
		t.Fatalf("MapperFromWire = (%+v, %v), want a 2-stop mapper", m, err)
	}
	if _, err := MapperFromWire(wire.GraphicsColorMapper{Values: []float64{0}, Colors: []float32{1, 2}}); err == nil {
		t.Error("a mapper with colors != 4*values should error")
	}
}

// TestTransformFromWire builds an identity transform and rejects a bad-length matrix.
func TestTransformFromWire(t *testing.T) {
	identity := []float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	if _, has, err := TransformFromWire(identity); err != nil || !has {
		t.Fatalf("TransformFromWire(identity) = (has %v, err %v), want (true, nil)", has, err)
	}
	if _, has, err := TransformFromWire(nil); err != nil || has {
		t.Errorf("TransformFromWire(nil) = (has %v, err %v), want (false, nil)", has, err)
	}
	if _, _, err := TransformFromWire([]float64{1, 2, 3}); err == nil {
		t.Error("a 3-element transform should error")
	}
}
