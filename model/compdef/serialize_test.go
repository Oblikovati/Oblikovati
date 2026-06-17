// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"path/filepath"
	"testing"

	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
	"oblikovati.org/persistence"
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

// TestRestoreRecipeReplacesRatherThanMerges is the regression test for the undo restore
// path: applying a snapshot to an already-populated definition must yield exactly the
// snapshot, not a union with what was there. (ApplyRecipe alone merges additively — it
// is for loading onto a fresh part — so undo would otherwise duplicate sketches and
// re-add parameters.)
func TestRestoreRecipeReplacesRatherThanMerges(t *testing.T) {
	ws := doc.NewWorkspace(nil)
	d, err := compdef.AddPart(ws, "Part1", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := d.Content().(*compdef.PartComponentDefinition)

	// Snapshot an empty part, then populate it.
	empty, err := def.MarshalRecipe()
	if err != nil {
		t.Fatalf("marshal empty: %v", err)
	}
	def.Sketches().Add(sketch.XYPlane())
	if _, err := def.Parameters().AddUserParameter("w", "5 mm"); err != nil {
		t.Fatalf("add param: %v", err)
	}
	populated, err := def.MarshalRecipe()
	if err != nil {
		t.Fatalf("marshal populated: %v", err)
	}

	// Restoring the empty snapshot onto the populated part must clear the additions.
	if err := def.RestoreRecipe(empty); err != nil {
		t.Fatalf("RestoreRecipe(empty): %v", err)
	}
	if def.Sketches().Count() != 0 {
		t.Errorf("after restore-empty: %d sketches, want 0 (merge, not replace)", def.Sketches().Count())
	}
	if _, ok := def.Parameters().ByName("w"); ok {
		t.Error("after restore-empty: parameter w still present (merge, not replace)")
	}

	// Restoring the populated snapshot brings them back exactly once.
	if err := def.RestoreRecipe(populated); err != nil {
		t.Fatalf("RestoreRecipe(populated): %v", err)
	}
	if def.Sketches().Count() != 1 {
		t.Errorf("after restore-populated: %d sketches, want 1", def.Sketches().Count())
	}
	if _, ok := def.Parameters().ByName("w"); !ok {
		t.Error("after restore-populated: parameter w missing")
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
	_ = num.SetToleranceDeviation(0.01, -0.02)
	_ = num.SetModelValueType(param.Median)
	num.DisplayFormat = param.DisplayFormatFractional
	num.Visible = false
	num.CustomProperty.PropertyType, num.CustomProperty.ShowTrailingZeros = types.CustomPropertyNumber, true
	_ = num.SetExpressionList([]string{"10 mm", "20 mm"}, true)
	txt, _ := ps.AddTextUserParameter("material", "steel")
	flag, _ := ps.AddBooleanUserParameter("vented", true)
	_, _ = ps.AddGroup("com.example:frame", "Frame", "com.example")
	_ = ps.AddToGroup(num.ID(), "com.example:frame")
	_ = ps.AddToGroup(txt.ID(), "com.example:frame")
	ps.Settings().LinearStandardTolerance = "0.1 mm"
	ps.Settings().UseStandardTolerances = true
	ps.Settings().DimensionDisplayType = types.DimensionDisplayExpression

	r := reopenThroughStore(t, d).Parameters()

	rl, _ := r.ByName("len")
	if rl.Comment != "the length" || !rl.IsKey || !rl.ExposedAsProperty {
		t.Errorf("len comment/key/export not restored: %+v", rl)
	}
	if tol := rl.Tolerance(); tol.Upper != 0.01 || tol.Lower != -0.02 || tol.Type != types.ToleranceDeviation {
		t.Errorf("tolerance = %+v, want deviation {0.01 -0.02}", tol)
	}
	if rl.ModelValueType() != param.Median {
		t.Errorf("model value type = %s, want median", rl.ModelValueType())
	}
	if rl.DisplayFormat != param.DisplayFormatFractional || rl.Visible {
		t.Errorf("display format/visibility not restored: %s visible=%v", rl.DisplayFormat, rl.Visible)
	}
	if cp := rl.CustomProperty; cp.PropertyType != types.CustomPropertyNumber || !cp.ShowTrailingZeros || !cp.ShowLeadingZeros {
		t.Errorf("custom property format not restored: %+v", cp)
	}
	if !rl.CustomOrder() {
		t.Error("multi-value list must restore as authored-order")
	}
	if !rl.IsMultiValue() || !rl.AllowsCustomValue() || len(rl.ExpressionList()) != 2 {
		t.Errorf("multi-value not restored: %v custom=%v", rl.ExpressionList(), rl.AllowsCustomValue())
	}
	rg, ok := r.GroupByKey("com.example:frame")
	if !ok || rg.DisplayName != "Frame" || rg.ClientID != "com.example" {
		t.Errorf("group record = %+v, want key/display/client restored", rg)
	}
	if got := r.GroupsOf(rl.ID()); len(got) != 1 || got[0] != "com.example:frame" {
		t.Errorf("len groups = %v, want [com.example:frame]", got)
	}

	rt, _ := r.ByName("material")
	if !rt.IsText() || rt.Text() != "steel" {
		t.Errorf("text parameter not restored: isText=%v text=%q", rt.IsText(), rt.Text())
	}
	rf, _ := r.ByName("vented")
	if !rf.IsBoolean() || !rf.Bool() {
		t.Errorf("boolean parameter not restored: isBool=%v val=%v", rf.IsBoolean(), rf.Bool())
	}
	if got := r.GroupsOf(rt.ID()); len(got) != 1 || got[0] != "com.example:frame" {
		t.Errorf("material groups = %v, want [com.example:frame]", got)
	}
	rs := r.Settings()
	if rs.LinearStandardTolerance != "0.1 mm" || !rs.UseStandardTolerances || rs.DimensionDisplayType != types.DimensionDisplayExpression {
		t.Errorf("parameter settings = %+v, want the saved standard tolerance/display restored", rs)
	}
	_ = flag
}

// TestDerivedTableSurvivesRoundTrip checks a derived parameter table restores
// with its id, source link, and produced parameters reconnected by name.
func TestDerivedTableSurvivesRoundTrip(t *testing.T) {
	ws := doc.NewWorkspace(nil)
	d, _ := compdef.AddPart(ws, "Part1", true)
	ps := d.Content().(*compdef.PartComponentDefinition).Parameters()
	src := []param.SourceParameterValue{{Name: "module", Value: param.Quantity{Value: 0.2, Unit: param.Length}}}
	table, err := ps.AddDerivedTable("gears.obk", []string{"module"}, src)
	if err != nil {
		t.Fatalf("AddDerivedTable: %v", err)
	}

	r := reopenThroughStore(t, d).Parameters()
	rt, ok := r.DerivedTableByID(table.ID())
	if !ok || rt.SourceDocument() != "gears.obk" || len(rt.Linked()) != 1 {
		t.Fatalf("restored table = %+v, want gears.obk linking module under id %d", rt, table.ID())
	}
	p, ok := r.ByName("module")
	if !ok || p.Kind() != param.DerivedParam {
		t.Fatalf("restored derived parameter = %+v, want read-only module", p)
	}
	// The reconnected link must sync — proof the produced map rebuilt by name.
	changed := []param.SourceParameterValue{{Name: "module", Value: param.Quantity{Value: 0.4, Unit: param.Length}}}
	if err := r.SyncDerivedTable(rt.ID(), changed, true); err != nil {
		t.Fatalf("sync after reload: %v", err)
	}
	if p.Value().Value != 0.4 {
		t.Errorf("synced reloaded value = %v, want 0.4", p.Value().Value)
	}
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

// TestCreationOrderAndSharedSurviveRoundTrip is the issue-#132 persistence regression: the
// global creation stamps that interleave sketches/work planes/features, and a sketch's
// Shared flag, must round-trip so a reopened browser shows the same chronological tree.
func TestCreationOrderAndSharedSurviveRoundTrip(t *testing.T) {
	ws := doc.NewWorkspace(nil)
	d, _ := compdef.AddPart(ws, "Part1", true)
	def := d.Content().(*compdef.PartComponentDefinition)

	// Build in a deliberate order: sketch1 → extrude → work plane → sketch2 (shared).
	sk1 := def.Sketches().Add(sketch.XYPlane())
	rectangle(sk1, 4, 3)
	ext := feature.NewExtrudeFeatures(def.Features()).
		AddByDistanceExtent(sk1, 0, ops.NewBody, func() float64 { return 5 })
	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 2 })
	sk2 := def.Sketches().Add(sketch.XYPlane())
	sk2.SetShared(true)
	def.Recompute()

	// The stamps must reflect creation order before saving.
	if !strictlyAscending(sk1.Seq(), ext.Seq(), wp.Seq(), sk2.Seq()) {
		t.Fatalf("pre-save creation order wrong: sk1=%d ext=%d wp=%d sk2=%d",
			sk1.Seq(), ext.Seq(), wp.Seq(), sk2.Seq())
	}

	reopened := reopenThroughStore(t, d)
	rs1, rs2 := reopened.Sketches().Item(0), reopened.Sketches().Item(1)
	rext := reopened.Features().Item(0)
	rwp := reopened.WorkPlanes().Item(reopened.WorkPlanes().Count() - 1)

	if !strictlyAscending(rs1.Seq(), rext.Seq(), rwp.Seq(), rs2.Seq()) {
		t.Errorf("reopened creation order wrong: sk1=%d ext=%d wp=%d sk2=%d",
			rs1.Seq(), rext.Seq(), rwp.Seq(), rs2.Seq())
	}
	if rs1.Shared() {
		t.Error("sketch1 must not be shared after reopen")
	}
	if !rs2.Shared() {
		t.Error("sketch2's Shared flag was lost on reopen")
	}
}

// strictlyAscending reports whether the creation stamps increase at every step (the
// chronological order the browser interleaves by).
func strictlyAscending(seqs ...uint64) bool {
	for i := 1; i < len(seqs); i++ {
		if seqs[i] <= seqs[i-1] {
			return false
		}
	}
	return true
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

// TestEditedWorkPlaneScalarSurvivesRoundTrip: an offset edited through the redefine seam
// (browser dialog / workPlanes.redefine) severs the creation-time closure with an explicit
// value; the serializer must persist the EFFECTIVE distance, and the reopened plane must sit
// at the edited offset.
func TestEditedWorkPlaneScalarSurvivesRoundTrip(t *testing.T) {
	ws := doc.NewWorkspace(nil)
	d, _ := compdef.AddPart(ws, "Part1", true)
	def := d.Content().(*compdef.PartComponentDefinition)
	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 2 })
	def.Recompute()

	wp.EditableScalars()[0].Set(7) // the browser edit overrides the closure's 2
	def.Recompute()

	reopened := reopenThroughStore(t, d)
	user := reopened.WorkPlanes().Item(3)
	if !user.Plane().Origin().IsEqualTo(math.P3(0, 0, 7), 1e-9) {
		t.Errorf("reopened edited plane origin = %v, want (0,0,7) — the edited offset was lost", user.Plane().Origin())
	}
	if got := user.EditableScalars()[0].Get(); got != 7 {
		t.Errorf("reopened edited offset reads %v, want 7", got)
	}
}

