// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"reflect"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// noSketchIndex is the sketch-index resolver for kinds that hold no sketch.
func noSketchIndex(*sketch.Sketch) int { return 0 }

// TestAssemblyFeatureDataRoundTrip pins every serializable assembly feature kind: its
// MarshalAssembly → RestoreAssemblyFeature → MarshalAssembly is identity, so its inputs survive a
// save/undo round-trip (#785). Sketch-bearing kinds resolve sketch index 0.
func TestAssemblyFeatureDataRoundTrip(t *testing.T) {
	sk := []*sketch.Sketch{{}} // index 0; the features only hold the pointer, not its contents
	down, _ := math.NewUnitVector3(0, 0, -1)
	yaw, _ := math.NewUnitVector3(0, 1, 0)

	hole, _ := NewAssemblyHoleFeature(math.P3(1, 2, 3), down, 0.5, 4)
	cases := map[string]Feature{
		"extrude":  NewAssemblyExtrudeFeature(sk[0], 1, ops.Cut, func() float64 { return 2.5 }),
		"revolve":  NewAssemblyRevolveFeature(sk[0], 0, NewDatumAxis(math.P3(0, 0, 0), yaw), ops.Join, func() float64 { return 1.5 }),
		"sweep":    NewAssemblySweepFeature(sk[0], 0, ops.Cut, []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0)}),
		"hole":     hole,
		"chamfer":  NewAssemblyChamferFeature([][]byte{[]byte("e0")}, func() float64 { return 0.2 }),
		"fillet":   NewAssemblyFilletFeature([][]byte{[]byte("e1")}, func() float64 { return 0.3 }),
		"moveFace": NewAssemblyMoveFaceFeature([][]byte{[]byte("f0")}, math.V3(0, 0, 1)),
	}
	for name, f := range cases {
		m, ok := f.(AssemblyFeatureMarshaler)
		if !ok {
			t.Errorf("%s: does not implement AssemblyFeatureMarshaler", name)
			continue
		}
		data := m.MarshalAssembly(noSketchIndex)
		restored, err := RestoreAssemblyFeature(data, sk)
		if err != nil {
			t.Errorf("%s: restore: %v", name, err)
			continue
		}
		again := restored.(AssemblyFeatureMarshaler).MarshalAssembly(noSketchIndex)
		if !reflect.DeepEqual(data, again) {
			t.Errorf("%s: data did not round-trip:\n have %+v\n want %+v", name, again, data)
		}
	}
}

// TestRestoreAssemblyFeatureRejectsUnknownKind / out-of-range sketch are the corrupt-recipe guards.
func TestRestoreAssemblyFeatureRejectsBadData(t *testing.T) {
	if _, err := RestoreAssemblyFeature(AssemblyFeatureData{Kind: "nope"}, nil); err == nil {
		t.Error("an unknown kind should be rejected")
	}
	if _, err := RestoreAssemblyFeature(AssemblyFeatureData{Kind: "assemblyExtrude", SketchIndex: 5}, nil); err == nil {
		t.Error("an out-of-range sketch index should be rejected")
	}
}
