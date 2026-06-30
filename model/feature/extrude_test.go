// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/health"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// squareSketch returns a sketch with one closed square profile of the given side.
func squareSketch(side float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	c0 := s.Points().Add(math.P2(0, 0))
	c1 := s.Points().Add(math.P2(side, 0))
	c2 := s.Points().Add(math.P2(side, side))
	c3 := s.Points().Add(math.P2(0, side))
	s.Lines().Add(c0, c1)
	s.Lines().Add(c1, c2)
	s.Lines().Add(c2, c3)
	s.Lines().Add(c3, c0)
	return s
}

func TestExtrudeCreatesValidSolid(t *testing.T) {
	ps := param.NewParameters()
	h, _ := ps.AddUserParameter("height", "5 cm")
	fs := NewPartFeatures(ps)
	extrudes := NewExtrudeFeatures(fs)
	pf := extrudes.AddByDistanceExtent(squareSketch(2), 0, ops.NewBody, func() float64 { return h.ModelValue() })

	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("extrude went sick: %+v", pf.Health())
	}
	bodies := fs.Result()
	if len(bodies) != 1 {
		t.Fatalf("result has %d bodies, want 1", len(bodies))
	}
	body := bodies[0]
	if !body.IsSolid() {
		t.Error("extrude result is not a solid")
	}
	// A square extrusion → 6 faces (4 sides + 2 caps), and watertight.
	if len(body.Faces()) != 6 {
		t.Errorf("prism has %d faces, want 6", len(body.Faces()))
	}
	if open := ops.BoundaryEdges(body); len(open) != 0 {
		t.Errorf("prism has %d boundary edges, want 0 (watertight)", len(open))
	}
	if r := ops.Validate(body); !r.Valid {
		t.Errorf("prism failed validation: %+v", r)
	}
	// Bounding box spans the 2×2 base and the 5 height.
	box := body.RangeBox()
	if d := box.Diagonal(); !approxEq(d.X, 2) || !approxEq(d.Y, 2) || !approxEq(d.Z, 5) {
		t.Errorf("prism box diagonal = %v, want (2,2,5)", d)
	}
}

func TestExtrudeRecomputesOnParameterChange(t *testing.T) {
	ps := param.NewParameters()
	h, _ := ps.AddUserParameter("height", "5 cm")
	fs := NewPartFeatures(ps)
	extrudes := NewExtrudeFeatures(fs)
	extrudes.AddByDistanceExtent(squareSketch(2), 0, ops.NewBody, func() float64 { return h.ModelValue() })
	fs.Recompute()
	if z := fs.Result()[0].RangeBox().Diagonal().Z; !approxEq(z, 5) {
		t.Fatalf("initial height = %v, want 5", z)
	}
	// Edit the driving parameter and re-run: the solid grows.
	if err := h.SetExpression("9 cm"); err != nil {
		t.Fatalf("SetExpression: %v", err)
	}
	fs.MarkDirty(fs.Item(0))
	fs.Recompute()
	if z := fs.Result()[0].RangeBox().Diagonal().Z; !approxEq(z, 9) {
		t.Errorf("height after param change = %v, want 9", z)
	}
}

func TestExtrudeGoesSickOnOpenProfile(t *testing.T) {
	// A sketch with an open chain has no closed profile to extrude into a solid.
	s := sketch.NewSketches().Add(sketch.XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(1, 0))
	c := s.Points().Add(math.P2(2, 1))
	s.Lines().Add(a, b)
	s.Lines().Add(b, c) // open

	fs := NewPartFeatures(nil)
	pf := NewExtrudeFeatures(fs).AddByDistanceExtent(s, 0, ops.NewBody, func() float64 { return 5 })
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Errorf("extrude of an open profile = %v, want sick", pf.Health().Status)
	}
}

func TestUngeneratedFeaturesGoSick(t *testing.T) {
	fs := NewPartFeatures(nil)
	sk := squareSketch(1)
	feats := []*PartFeature{
		NewRevolveFeatures(fs).Add(sk, 0, nil, nil, ops.Join),
		NewRevolveFeatures(fs).engine.Add(&SweepFeature{def: &SweepDefinition{Sketch: sk}}),
		fs.Add(&LoftFeature{def: &LoftDefinition{}}),
		fs.Add(&CoilFeature{def: &CoilDefinition{Sketch: sk}}),
		fs.Add(&RibFeature{def: &RibDefinition{Sketch: sk}}),
	}
	fs.Recompute()
	// Revolve/coil/sweep/loft now generate geometry but go sick here on missing inputs
	// (no axis / path / sections); rib is still generation-deferred.
	for _, pf := range feats {
		if pf.Health().Status != health.Sick {
			t.Errorf("%s should be sick (missing inputs or deferred), got %v", pf.Kind(), pf.Health().Status)
		}
	}
	// Definitions are still accessible (the triangle/API is complete).
	if NewRevolveFeatures(fs).Add(sk, 0, nil, nil, ops.Join).Kind() != "revolve" {
		t.Error("revolve kind wrong")
	}
}