// TestNodesCreatedAfterReopenSortLast: the whole point of seq.Restore's clock bump — a node
// added to a reopened document must sort after every restored node, or the browser timeline
// would interleave new work into the middle of the restored history.
func TestNodesCreatedAfterReopenSortLast(t *testing.T) {
	ws := doc.NewWorkspace(nil)
	d, _ := compdef.AddPart(ws, "Part1", true)
	def := d.Content().(*compdef.PartComponentDefinition)
	sk1 := def.Sketches().Add(sketch.XYPlane())
	rectangle(sk1, 4, 3)
	feature.NewExtrudeFeatures(def.Features()).
		AddByDistanceExtent(sk1, 0, ops.NewBody, func() float64 { return 5 })
	def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 2 })
	def.Recompute()

	reopened := reopenThroughStore(t, d)
	maxRestored := reopened.Sketches().Item(0).Seq()
	if s := reopened.Features().Item(0).Seq(); s > maxRestored {
		maxRestored = s
	}
	if s := reopened.WorkPlanes().Item(3).Seq(); s > maxRestored {
		maxRestored = s
	}

	fresh := reopened.Sketches().Add(sketch.XYPlane())
	if fresh.Seq() <= maxRestored {
		t.Errorf("sketch created after reopen has stamp %d ≤ restored max %d — it would sort into the restored history", fresh.Seq(), maxRestored)
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

// TestUnitsPrecisionAndFormatSurviveRoundTrip persists the display precision and
// the length/angle formats (the #146 additions) and restores them.
func TestUnitsPrecisionAndFormatSurviveRoundTrip(t *testing.T) {
	ws := doc.NewWorkspace(nil)
	d, _ := compdef.AddPart(ws, "Part1", true)
	def := d.Content().(*compdef.PartComponentDefinition)

	u := def.Units().Clone()
	if err := u.SetPreferred(param.Length, "in"); err != nil {
		t.Fatal(err)
	}
	if err := u.SetLengthPrecision(5); err != nil {
		t.Fatal(err)
	}
	if err := u.SetAnglePrecision(4); err != nil {
		t.Fatal(err)
	}
	u.SetLengthFormat(types.DisplayFormatFractional)
	u.SetAngleFormat(param.AngleDMS)
	def.SetUnits(u)

	got := reopenThroughStore(t, d).Units()
	if got.LengthPrecision() != 5 || got.AnglePrecision() != 4 {
		t.Errorf("precision after reopen = %d/%d, want 5/4", got.LengthPrecision(), got.AnglePrecision())
	}
	if got.LengthFormat() != types.DisplayFormatFractional || got.AngleFormat() != param.AngleDMS {
		t.Errorf("formats after reopen = %v/%v, want fractional/DMS", got.LengthFormat(), got.AngleFormat())
	}
}

// TestRevolveTwoDirectionalSurvivesReopen: the second-direction sweep (#313)
// persists and the restored revolve rebuilds the same straddling solid.
func TestRevolveTwoDirectionalSurvivesReopen(t *testing.T) {
	ws := doc.NewWorkspace(nil)
	d, _ := compdef.AddPart(ws, "Part1", true)
	def := d.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	offsetRect(sk, 2, 2)
	yAxis := def.WorkAxes().Item(1) // the origin Y axis
	feature.NewRevolveFeatures(def.Features()).AddTwoDirectional(sk, 0, yAxis,
		func() float64 { return stdmath.Pi / 2 }, func() float64 { return stdmath.Pi / 2 }, ops.NewBody)
	def.Recompute()

	reopened := reopenThroughStore(t, d)
	rev, ok := featureByKind(reopened.Features(), "revolve")
	if !ok {
		t.Fatal("revolve feature missing after reopen")
	}
	rdef := rev.Definition().(*feature.RevolveFeature).Definition()
	if rdef.Angle2 == nil || stdmath.Abs(rdef.Angle2()-stdmath.Pi/2) > 1e-12 {
		t.Fatal("Angle2 must survive the reopen")
	}
	reopened.Recompute()
	want := stdmath.Pi * (4*4 - 2*2) * 2 / 2 // the 12π half washer
	got := ops.BodyGeometryProperties(reopened.SurfaceBodies().All()[0], ops.DefaultQuality()).Volume
	if stdmath.Abs(got-want)/want > 0.01 {
		t.Errorf("reopened two-directional revolve volume = %g, want ≈%g", got, want)
	}
}

// offsetRect draws a rectangle x∈[x0, x0+side], y∈[0, side] (a profile offset
// from the revolve axis).
func offsetRect(sk *sketch.Sketch, x0, side float64) {
	p := func(x, y float64) *sketch.Point { return sk.Points().Add(math.P2(math.Scalar(x), math.Scalar(y))) }
	c0, c1, c2, c3 := p(x0, 0), p(x0+side, 0), p(x0+side, side), p(x0, side)
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
}

// TestSweepUnionSurvivesReopen: the M08 sweep union fields (#314 — taper,
// twist stations, rail, scaling, orientation) persist and the restored sweep
// rebuilds the same solid.
func TestSweepUnionSurvivesReopen(t *testing.T) {
	ws := doc.NewWorkspace(nil)
	d, _ := compdef.AddPart(ws, "Part1", true)
	def := d.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	centeredRect(sk, 1)
	rail := sketch.NewPath3D([]*sketch.Point3D{
		sketch.NewPoint3D(math.P3(2, 0, 0)), sketch.NewPoint3D(math.P3(4, 0, 6)),
	}, false)
	zPath := sketch.NewPath3D([]*sketch.Point3D{
		sketch.NewPoint3D(math.P3(0, 0, 0)), sketch.NewPoint3D(math.P3(0, 0, 6)),
	}, false)
	sdef := &feature.SweepDefinition{
		Sketch: sk, ProfileIndex: 0,
		Path:      func() *sketch.Path3D { return zPath },
		GuideRail: func() *sketch.Path3D { return rail },
		Scaling:   types.XYProfileScaling,
		TwistStations: []feature.SweepTwistStation{
			{T: 0, Angle: 0}, {T: 1, Angle: 0.3},
		},
		Operation: ops.NewBody,
	}
	feature.NewSweepFeatures(def.Features()).AddDefinition(sdef)
	def.Recompute()
	before := ops.BodyGeometryProperties(def.SurfaceBodies().All()[0], ops.DefaultQuality()).Volume

	reopened := reopenThroughStore(t, d)
	sw, ok := featureByKind(reopened.Features(), "sweep")
	if !ok {
		t.Fatal("sweep feature missing after reopen")
	}
	rdef := sw.Definition().(*feature.SweepFeature).Definition()
	if rdef.DefinitionType() != types.PathAndGuideRailSweepDef {
		t.Errorf("restored definition type = %v, want pathAndGuideRail", rdef.DefinitionType())
	}
	if len(rdef.TwistStations) != 2 || rdef.Scaling != types.XYProfileScaling {
		t.Errorf("restored union fields = %+v stations / %v scaling", rdef.TwistStations, rdef.Scaling)
	}
	reopened.Recompute()
	after := ops.BodyGeometryProperties(reopened.SurfaceBodies().All()[0], ops.DefaultQuality()).Volume
	if stdmath.Abs(after-before)/before > 0.01 {
		t.Errorf("reopened sweep volume = %g, want %g", after, before)
	}
}

// centeredRect draws a square of the given half-side centered on the origin.
func centeredRect(sk *sketch.Sketch, half float64) {
	p := func(x, y float64) *sketch.Point { return sk.Points().Add(math.P2(math.Scalar(x), math.Scalar(y))) }
	c0, c1, c2, c3 := p(-half, -half), p(half, -half), p(half, half), p(-half, half)
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
}

// TestFilletEdgeSetsSurviveReopen: a mixed constant + variable edge-set fillet
// (#323) reopens with its sets intact — same health, same volume.
func TestFilletEdgeSetsSurviveReopen(t *testing.T) {
	ws := doc.NewWorkspace(nil)
	d, _ := compdef.AddPart(ws, "Part1", true)
	def := d.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	rectangle(sk, 4, 3)
	feature.NewExtrudeFeatures(def.Features()).
		AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()

	var vertical [][]byte
	for _, e := range def.SurfaceBodies().All()[0].Edges() {
		if a, b := e.StartVertex().Point(), e.EndVertex().Point(); a.X == b.X && a.Y == b.Y {
			vertical = append(vertical, e.ReferenceKey())
		}
	}
	if len(vertical) < 2 {
		t.Fatalf("found %d vertical edges, want ≥2", len(vertical))
	}
	pf := feature.NewDressUpFeatures(def.Features()).AddFilletSets([]feature.FilletEdgeSet{
		{EdgeKeys: [][]byte{vertical[0]}, Radius: func() float64 { return 0.4 }},
		{EdgeKeys: [][]byte{vertical[1]}, StartRadius: func() float64 { return 0.2 }, EndRadius: func() float64 { return 0.7 }},
	})
	def.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("edge-set fillet before save = %v, want OK", pf.Health())
	}
	before := ops.BodyGeometryProperties(def.SurfaceBodies().All()[0], ops.DefaultQuality()).Volume

	reopened := reopenThroughStore(t, d)
	got, ok := featureByKind(reopened.Features(), "fillet")
	if !ok {
		t.Fatal("fillet feature missing after reopen")
	}
	if !got.Health().OK() {
		t.Fatalf("edge-set fillet after reopen = %v, want OK", got.Health())
	}
	rdef := got.Definition().(*feature.FilletFeature).Definition()
	if len(rdef.EdgeSets) != 2 || rdef.EdgeSets[1].Radius != nil || rdef.EdgeSets[1].EndRadius() != 0.7 {
		t.Errorf("restored edge sets = %d sets, want the constant+variable pair back", len(rdef.EdgeSets))
	}
	after := ops.BodyGeometryProperties(reopened.SurfaceBodies().All()[0], ops.DefaultQuality()).Volume
	if stdmath.Abs(after-before) > 1e-9 {
		t.Errorf("reopened fillet volume = %g, want %g", after, before)
	}
}
