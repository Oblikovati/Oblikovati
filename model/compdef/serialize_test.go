// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"path/filepath"
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/doc"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/model/param"
	"github.com/Oblikovati/oblikovati/model/sketch"
	"github.com/Oblikovati/oblikovati/persistence"
)

// reopenThroughStore saves ws's active document to a temp .obk and reopens it in a
// fresh, store-backed workspace — the real save→YAML→open path.
func reopenThroughStore(t *testing.T, d *doc.Document) *compdef.PartComponentDefinition {
	t.Helper()
	store := persistence.NewPackageStore()
	path := filepath.Join(t.TempDir(), "part.obk")
	saveWS := doc.NewWorkspace(store)
	// Re-register d under the store-backed workspace identity, then save.
	saved, err := compdef.AddPart(saveWS, path, true)
	if err != nil {
		t.Fatalf("AddPart(save): %v", err)
	}
	// Copy the recipe across by marshaling the source and applying onto the saved doc.
	src := d.Content().(*compdef.PartComponentDefinition)
	model, err := src.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if err := saved.Content().(*compdef.PartComponentDefinition).ApplyRecipe(model); err != nil {
		t.Fatalf("ApplyRecipe(seed): %v", err)
	}
	if err := saveWS.Save(saved); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reopened, err := doc.NewWorkspace(store).Open(path, true)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	def, ok := reopened.Content().(*compdef.PartComponentDefinition)
	if !ok {
		t.Fatalf("reopened content is %T, want *PartComponentDefinition", reopened.Content())
	}
	return def
}

