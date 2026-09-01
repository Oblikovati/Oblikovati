// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// #1521: editability used to be opt-in through two unrelated bespoke mechanisms (a hand-maintained
// editToolFor type-switch and per-feature generic Editable/ReferenceEditable methods), with nothing
// forcing a new feature into either — so loft and sweep silently shipped with NO edit path: their
// browser Edit entry was greyed and double-clicking them did nothing. These tests pin the contract
// that every primary solid feature is editable, and that loft (full panel) and sweep (generic editor)
// actually round-trip.

// primarySolidFeatures are the sketch-based features that add or remove solid material — the ribbon's
// core part-modeling commands (the ToolFeature set the user places and expects to double-click-edit).
// EVERY one MUST be editable in place, either through a registered full-panel editor (featureEditors)
// or the generic scalar/reference editor (feature.Editable / feature.ReferenceEditable). The prototypes
// are typed nils: interface assertions are static, so they classify the type without calling a method.
var primarySolidFeatures = []struct {
	kind  string
	proto feature.Feature
}{
	{"extrude", (*feature.ExtrudeFeature)(nil)},
	{"revolve", (*feature.RevolveFeature)(nil)},
	{"coil", (*feature.CoilFeature)(nil)},
	{"hole", (*feature.HoleFeature)(nil)},
	{"rib", (*feature.RibFeature)(nil)},
	{"emboss", (*feature.EmbossFeature)(nil)},
	{"sweep", (*feature.SweepFeature)(nil)},
	{"loft", (*feature.LoftFeature)(nil)},
}

// TestPrimarySolidFeaturesAreEditable is the regression guard for #1521: a primary solid feature with
// neither a registered full-panel editor nor a generic editable surface is not editable in the browser
// — the exact loft/sweep defect. A new solid feature added without an edit path fails here.
func TestPrimarySolidFeaturesAreEditable(t *testing.T) {
	t.Parallel()
	editors := defaultFeatureEditors()
	for _, f := range primarySolidFeatures {
		if _, hasEditor := editors[f.kind]; hasEditor || implementsEditableContract(f.proto) {
			continue
		}
		t.Errorf("primary solid feature %q is not editable: no registered full-panel editor and its "+
			"definition implements neither feature.Editable nor feature.ReferenceEditable — double-clicking "+
			"it in the browser would do nothing (#1521)", f.kind)
	}
}

// implementsEditableContract reports whether a feature type exposes the generic edit surface — it
// implements feature.Editable (scalar params) or feature.ReferenceEditable (re-pickable references).
func implementsEditableContract(f feature.Feature) bool {
	_, params := f.(feature.Editable)
	_, refs := f.(feature.ReferenceEditable)
	return params || refs
}

// TestRegisterFeatureEditorRejectsBadRegistration locks the anti-drift guard: an empty kind, a nil
// editor, or a duplicate (a second tool claiming one kind) panics at startup rather than silently
// overwriting — the same discipline the serialization codec registry enforces (#1416).
func TestRegisterFeatureEditorRejectsBadRegistration(t *testing.T) {
	t.Parallel()
	ok := func(_ *Session, _ *feature.PartFeature) (Tool, bool) { return nil, false }
	assertPanicsApp(t, "empty kind", func() { featureEditorSet{}.register("", ok) })
	assertPanicsApp(t, "nil editor", func() { featureEditorSet{}.register("test.nil-editor", nil) })
	assertPanicsApp(t, "duplicate", func() { defaultFeatureEditors().register("loft", ok) }) // loft is already registered
}

// assertPanicsApp fails unless fn panics.
func assertPanicsApp(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s: expected a panic, got none", name)
		}
	}()
	fn()
}

// loftedFeatureSession commits a frustum loft (4×4 → 2×2, height 5) and returns a handle to it — the
// starting point for the loft edit-flow tests.
func loftedFeatureSession(t *testing.T) (*Session, FeatureHandle) {
	t.Helper()
	s, bottom, top := newPartWithStackedSquares(t)
	s.SetPicker(&seqPicker{sels: []Selectable{bottom, top}})
	l := NewLoftTool()
	s.StartTool(l)
	s.Click(10, 10)
	s.Click(10, 200)
	if err := s.OK(); err != nil {
		t.Fatalf("seed loft: %v", err)
	}
	return s, FeatureHandle{Feature: l.AddedFeature()}
}

// TestBeginEditFeatureReopensLoftPanel locks loft's full-panel edit (#1521): double-clicking a
// committed loft re-opens the Loft tool in edit mode, seeded with its sections — not the generic editor.
func TestBeginEditFeatureReopensLoftPanel(t *testing.T) {
	t.Parallel()
	s, h := loftedFeatureSession(t)
	if !s.FeatureIsEditable(h.Feature) {
		t.Fatal("a committed loft must report as editable (#1521)")
	}
	s.BeginEditFeature(h)
	l := s.ActiveLoft()
	if l == nil {
		t.Fatal("BeginEditFeature should re-open the Loft tool for a loft")
	}
	if !l.IsEditing() || l.EditingName() != h.Feature.Name() {
		t.Errorf("edit binding = (%v, %q), want (true, %q)", l.IsEditing(), l.EditingName(), h.Feature.Name())
	}
	if l.SectionCount() != 2 {
		t.Errorf("seeded section count = %d, want 2", l.SectionCount())
	}
	if s.IsEditingFeature() {
		t.Error("the generic feature editor must not be open for a loft")
	}
}

