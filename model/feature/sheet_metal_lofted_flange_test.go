// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
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
	fs := NewPartFeatures(thicknessParams(t))
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

// bulgingLoftBundle is a curved die-formed bundle: two offset square bands on parallel planes, so
// the Hermite curves bow (their tangents leave both planes along +Z while the points shift in-plane).
func bulgingLoftBundle() []hermiteCurve {
	up := math.V3(0, 0, 1).AsUnit()
	bandA := []math.Point3{math.P3(0, 0, 0), math.P3(2, 0, 0), math.P3(2, 2, 0), math.P3(0, 2, 0)}
	bandB := []math.Point3{math.P3(1, 1, 3), math.P3(3, 1, 3), math.P3(3, 3, 3), math.P3(1, 3, 3)}
	return hermiteBundle(bandA, bandB, up, up)
}

// TestLoftSectionCountTightensWithTolerance the die-formed wall is fixed and fine; a press-brake
// chord-tolerance output uses FEWER facets as the tolerance loosens, and never more than die-formed.
func TestLoftSectionCountTightensWithTolerance(t *testing.T) {
	bundle := bulgingLoftBundle()
	if got := loftSectionCount(bundle, DieFormedLoftedFlange, 0); got != dieFormedSections {
		t.Errorf("die-formed section count = %d, want %d", got, dieFormedSections)
	}
	loose := loftSectionCount(bundle, PressBrakeChordToleranceLoftedFlange, 0.20)
	tight := loftSectionCount(bundle, PressBrakeChordToleranceLoftedFlange, 0.02)
	if loose >= tight {
		t.Errorf("press-brake facets did not tighten: loose(%d) should be < tight(%d)", loose, tight)
	}
	if tight > dieFormedSections {
		t.Errorf("press-brake facets %d exceed die-formed %d", tight, dieFormedSections)
	}
	// The measured chord deviation of the chosen facet count must actually meet the tolerance.
	if e := facetError(bundle, tight, PressBrakeChordToleranceLoftedFlange); e > 0.02 {
		t.Errorf("chosen facet count leaves sagitta %.5f > tol 0.02", e)
	}
}

// TestLoftSectionCountStraightBundle a bundle with no in-plane offset is already straight, so a
// press-brake output needs a single facet (no curvature to approximate).
func TestLoftSectionCountStraightBundle(t *testing.T) {
	up := math.V3(0, 0, 1).AsUnit()
	bandA := []math.Point3{math.P3(0, 0, 0), math.P3(2, 0, 0), math.P3(0, 2, 0)}
	bandB := []math.Point3{math.P3(0, 0, 3), math.P3(2, 0, 3), math.P3(0, 2, 3)} // pure +Z translation
	bundle := hermiteBundle(bandA, bandB, up, up)
	if got := loftSectionCount(bundle, PressBrakeChordToleranceLoftedFlange, 0.01); got != 1 {
		t.Errorf("straight bundle press-brake count = %d, want 1 (no curvature)", got)
	}
}

// TestLoftedFlangePressBrakeCoarserThanDieFormed both output forms build a valid watertight wall,
// and a loose press-brake facets the transition into fewer faces than the smooth die-formed wall.
func TestLoftedFlangePressBrakeCoarserThanDieFormed(t *testing.T) {
	build := func(out LoftedFlangeOutputType, tol float64) *topo.Body {
		planeB, _ := sketch.NewPlane(math.P3(1, 1, 3), math.V3(1, 0, 0).AsUnit(), math.V3(0, 1, 0).AsUnit())
		fs := NewPartFeatures(thicknessParams(t))
		pf := NewSheetMetalLoftedFlangeFeatures(fs).Add(&SheetMetalLoftedFlangeDefinition{
			ProfileA: lProfileOnPlane(sketch.XYPlane(), 1, 1), ProfileB: lProfileOnPlane(planeB, 2, 2),
			Operation: ops.NewBody, Output: out, FacetTolerance: tol,
		})
		fs.Recompute()
		if !pf.Health().OK() {
			t.Fatalf("lofted flange (%s) sick: %+v", out, pf.Health())
		}
		body := fs.Result()[0]
		if r := ops.Validate(body); !r.Valid {
			t.Fatalf("lofted flange (%s) invalid: %v", out, r.Issues)
		}
		if open := ops.BoundaryEdges(body); len(open) != 0 {
			t.Errorf("lofted flange (%s) not watertight: %d boundary edges", out, len(open))
		}
		return body
	}
	die := build(DieFormedLoftedFlange, 0)
	press := build(PressBrakeFacetDistanceLoftedFlange, 1.0) // wide facets → few
	if nd, np := len(die.Faces()), len(press.Faces()); np >= nd {
		t.Errorf("press-brake wall has %d faces, die-formed %d; press-brake should be coarser", np, nd)
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

	fs := NewPartFeatures(thicknessParams(t))
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
	fs := NewPartFeatures(param.NewParameters()) // no Thickness
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
	fs := NewPartFeatures(nil)
	NewSheetMetalLoftedFlangeFeatures(fs).Add(&SheetMetalLoftedFlangeDefinition{
		ProfileA: pa, ProfileB: pb, Operation: ops.Join, Output: PressBrakeFacetAngleLoftedFlange,
		FacetTolerance: 0.1, Converge: true, Radius: constFloat(0.2),
	})

	data, err := fs.MarshalRecipe(idx)
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	d := data[0].SheetMetalLoftedFlange
	if data[0].Kind != "sheet-metal-lofted-flange" || d == nil {
		t.Fatalf("marshaled = %+v, want sheet-metal-lofted-flange", data[0])
	}
	if d.ProfileA != 0 || d.ProfileB != 1 || d.Operation != int32(ops.Join) ||
		d.Output != int32(PressBrakeFacetAngleLoftedFlange) || d.FacetTolerance != 0.1 || !d.Converge || d.Radius != 0.2 {
		t.Errorf("payload = %+v, want profiles 0/1 / join / facetAngle / 0.1 / converge / 0.2", d)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, idx, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	got := fresh.Item(0).Definition().(*SheetMetalLoftedFlangeFeature).Definition()
	if got.Output != PressBrakeFacetAngleLoftedFlange || got.FacetTolerance != 0.1 || !got.Converge || evalFloat(got.Radius) != 0.2 {
		t.Errorf("restored = %+v, want facetAngle / 0.1 / converge / 0.2", got)
	}
}

// TestLoftedFlangeMissingPayload / unknown sketch restore errors.
func TestLoftedFlangeMissingPayload(t *testing.T) {
	if _, err := restoreSheetMetalLoftedFlange(NewPartFeatures(nil), nil, twoSketchIndexer{}); err == nil {
		t.Error("restoreSheetMetalLoftedFlange(nil) must error")
	}
	if _, err := serializeSheetMetalLoftedFlange(&SheetMetalLoftedFlangeDefinition{ProfileA: lProfileOnPlane(sketch.XYPlane(), 1, 1)}, twoSketchIndexer{}); err == nil {
		t.Error("serialize with an unknown profile A must error")
	}
}
