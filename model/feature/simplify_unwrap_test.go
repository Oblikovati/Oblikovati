// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// axisAligned reports whether a normal points (nearly) along ±X/±Y/±Z.
func axisAligned(n math.Vector3) bool {
	return stdmath.Abs(n.X) > 0.99 || stdmath.Abs(n.Y) > 0.99 || stdmath.Abs(n.Z) > 0.99
}

// TestSimplifyRemovesChamferLighter chamfers a box edge then simplifies away the chamfer face,
// healing back to a sharp box with fewer faces — a validated lighter body (M20-F13 #490).
func TestSimplifyRemovesChamferLighter(t *testing.T) {
	box, edge := box2(t)
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(box)
	NewDressUpFeatures(fs).AddChamfer([][]byte{edge}, func() float64 { return 0.5 })
	fs.Recompute()
	chamfered := fs.Result()[0]
	before := len(chamfered.Faces())
	var chamferFace []byte
	for _, f := range chamfered.Faces() {
		if !axisAligned(f.Geometry().NormalAt(0, 0)) {
			chamferFace = f.ReferenceKey()
			break
		}
	}
	if chamferFace == nil {
		t.Fatal("no chamfer (non-axis) face found")
	}

	simp := NewModifyFeatures(fs).AddSimplify([][]byte{chamferFace}, false)
	fs.Recompute()
	if !simp.Health().OK() {
		t.Fatalf("simplify sick: %+v", simp.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("simplified body not a valid solid: %+v", r)
	}
	if got := len(res.Faces()); got >= before {
		t.Errorf("simplified face count = %d, want < %d (lighter)", got, before)
	}
}

// TestUnwrapFlattensCylinder unwraps a cylinder's side face into a flat sheet of circumference
// × height (M20-F13 #490).
func TestUnwrapFlattensCylinder(t *testing.T) {
	const r, h = 5.0, 10.0
	fs := NewPartFeatures(nil, nil)
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	sk.Circles().AddByCenterRadius(math.P2(0, 0), r)
	NewExtrudeFeatures(fs).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return h })
	fs.Recompute()
	var cylFace []byte
	for _, f := range fs.Result()[0].Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			cylFace = f.ReferenceKey()
			break
		}
	}
	if cylFace == nil {
		t.Fatal("no cylinder face on the extruded circle")
	}

	uw := NewModifyFeatures(fs).AddUnwrap(cylFace)
	fs.Recompute()
	if !uw.Health().OK() {
		t.Fatalf("unwrap sick: %+v", uw.Health())
	}
	bodies := fs.Result()
	if len(bodies) != 2 {
		t.Fatalf("unwrap result = %d bodies, want 2 (cylinder + flat patch)", len(bodies))
	}
	patch := bodies[1]
	if patch.IsSolid() {
		t.Error("unwrapped patch is solid, want an open sheet")
	}
	box := patch.RangeBox()
	if w := box.Max.X - box.Min.X; stdmath.Abs(w-2*stdmath.Pi*r) > 1e-3 {
		t.Errorf("patch width = %g, want circumference 2πr = %g", w, 2*stdmath.Pi*r)
	}
	if ht := box.Max.Y - box.Min.Y; stdmath.Abs(ht-h) > 1e-3 {
		t.Errorf("patch height = %g, want %g", ht, h)
	}
}

// TestSimplifyUnwrapRoundTrip checks both features survive an .obk round trip.
func TestSimplifyUnwrapRoundTrip(t *testing.T) {
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	fs := NewPartFeatures(nil, nil)
	NewExtrudeFeatures(fs).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 1 })
	NewModifyFeatures(fs).AddSimplify([][]byte{[]byte("f0")}, true)
	NewModifyFeatures(fs).AddUnwrap([]byte("f1"))

	data, err := fs.MarshalRecipe(oneSketch{sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	sd := fresh.Item(1).Definition().(*SimplifyFeature).Definition()
	if len(sd.RemoveFaceKeys) != 1 || !sd.FillVoids {
		t.Errorf("restored simplify = %d faces, fillVoids %v; want 1, true", len(sd.RemoveFaceKeys), sd.FillVoids)
	}
	ud := fresh.Item(2).Definition().(*UnwrapFeature).Definition()
	if string(ud.FaceKey) != "f1" {
		t.Errorf("restored unwrap face = %q, want f1", ud.FaceKey)
	}
}