// TestSketch3DSurvivesRoundTrip checks a part's 3D sketches (points, properties, and a
// geometric constraint) survive the real save→YAML→open path (M22-F01).
func TestSketch3DSurvivesRoundTrip(t *testing.T) {
	ws := doc.NewWorkspace(nil)
	d, err := compdef.AddPart(ws, "Part1", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := d.Content().(*compdef.PartComponentDefinition)
	s := def.Sketches3D().Add()
	s.SetColor("#abcdef")
	s.SetDimensionsVisible(false)
	a := s.AddPoint3D(math.P3(1, 2, 3))
	b := s.AddPoint3D(math.P3(4, 5, 6))
	s.GeometricConstraints3D().Add(sketch.NewCoincident3D(a, b))

	reopened := reopenThroughStore(t, d)

	if reopened.Sketches3D().Count() != 1 {
		t.Fatalf("3D sketch count after reopen = %d, want 1", reopened.Sketches3D().Count())
	}
	got := reopened.Sketches3D().Item(0)
	if got.Color() != "#abcdef" || got.DimensionsVisible() {
		t.Errorf("3D sketch properties after reopen = %+v", got)
	}
	if got.EntityCount() != 2 {
		t.Errorf("3D sketch entity count after reopen = %d, want 2", got.EntityCount())
	}
	if pts := got.AllPoints3D(); pts[0].Position() != math.P3(1, 2, 3) || pts[1].Position() != math.P3(4, 5, 6) {
		t.Errorf("3D point positions after reopen = %v", pts)
	}
	if got.GeometricConstraints3D().Count() != 1 {
		t.Errorf("3D constraint count after reopen = %d, want 1", got.GeometricConstraints3D().Count())
	}
}

func TestParametersSurviveRoundTrip(t *testing.T) {
	ws := doc.NewWorkspace(nil)
	d, err := compdef.AddPart(ws, "Part1", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := d.Content().(*compdef.PartComponentDefinition)
	if _, err := def.Parameters().AddUserParameter("width", "4 cm"); err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}
	if _, err := def.Parameters().AddUserParameter("height", "width / 2"); err != nil {
		t.Fatalf("AddUserParameter(height): %v", err)
	}

	reopened := reopenThroughStore(t, d)

	width, ok := reopened.Parameters().ByName("width")
	if !ok || width.Expression() != "4 cm" {
		t.Errorf("width after reopen = %v ok=%v, want expression 4 cm", width, ok)
	}
	// A dependent expression must survive too (proves creation-order restore is valid).
	height, ok := reopened.Parameters().ByName("height")
	if !ok || height.Expression() != "width / 2" {
		t.Errorf("height after reopen = %v ok=%v, want expression 'width / 2'", height, ok)
	}
	if got := reopened.Parameters().Count(); got != 2 {
		t.Errorf("parameter count after reopen = %d, want 2", got)
	}
}

// TestParameterFieldsSurviveRoundTrip exercises the extended parameter recipe: text and
// boolean parameters, comment/key/export/tolerance/multi-value state, and group membership.
func TestParameterFieldsSurviveRoundTrip(t *testing.T) {
	ws := doc.NewWorkspace(nil)
	d, _ := compdef.AddPart(ws, "Part1", true)
	ps := d.Content().(*compdef.PartComponentDefinition).Parameters()

	num, _ := ps.AddUserParameter("len", "10 mm")
	num.Comment, num.IsKey, num.ExposedAsProperty = "the length", true, true
	num.SetTolerance(param.Tolerance{Upper: 0.01, Lower: -0.02, Type: param.Median})
	_ = num.SetExpressionList([]string{"10 mm", "20 mm"}, true)
	txt, _ := ps.AddTextUserParameter("material", "steel")
	flag, _ := ps.AddBooleanUserParameter("vented", true)
	_ = ps.AddToGroup(num.ID(), "Frame")
	_ = ps.AddToGroup(txt.ID(), "Frame")

	r := reopenThroughStore(t, d).Parameters()

	rl, _ := r.ByName("len")
	if rl.Comment != "the length" || !rl.IsKey || !rl.ExposedAsProperty {
		t.Errorf("len comment/key/export not restored: %+v", rl)
	}
	if tol := rl.Tolerance(); tol.Upper != 0.01 || tol.Lower != -0.02 || tol.Type != param.Median {
		t.Errorf("tolerance = %+v, want {0.01 -0.02 Median}", tol)
	}
	if !rl.IsMultiValue() || !rl.AllowsCustomValue() || len(rl.ExpressionList()) != 2 {
		t.Errorf("multi-value not restored: %v custom=%v", rl.ExpressionList(), rl.AllowsCustomValue())
	}
	if g, _ := r.GroupOf(rl.ID()); g != "Frame" {
		t.Errorf("len group = %q, want Frame", g)
	}

	rt, _ := r.ByName("material")
	if !rt.IsText() || rt.Text() != "steel" {
		t.Errorf("text parameter not restored: isText=%v text=%q", rt.IsText(), rt.Text())
	}
	rf, _ := r.ByName("vented")
	if !rf.IsBoolean() || !rf.Bool() {
		t.Errorf("boolean parameter not restored: isBool=%v val=%v", rf.IsBoolean(), rf.Bool())
	}
	if g, _ := r.GroupOf(rt.ID()); g != "Frame" {
		t.Errorf("material group = %q, want Frame", g)
	}
	_ = flag
}

func TestSketchGeometrySurvivesRoundTrip(t *testing.T) {
	ws := doc.NewWorkspace(nil)
	d, _ := compdef.AddPart(ws, "Part1", true)
	def := d.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	// A closed rectangle from four corner-sharing lines.
	c0 := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	c1 := sk.Lines().AddByTwoPoints(math.P2(4, 0), math.P2(4, 3))
	sk.GeometricConstraints().AddPerpendicular(c0, c1)

	reopened := reopenThroughStore(t, d)
	if got := reopened.Sketches().Count(); got != 1 {
		t.Fatalf("sketch count after reopen = %d, want 1", got)
	}
	rs := reopened.Sketches().Item(0)
	if got := rs.Lines().Count(); got != 2 {
		t.Errorf("line count after reopen = %d, want 2", got)
	}
	if got := rs.GeometricConstraints().Count(); got != 1 {
		t.Errorf("constraint count after reopen = %d, want 1", got)
	}
}

func TestExtrudedSolidSurvivesRoundTrip(t *testing.T) {
	ws := doc.NewWorkspace(nil)
	d, _ := compdef.AddPart(ws, "Part1", true)
	def := d.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	rectangle(sk, 4, 3)
	feature.NewExtrudeFeatures(def.Features()).
		AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()
	before := def.RangeBox()
	if len(def.SurfaceBodies().All()) != 1 {
		t.Fatalf("expected 1 body before save, got %d", len(def.SurfaceBodies().All()))
	}

	reopened := reopenThroughStore(t, d)
	bodies := reopened.SurfaceBodies().All()
	if len(bodies) != 1 {
		t.Fatalf("expected 1 body after reopen+recompute, got %d (the extrude must regenerate)", len(bodies))
	}
	after := reopened.RangeBox()
	if !after.Min.IsEqualTo(before.Min, 1e-6) || !after.Max.IsEqualTo(before.Max, 1e-6) {
		t.Errorf("range box after reopen = [%v..%v], want [%v..%v]",
			after.Min, after.Max, before.Min, before.Max)
	}
}

// rectangle adds a closed w×h rectangle (one profile) at the sketch origin, with the
// corners as shared points so the loop is structurally closed (a shared point IS a
// coincidence) — the same way the head's rectangle tool builds it.
func rectangle(s *sketch.Sketch, w, h float64) {
	c0 := s.Points().Add(math.P2(0, 0))
	c1 := s.Points().Add(math.P2(w, 0))
	c2 := s.Points().Add(math.P2(w, h))
	c3 := s.Points().Add(math.P2(0, h))
	s.Lines().Add(c0, c1)
	s.Lines().Add(c1, c2)
	s.Lines().Add(c2, c3)
	s.Lines().Add(c3, c0)
}

func TestFilletEdgeKeyRebindsAfterReopen(t *testing.T) {
	ws := doc.NewWorkspace(nil)
	d, _ := compdef.AddPart(ws, "Part1", true)
	def := d.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	rectangle(sk, 4, 3)
	feature.NewExtrudeFeatures(def.Features()).
		AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()

	// Pick a real edge off the extruded body and fillet it by reference key.
	edgeKey := def.SurfaceBodies().All()[0].Edges()[0].ReferenceKey()
	fillet := feature.NewDressUpFeatures(def.Features()).
		AddFillet([][]byte{edgeKey}, func() float64 { return 0.2 })
	def.Recompute()
	if !fillet.Health().OK() {
		t.Fatalf("fillet health before save = %v, want OK (edge resolved, real blend built)", fillet.Health().Status)
	}

	reopened := reopenThroughStore(t, d)
	// The reopened extrude regenerates the body with the same lineage, so the fillet's
	// edge key must re-bind and the blend rebuild — proving reference keys survive
	// save→reopen (a lost key would make the fillet Sick).
	got, ok := featureByKind(reopened.Features(), "fillet")
	if !ok {
		t.Fatal("fillet feature missing after reopen")
	}
	if !got.Health().OK() {
		t.Errorf("fillet health after reopen = %v, want OK (edge key must re-bind, not be lost/Sick)", got.Health().Status)
	}
}

// featureByKind returns the first feature of the given kind.
func featureByKind(fs *feature.PartFeatures, kind string) (*feature.PartFeature, bool) {
	for i := 0; i < fs.Count(); i++ {
		if fs.Item(i).Kind() == kind {
			return fs.Item(i), true
		}
	}
	return nil, false
}

func TestWorkFeaturesSurviveRoundTrip(t *testing.T) {
	ws := doc.NewWorkspace(nil)
	d, _ := compdef.AddPart(ws, "Part1", true)
	def := d.Content().(*compdef.PartComponentDefinition)
	// A user work plane offset 5 above the origin XY plane (references the origin).
	def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 5 })

	reopened := reopenThroughStore(t, d)
	if got := reopened.WorkPlanes().Count(); got != 4 {
		t.Fatalf("work plane count after reopen = %d, want 4 (3 origin + 1 user)", got)
	}
	// The origin frame is regenerated (grounded coordinate-system elements).
	if !reopened.WorkPlanes().Item(0).IsCoordinateSystemElement() {
		t.Error("origin plane lost its coordinate-system flag after reopen")
	}
	user := reopened.WorkPlanes().Item(3)
	if user.IsCoordinateSystemElement() {
		t.Error("restored user plane should not be a coordinate-system element")
	}
	if !user.Plane().Origin().IsEqualTo(math.P3(0, 0, 5), 1e-9) {
		t.Errorf("restored work plane origin = %v, want (0,0,5) — its ref to the origin XY plane must re-resolve", user.Plane().Origin())
	}
}