func TestExtrudesGetUniqueInventorNames(t *testing.T) {
	fs := NewPartFeatures(nil)
	ex := NewExtrudeFeatures(fs)
	a := ex.AddByDistanceExtent(squareSketch(2), 0, ops.NewBody, func() float64 { return 3 })
	b := ex.AddByDistanceExtent(squareSketchAt(2, 10), 0, ops.NewBody, func() float64 { return 3 })
	// Distinct names keep the browser rows — and Dear ImGui's per-node ids — unique.
	if a.Name() != "Extrusion1" || b.Name() != "Extrusion2" {
		t.Errorf("extrude names = %q, %q; want Extrusion1, Extrusion2", a.Name(), b.Name())
	}
}

func TestExtrudeSetDistanceAndOperationDriveRecompute(t *testing.T) {
	fs := NewPartFeatures(nil)
	pf := NewExtrudeFeatures(fs).AddByDistanceExtent(squareSketch(2), 0, ops.NewBody, func() float64 { return 5 })
	ext := pf.Definition().(*ExtrudeFeature)
	fs.Recompute()
	if z := fs.Result()[0].RangeBox().Diagonal().Z; !approxEq(z, 5) {
		t.Fatalf("initial height = %v, want 5", z)
	}
	if ext.DistanceValue() != 5 || ext.Operation() != ops.NewBody {
		t.Fatalf("read back distance/op = %v/%v, want 5/NewBody", ext.DistanceValue(), ext.Operation())
	}
	// Editing the distance through the feature (the double-click edit path) grows the solid.
	ext.SetDistance(8)
	ext.SetOperation(ops.Join)
	fs.MarkDirty(pf)
	fs.Recompute()
	if z := fs.Result()[0].RangeBox().Diagonal().Z; !approxEq(z, 8) {
		t.Errorf("height after SetDistance(8) = %v, want 8", z)
	}
	if ext.Operation() != ops.Join {
		t.Errorf("operation after SetOperation = %v, want Join", ext.Operation())
	}
}

func approxEq(a, b float64) bool { return a-b < 1e-9 && b-a < 1e-9 }

// squareSketchAt is squareSketch translated by (dx,0) so two extrudes are disjoint.
func squareSketchAt(side, dx float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	c0 := s.Points().Add(math.P2(dx, 0))
	c1 := s.Points().Add(math.P2(dx+side, 0))
	c2 := s.Points().Add(math.P2(dx+side, side))
	c3 := s.Points().Add(math.P2(dx, side))
	s.Lines().Add(c0, c1)
	s.Lines().Add(c1, c2)
	s.Lines().Add(c2, c3)
	s.Lines().Add(c3, c0)
	return s
}

func TestExtrudeJoinDisjointMergesBodies(t *testing.T) {
	fs := NewPartFeatures(nil)
	ex := NewExtrudeFeatures(fs)
	ex.AddByDistanceExtent(squareSketch(2), 0, ops.NewBody, func() float64 { return 3 })
	ex.AddByDistanceExtent(squareSketchAt(2, 10), 0, ops.Join, func() float64 { return 3 })
	fs.Recompute()
	if len(fs.Result()) != 1 {
		t.Fatalf("join result has %d bodies, want 1 merged", len(fs.Result()))
	}
	if got := len(fs.Result()[0].Faces()); got != 12 {
		t.Errorf("merged body has %d faces, want 12 (two prisms)", got)
	}
}

func TestFeatureDefinitionsAccessible(t *testing.T) {
	fs := NewPartFeatures(nil)
	sk := squareSketch(1)
	ex := NewExtrudeFeatures(fs).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 1 })
	if ex.Definition().(*ExtrudeFeature).Definition().Operation != ops.NewBody {
		t.Error("extrude definition not accessible")
	}
	if (&RevolveFeature{def: &RevolveDefinition{Sketch: sk}}).Definition().Sketch != sk {
		t.Error("revolve definition not accessible")
	}
	if (&SweepFeature{def: &SweepDefinition{}}).Definition() == nil ||
		(&LoftFeature{def: &LoftDefinition{}}).Definition() == nil ||
		(&CoilFeature{def: &CoilDefinition{}}).Definition() == nil ||
		(&RibFeature{def: &RibDefinition{}}).Definition() == nil {
		t.Error("a deferred-feature definition is not accessible")
	}
	// An extent with no distance closure measures zero.
	if (Extent{Type: DistanceExtent}).distance() != 0 {
		t.Error("nil-distance extent should be zero")
	}
}
