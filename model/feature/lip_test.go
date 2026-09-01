// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestLipRaisesBeadAlongTopEdge runs a lip along a box top edge and checks it adds material
// into a valid solid; the groove variant cuts (M20-F10 #485).
func TestLipRaisesBeadAlongTopEdge(t *testing.T) {
	t.Parallel()
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "box")
	var top []byte
	for _, e := range box.Edges() {
		a, b := e.StartVertex().Point(), e.EndVertex().Point()
		if stdmath.Abs(a.Z-2) < 1e-9 && stdmath.Abs(b.Z-2) < 1e-9 && a.Z == b.Z { // a horizontal top edge
			top = e.ReferenceKey()
			break
		}
	}
	if top == nil {
		t.Fatal("no top edge found")
	}
	base := 8.0 // box volume

	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)
	lip := NewDressUpFeatures(fs).AddLip([][]byte{top}, func() float64 { return 0.3 }, func() float64 { return 0.3 }, false)
	fs.Recompute()
	if !lip.Health().OK() {
		t.Fatalf("lip sick: %+v", lip.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("lip body not a valid solid: %+v", r)
	}
	if got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; got <= base {
		t.Errorf("lip volume = %g, want > %g (lip adds material)", got, base)
	}
}

// TestLipGrooveCutsAlongTopEdge checks the groove variant removes material.
func TestLipGrooveCutsAlongTopEdge(t *testing.T) {
	t.Parallel()
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "box")
	var top []byte
	for _, e := range box.Edges() {
		a, b := e.StartVertex().Point(), e.EndVertex().Point()
		if stdmath.Abs(a.Z-2) < 1e-9 && stdmath.Abs(b.Z-2) < 1e-9 && a.Z == b.Z {
			top = e.ReferenceKey()
			break
		}
	}
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)
	g := NewDressUpFeatures(fs).AddLip([][]byte{top}, func() float64 { return 0.3 }, func() float64 { return 0.3 }, true)
	fs.Recompute()
	if !g.Health().OK() {
		t.Fatalf("groove sick: %+v", g.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("groove body not a valid solid: %+v", r)
	}
	if got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; got >= 8.0 {
		t.Errorf("groove volume = %g, want < 8 (groove removes material)", got)
	}
}

// TestLipRoundTrip checks the lip's size + groove flag survive an .obk round trip.
func TestLipRoundTrip(t *testing.T) {
	t.Parallel()
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	fs := NewPartFeatures(nil)
	NewExtrudeFeatures(fs).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 1 })
	NewDressUpFeatures(fs).AddLip([][]byte{[]byte("e0")}, func() float64 { return 0.3 }, func() float64 { return 0.5 }, true)

	data, err := fs.MarshalRecipe(oneSketch{sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	def := fresh.Item(1).Definition().(*LipFeature).Definition()
	if def.Width() != 0.3 || def.Height() != 0.5 || !def.Groove {
		t.Errorf("restored lip = (w %g, h %g, groove %v), want (0.3, 0.5, true)", def.Width(), def.Height(), def.Groove)
	}
}
