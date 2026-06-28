// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/model/feature"
)

// The feature-editor registry is the SINGLE place a feature opts into the rich, full-panel edit
// path (Inventor's double-click on a history feature re-opening its creation dialog). It replaces
// the hand-maintained editToolFor type-switch whose silent omissions left whole features — loft and
// sweep — with no in-place edit at all (#1521): a type-switch has no failure when a case is missing,
// so the gap was invisible until a user tried to double-click a loft. A registry makes the contract
// explicit and enumerable, so an enforcement test (feature_edit_registry_test) can assert every
// primary solid feature is editable — the same anti-drift discipline the serialization codec
// registry brought to save/reload (#1416). It mirrors that pattern deliberately: one registration
// per feature, paired with its kind, validated at startup.
//
// A kind ABSENT from this registry is not an error — it falls back to the generic parameter /
// reference editor (FeatureEditTool), which serves features whose whole edit surface is scalars and
// re-pickable references (rib, emboss, the patterns, mirror, move). A kind in NEITHER the registry
// nor the generic interfaces is not editable; the enforcement test forbids that for solid features.

// featureEditor re-opens a creation tool over a committed feature, seeded from its definition, so the
// same property panel serves creation and editing. It returns ok=false when the tool cannot show this
// particular feature (e.g. a multi-set fillet the single-radius panel can't represent), so the caller
// falls back to the generic editor.
type featureEditor func(s *Session, f *feature.PartFeature) (Tool, bool)

// featureEditors maps a feature kind to its full-panel editor. Populated by registerFeatureEditor in
// this package's init()s; read by editToolFor. Read-only after init, so no synchronization is needed.
var featureEditors = map[string]featureEditor{}

// registerFeatureEditor records one kind's full-panel editor, panicking on an empty kind, a nil
// editor, or a duplicate — a programming error caught at startup, not a silent overwrite.
func registerFeatureEditor(kind string, e featureEditor) {
	if kind == "" {
		panic("app: registerFeatureEditor with empty kind")
	}
	if e == nil {
		panic(fmt.Sprintf("app: nil feature editor for kind %q", kind))
	}
	if _, dup := featureEditors[kind]; dup {
		panic(fmt.Sprintf("app: duplicate feature editor for kind %q", kind))
	}
	featureEditors[kind] = e
}

// hasFeatureEditor reports whether a kind has a registered full-panel editor (so the browser can
// enable Edit without constructing the tool).
func hasFeatureEditor(kind string) bool {
	_, ok := featureEditors[kind]
	return ok
}

// editToolFor builds the full-panel creation tool re-opened over a committed feature, or false for
// kinds the generic parameter/reference editor serves (or which are not editable at all).
func editToolFor(s *Session, f *feature.PartFeature) (Tool, bool) {
	e, ok := featureEditors[f.Kind()]
	if !ok {
		return nil, false
	}
	return e(s, f)
}

// registerSolidFeatureEditors wires the sketch-based solid features whose edit surface needs the full
// creation panel — enum choices and ordered section/condition lists the generic scalar/reference editor
// cannot express (extrude, revolve, coil, hole, loft). Sweep, by contrast, edits cleanly through the
// generic editor (twist, taper, profile) and its path is an opaque live provider that cannot be reversed
// into a re-pickable handle, so it is intentionally NOT registered here — see SweepFeature.EditableParams.
// Each closure type-asserts the concrete feature and calls the family's seeder; a mismatched kind→type
// registration would fail that assertion in the enforcement test, so the map cannot silently bind the
// wrong tool.
func registerSolidFeatureEditors() {
	registerFeatureEditor("extrude", func(_ *Session, f *feature.PartFeature) (Tool, bool) {
		return editExtrudeTool(f, f.Definition().(*feature.ExtrudeFeature)), true
	})
	registerFeatureEditor("revolve", func(s *Session, f *feature.PartFeature) (Tool, bool) {
		return editRevolveTool(s, f, f.Definition().(*feature.RevolveFeature)), true
	})
	registerFeatureEditor("coil", func(s *Session, f *feature.PartFeature) (Tool, bool) {
		return editCoilTool(s, f, f.Definition().(*feature.CoilFeature)), true
	})
	registerFeatureEditor("hole", func(_ *Session, f *feature.PartFeature) (Tool, bool) {
		return editHoleTool(f, f.Definition().(*feature.HoleFeature)), true
	})
	registerFeatureEditor("loft", func(_ *Session, f *feature.PartFeature) (Tool, bool) {
		return editLoftTool(f, f.Definition().(*feature.LoftFeature)), true
	})
}

// registerDressUpFeatureEditors wires the dress-up / local-operation features (fillet, face fillet,
// full-round fillet, chamfer, shell, draft) to their creation panels. The fillet panel declines a
// multi-set fillet (returns false), routing it to the generic editor.
func registerDressUpFeatureEditors() {
	registerFeatureEditor("fillet", func(_ *Session, f *feature.PartFeature) (Tool, bool) {
		return editFilletToolOr(f, f.Definition().(*feature.FilletFeature))
	})
	registerFeatureEditor("face-fillet", func(_ *Session, f *feature.PartFeature) (Tool, bool) {
		return editFaceFilletTool(f, f.Definition().(*feature.FaceFilletFeature)), true
	})
	registerFeatureEditor("full-round-fillet", func(_ *Session, f *feature.PartFeature) (Tool, bool) {
		return editFullRoundFilletTool(f, f.Definition().(*feature.FullRoundFilletFeature)), true
	})
	registerFeatureEditor("chamfer", func(_ *Session, f *feature.PartFeature) (Tool, bool) {
		return editChamferTool(f, f.Definition().(*feature.ChamferFeature)), true
	})
	registerFeatureEditor("shell", func(_ *Session, f *feature.PartFeature) (Tool, bool) {
		return editShellTool(f, f.Definition().(*feature.ShellFeature)), true
	})
	registerFeatureEditor("draft", func(_ *Session, f *feature.PartFeature) (Tool, bool) {
		return editDraftTool(f, f.Definition().(*feature.FaceDraftFeature)), true
	})
}

func init() {
	registerSolidFeatureEditors()
	registerDressUpFeatureEditors()
}