func TestRevolveAxisRebindsAfterReopen(t *testing.T) {
	ws := doc.NewWorkspace(nil)
	d, _ := compdef.AddPart(ws, "Part1", true)
	def := d.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	rectangle(sk, 4, 3)
	zAxis := def.WorkAxes().Item(2) // the origin Z axis
	feature.NewRevolveFeatures(def.Features()).
		Add(sk, 0, zAxis, func() float64 { return 0 }, ops.NewBody)

	reopened := reopenThroughStore(t, d)
	rev, ok := featureByKind(reopened.Features(), "revolve")
	if !ok {
		t.Fatal("revolve feature missing after reopen")
	}
	axis := rev.Definition().(*feature.RevolveFeature).Definition().Axis
	if axis == nil || axis.Key() != feature.OriginZAxis {
		t.Errorf("revolve axis after reopen = %v, want origin Z axis — it must re-bind by WorkRef", axis)
	}
}

func TestLengthUnitSurvivesRoundTrip(t *testing.T) {
	ws := doc.NewWorkspace(nil)
	d, _ := compdef.AddPart(ws, "Part1", true)
	def := d.Content().(*compdef.PartComponentDefinition)
	if err := def.SetLengthUnit("in"); err != nil {
		t.Fatalf("SetLengthUnit: %v", err)
	}

	reopened := reopenThroughStore(t, d)
	if got := reopened.Units().PreferredName(param.Length); got != "in" {
		t.Errorf("length unit after reopen = %q, want in", got)
	}
}
