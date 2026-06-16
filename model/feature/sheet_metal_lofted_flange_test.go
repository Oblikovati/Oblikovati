// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// lProfileOnPlane returns an open L-profile (out `w`, then up `h`) on the given plane.
func lProfileOnPlane(plane sketch.Plane, w, h float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(plane)
	p0 := s.Points().Add(math.P2(0, 0))
	p1 := s.Points().Add(math.P2(math.Scalar(w), 0))
	p2 := s.Points().Add(math.P2(math.Scalar(w), math.Scalar(h)))
	s.Lines().Add(p0, p1)
	s.Lines().Add(p1, p2)
	return s
}

// thicknessParams returns a parameter set with just the Thickness parameter.
func thicknessParams(t *testing.T) *param.Parameters {
	t.Helper()
	ps := param.NewParameters()
	mustParam(t, ps, "Thickness", "2 mm")
	return ps
}

// TestLoftedFlangeLoftsTransition a lofted flange between two L-profiles on parallel planes
// builds one valid watertight transition wall.
func TestLoftedFlangeLoftsTransition(t *testing.T) {
	planeB, _ := sketch.NewPlane(math.P3(0, 0, 3), math.V3(1, 0, 0).AsUnit(), math.V3(0, 1, 0).AsUnit())
	fs := NewPartFeatures(thicknessParams(t), nil)
	pf := NewSheetMetalLoftedFlangeFeatures(fs).Add(&SheetMetalLoftedFlangeDefinition{
		ProfileA:  lProfileOnPlane(sketch.XYPlane(), 1, 1),
		ProfileB:  lProfileOnPlane(planeB, 2, 2),
		Operation: ops.NewBody,
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("lofted flange sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if !body.IsSolid() {
		t.Fatal("lofted flange is not a solid")
	}
	if r := ops.Validate(body); !r.Valid {
		t.Fatalf("lofted flange invalid: %v", r.Issues)
	}
	if open := ops.BoundaryEdges(body); len(open) != 0 {
		t.Errorf("lofted flange not watertight: %d boundary edges", len(open))
	}
	if v := ops.BodyGeometryProperties(body, ops.Quality{ChordTolerance: 1e-3}).Volume; v <= 0 {
		t.Errorf("lofted flange volume = %g, want positive", v)
	}
}

// TestLoftedFlangeMismatchedProfiles profiles with different vertex counts cannot loft.
func TestLoftedFlangeMismatchedProfiles(t *testing.T) {
	planeB, _ := sketch.NewPlane(math.P3(0, 0, 3), math.V3(1, 0, 0).AsUnit(), math.V3(0, 1, 0).AsUnit())
	threePt := lProfileOnPlane(planeB, 2, 2) // 3 vertices
	// A two-vertex profile (single segment) on XY.
	twoPt := sketch.NewSketches().Add(sketch.XYPlane())
	a := twoPt.Points().Add(math.P2(0, 0))
	b := twoPt.Points().Add(math.P2(1, 0))
	twoPt.Lines().Add(a, b)

	fs := NewPartFeatures(thicknessParams(t), nil)
	pf := NewSheetMetalLoftedFlangeFeatures(fs).Add(&SheetMetalLoftedFlangeDefinition{
		ProfileA: twoPt, ProfileB: threePt, Operation: ops.NewBody,
	})
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("lofting mismatched-vertex profiles should be sick")
	}
}

// TestLoftedFlangeNeedsThickness without a Thickness parameter the feature goes sick.
func TestLoftedFlangeNeedsThickness(t *testing.T) {
	fs := NewPartFeatures(param.NewParameters(), nil) // no Thickness
	pf := NewSheetMetalLoftedFlangeFeatures(fs).Add(&SheetMetalLoftedFlangeDefinition{
		ProfileA: lProfileOnPlane(sketch.XYPlane(), 1, 1), ProfileB: lProfileOnPlane(sketch.XYPlane(), 1, 1),
	})
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("lofted flange without a Thickness parameter should be sick")
	}
}

// TestLoftedFlangeDefinitionAndKind the accessors return the recipe.
func TestLoftedFlangeDefinitionAndKind(t *testing.T) {
	def := &SheetMetalLoftedFlangeDefinition{}
	f := &SheetMetalLoftedFlangeFeature{def: def}
	if f.Definition() != def || f.Kind() != "sheet-metal-lofted-flange" {
		t.Error("Definition/Kind mismatch")
	}
}

// twoSketchIndexer is a SketchIndexer over two sketches at indices 0 and 1.
type twoSketchIndexer struct{ a, b *sketch.Sketch }

func (x twoSketchIndexer) IndexOf(s *sketch.Sketch) (int, bool) {
	switch s {
	case x.a:
		return 0, true
	case x.b:
		return 1, true
	}
	return 0, false
}

func (x twoSketchIndexer) At(i int) (*sketch.Sketch, bool) {
	switch i {
	case 0:
		return x.a, true
	case 1:
		return x.b, true
	}
	return nil, false
}

// TestLoftedFlangeRoundTrip the recipe (two profile indices + operation) marshals and restores.
func TestLoftedFlangeRoundTrip(t *testing.T) {
	pa := lProfileOnPlane(sketch.XYPlane(), 1, 1)
	pb := lProfileOnPlane(sketch.XYPlane(), 2, 2)
	idx := twoSketchIndexer{a: pa, b: pb}
	fs := NewPartFeatures(nil, nil)
	NewSheetMetalLoftedFlangeFeatures(fs).Add(&SheetMetalLoftedFlangeDefinition{ProfileA: pa, ProfileB: pb, Operation: ops.Join})

	data, err := fs.MarshalRecipe(idx)
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	d := data[0].SheetMetalLoftedFlange
	if data[0].Kind != "sheet-metal-lofted-flange" || d == nil {
		t.Fatalf("marshaled = %+v, want sheet-metal-lofted-flange", data[0])
	}
	if d.ProfileA != 0 || d.ProfileB != 1 || d.Operation != int32(ops.Join) {
		t.Errorf("payload = %+v, want profiles 0/1 / join", d)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, idx, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 1 || fresh.Item(0).Kind() != "sheet-metal-lofted-flange" {
		t.Errorf("restored = %d features, want one lofted flange", fresh.Count())
	}
}

// TestLoftedFlangeMissingPayload / unknown sketch restore errors.
func TestLoftedFlangeMissingPayload(t *testing.T) {
	if _, err := restoreSheetMetalLoftedFlange(NewPartFeatures(nil, nil), nil, twoSketchIndexer{}); err == nil {
		t.Error("restoreSheetMetalLoftedFlange(nil) must error")
	}
	if _, err := serializeSheetMetalLoftedFlange(&SheetMetalLoftedFlangeDefinition{ProfileA: lProfileOnPlane(sketch.XYPlane(), 1, 1)}, twoSketchIndexer{}); err == nil {
		t.Error("serialize with an unknown profile A must error")
	}
}