// TestLoftEditCommitPreservesGeometry proves the loft edit round-trips losslessly: re-opening and
// OK-ing with no change rebuilds the same solid (sections restored, opaque guides preserved). A
// commitEdit that dropped sections or wiped guide providers would change the volume.
func TestLoftEditCommitPreservesGeometry(t *testing.T) {
	t.Parallel()
	s, h := loftedFeatureSession(t)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	before := ops.BodyGeometryProperties(def.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume
	s.BeginEditFeature(h)
	if err := s.OK(); err != nil {
		t.Fatalf("commit loft edit: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("part has %d bodies after a no-op loft edit, want 1", def.SurfaceBodies().Count())
	}
	after := ops.BodyGeometryProperties(def.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume
	if relErrApp(after, before) > 0.001 {
		t.Errorf("a no-op loft edit changed the volume: %g → %g", before, after)
	}
}

// TestLoftEditAppliesWaist proves a panel edit reaches the definition: pinching the mid-height area
// scale writes an area-graph waist back into the committed loft.
func TestLoftEditAppliesWaist(t *testing.T) {
	t.Parallel()
	s, h := loftedFeatureSession(t)
	s.BeginEditFeature(h)
	l := s.ActiveLoft()
	l.SetAreaMidScale(0.5)
	if err := s.OK(); err != nil {
		t.Fatalf("commit loft edit: %v", err)
	}
	lf := h.Feature.Definition().(*feature.LoftFeature).Definition()
	if got := midAreaScale(lf.AreaGraph); got != 0.5 {
		t.Errorf("edited mid-area scale = %g, want 0.5 (area graph %v)", got, lf.AreaGraph)
	}
}

// sweptFeatureSession commits a 2×2-along-5 sweep and returns a handle to it.
func sweptFeatureSession(t *testing.T) (*Session, FeatureHandle) {
	t.Helper()
	s, profile, path := newPartWithProfileAndPath(t)
	s.SetPicker(&seqPicker{sels: []Selectable{profile, path}})
	sw := NewSweepTool()
	s.StartTool(sw)
	s.Click(10, 10)
	s.Click(10, 200)
	if err := s.OK(); err != nil {
		t.Fatalf("seed sweep: %v", err)
	}
	return s, FeatureHandle{Feature: sw.AddedFeature()}
}

// TestSweepIsEditableViaGenericEditor locks sweep's edit path (#1521): a sweep is editable, and it
// opens the GENERIC parameter editor (not a full panel) exposing its twist — its opaque path provider
// has no full-panel re-pick, so the generic scalar/reference editor is the honest fit.
func TestSweepIsEditableViaGenericEditor(t *testing.T) {
	t.Parallel()
	s, h := sweptFeatureSession(t)
	if !s.FeatureIsEditable(h.Feature) {
		t.Fatal("a committed sweep must report as editable (#1521)")
	}
	s.BeginEditFeature(h)
	if !s.IsEditingFeature() {
		t.Fatal("a sweep should open the generic feature editor")
	}
	if s.ActiveSweep() != nil {
		t.Error("sweep has no full-panel editor; it must use the generic editor")
	}
	if !editParamPresent(s, "Twist") {
		t.Error("sweep edit should expose a Twist parameter")
	}
}

// TestSweepEditPreservesPath proves the opaque path survives a generic edit: editing the twist and
// committing keeps the swept solid (a commitEdit that touched the path provider would lose the body).
func TestSweepEditPreservesPath(t *testing.T) {
	t.Parallel()
	s, h := sweptFeatureSession(t)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	before := ops.BodyGeometryProperties(def.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume
	s.BeginEditFeature(h)
	setGenericEditParam(s, "Twist", 0) // a real edit through the generic path; geometry unchanged at twist 0
	if err := s.OK(); err != nil {
		t.Fatalf("commit sweep edit: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("sweep edit wiped the body (path lost?): %d bodies", def.SurfaceBodies().Count())
	}
	after := ops.BodyGeometryProperties(def.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume
	if relErrApp(after, before) > 0.001 {
		t.Errorf("a twist-0 sweep edit changed the volume: %g → %g", before, after)
	}
}

// editParamPresent reports whether the open generic editor exposes a scalar field with the given label.
func editParamPresent(s *Session, label string) bool {
	for i := 0; i < s.EditFeatureParamCount(); i++ {
		if s.EditFeatureParamLabel(i) == label {
			return true
		}
	}
	return false
}

// setGenericEditParam sets the named scalar field in the open generic editor (no-op if absent).
func setGenericEditParam(s *Session, label string, value float64) {
	for i := 0; i < s.EditFeatureParamCount(); i++ {
		if s.EditFeatureParamLabel(i) == label {
			s.SetEditFeatureParamValue(i, value)
			return
		}
	}
}

// TestSessionConsultsInjectedEditorSet pins the B6 seam (#1617): the edit flow
// consults the editor set the Session carries — a session given a minimal
// one-entry set edits through it and nothing falls back to a package global.
func TestSessionConsultsInjectedEditorSet(t *testing.T) {
	t.Parallel()
	s := NewSession()
	marker := &FilletTool{}
	s.featureEditors = featureEditorSet{}
	s.featureEditors.register("test.injected", func(*Session, *feature.PartFeature) (Tool, bool) {
		return marker, true
	})
	if !s.hasFeatureEditor("test.injected") {
		t.Error("the injected editor kind is not visible through the session")
	}
	// extrude has an editor in the DEFAULT set; a session with a minimal set must
	// not see it — if it does, a package-global fallback still exists.
	if s.hasFeatureEditor("extrude") {
		t.Error("a kind outside the injected set is visible — a global fallback exists")
	}
}
