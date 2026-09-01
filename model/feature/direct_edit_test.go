// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/health"
)

// The consolidated direct edit (M09-F04 PBI-108, #332): one feature, five
// operations, each pinned by an analytic volume.

// directEditBox builds a 4×4×4 box engine and returns the face key whose
// outward normal matches dir.
func directEditBox(t *testing.T, dir math.Vector3) (*PartFeatures, []byte) {
	t.Helper()
	box := boxBody()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)
	for _, f := range box.Faces() {
		if pl, ok := f.Geometry().(geom.Plane); ok && float64(pl.Normal().Dot(dir)) > 0.9 {
			return fs, f.ReferenceKey()
		}
	}
	t.Fatal("no face with the requested normal")
	return nil, nil
}

func directEditVolume(t *testing.T, fs *PartFeatures, def *DirectEditDefinition) float64 {
	t.Helper()
	pf := NewModifyFeatures(fs).AddDirectEdit(def)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("directEdit %v sick: %+v", def.Operation, pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("directEdit %v result not a valid solid: %+v", def.Operation, r)
	}
	return ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume
}

func TestDirectEditMovePushesFace(t *testing.T) {
	t.Parallel()
	fs, top := directEditBox(t, math.V3(0, 0, 1))
	got := directEditVolume(t, fs, &DirectEditDefinition{
		Operation: types.DirectEditMoveOperation, FaceKeys: [][]byte{top}, Translation: math.V3(0, 0, 1),
	})
	if stdmath.Abs(got-80) > 1e-9 { // 4×4×5
		t.Errorf("move volume = %g, want 80", got)
	}
}

func TestDirectEditSizePushesAlongDirection(t *testing.T) {
	t.Parallel()
	fs, side := directEditBox(t, math.V3(1, 0, 0))
	got := directEditVolume(t, fs, &DirectEditDefinition{
		Operation: types.DirectEditSizeOperation, FaceKeys: [][]byte{side},
		Direction: math.V3(2, 0, 0), Distance: constFloat(0.5), // direction normalizes
	})
	if stdmath.Abs(got-72) > 1e-9 { // 4.5×4×4
		t.Errorf("size volume = %g, want 72", got)
	}
}

func TestDirectEditRotateTiltsFace(t *testing.T) {
	t.Parallel()
	fs, top := directEditBox(t, math.V3(0, 0, 1))
	const theta = 0.1
	got := directEditVolume(t, fs, &DirectEditDefinition{
		Operation: types.DirectEditRotateOperation, FaceKeys: [][]byte{top},
		AxisPoint: math.P3(0, 0, 4), AxisDir: math.V3(0, 1, 0), Angle: constFloat(theta),
	})
	// Top plane tilts to z = 4 − x·tanθ over x,y ∈ [0,4]: removed = ∫∫ x·tanθ = 32·tanθ.
	want := 64 - 32*stdmath.Tan(theta)
	if stdmath.Abs(got-want) > 1e-9 {
		t.Errorf("rotate volume = %g, want %g", got, want)
	}
}

func TestDirectEditScaleAboutPoint(t *testing.T) {
	t.Parallel()
	fs, _ := directEditBox(t, math.V3(0, 0, 1))
	got := directEditVolume(t, fs, &DirectEditDefinition{
		Operation: types.DirectEditScaleOperation, ScaleFactor: constFloat(1.5), BasePoint: math.P3(2, 2, 2),
	})
	if stdmath.Abs(got-64*1.5*1.5*1.5) > 1e-9 {
		t.Errorf("scale volume = %g, want %g", got, 64*1.5*1.5*1.5)
	}
}

func TestDirectEditDeleteDispatches(t *testing.T) {
	t.Parallel()
	// Delete reuses the same kernel op the standalone delete-face feature pins
	// (healable cases live in its tests); here the dispatch path must go Sick
	// on an unhealable pick — a box cap has nowhere to heal to — instead of
	// corrupting the body.
	fs, top := directEditBox(t, math.V3(0, 0, 1))
	pf := NewModifyFeatures(fs).AddDirectEdit(&DirectEditDefinition{
		Operation: types.DirectEditDeleteOperation, FaceKeys: [][]byte{top},
	})
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Errorf("deleting a box cap = %v, want Sick (no healable neighbours)", pf.Health().Status)
	}
}

func TestDirectEditUnknownOperationSick(t *testing.T) {
	t.Parallel()
	fs, top := directEditBox(t, math.V3(0, 0, 1))
	pf := NewModifyFeatures(fs).AddDirectEdit(&DirectEditDefinition{
		Operation: types.DirectEditUnknownOperation, FaceKeys: [][]byte{top},
	})
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Errorf("unknown operation = %v, want Sick", pf.Health().Status)
	}
}

func TestDirectEditRoundTrip(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewModifyFeatures(fs).AddDirectEdit(&DirectEditDefinition{
		Operation: types.DirectEditRotateOperation, FaceKeys: [][]byte{[]byte("f")},
		AxisPoint: math.P3(0, 0, 4), AxisDir: math.V3(0, 1, 0), Angle: constFloat(0.1),
	})
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if data[0].DirectEdit == nil || data[0].DirectEdit.Operation != "rotate" {
		t.Fatalf("serialized = %+v, want a rotate directEdit", data[0].DirectEdit)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	def := fresh.Item(0).Definition().(*DirectEditFeature).Definition()
	if def.Operation != types.DirectEditRotateOperation || def.Angle() != 0.1 || def.AxisDir.Y != 1 {
		t.Errorf("restored = %+v, want the rotate definition back", def)
	}
}
