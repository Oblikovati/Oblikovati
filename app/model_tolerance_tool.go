// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/model/feature"
)

// ModelToleranceTool annotates model geometry with GD&T: a feature-control frame (a geometric
// characteristic over a tolerance zone, optionally relative to datums) or a datum-feature label.
//
// ModelToleranceDefinition and ToleranceFeatures.AddModelTolerance were implemented and routed
// over the API, but ToleranceFeatures was 0-of-1 UI-reachable: the Drawing tab's feature-control
// frame and datum annotate a drawing VIEW, not the model, which is what MBD consumers and
// downstream exchange read (#2049).
type ModelToleranceTool struct {
	datumMode      bool // false ⇒ feature-control frame, true ⇒ datum feature
	geometry       Selectable
	characteristic types.GeometricCharacteristic
	value          float64
	datums         string // frame mode: comma-separated datum labels, e.g. "A,B"
	label          string // datum mode: the datum letter
	added          *feature.PartFeature
}

// NewModelFrameTool returns the tool in feature-control-frame mode, defaulting to a
// 0.1 position tolerance — the most common frame.
func NewModelFrameTool() *ModelToleranceTool {
	return &ModelToleranceTool{characteristic: types.CharacteristicPosition, value: 0.1}
}

// NewModelDatumTool returns the tool in datum-feature mode, defaulting to datum A.
func NewModelDatumTool() *ModelToleranceTool {
	return &ModelToleranceTool{datumMode: true, label: "A"}
}

// Name implements [Tool].
func (t *ModelToleranceTool) Name() string {
	if t.datumMode {
		return "Datum Feature"
	}
	return "Feature Control Frame"
}

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *ModelToleranceTool) Start(*Session) {}

// AcceptedKinds declares the tool annotates a face or an edge.
func (t *ModelToleranceTool) AcceptedKinds() []SelectionKind {
	return []SelectionKind{SelectFace, SelectEdge}
}

// Picks reports the annotated geometry for the unified highlight.
func (t *ModelToleranceTool) Picks() []Selectable {
	if t.geometry == nil {
		return nil
	}
	return []Selectable{t.geometry}
}

// Pick records the geometry to annotate, replacing any previous pick — one annotation names one
// reference, so a second click means "that one instead".
func (t *ModelToleranceTool) Pick(_ *Session, sel Selectable) {
	switch sel.(type) {
	case FaceHandle, EdgeHandle:
		t.geometry = sel
	}
}

// DatumMode reports which annotation the tool builds.
func (t *ModelToleranceTool) DatumMode() bool { return t.datumMode }

// GeometryPicked reports whether the annotated reference is chosen.
func (t *ModelToleranceTool) GeometryPicked() bool { return t.geometry != nil }

// ClearGeometry drops the picked reference — the property panel's selector clear (⊗).
func (t *ModelToleranceTool) ClearGeometry() { t.geometry = nil }

// CharacteristicIndex / SetCharacteristicIndex expose the geometric characteristic as an index
// into [GeometricCharacteristicOptions].
func (t *ModelToleranceTool) CharacteristicIndex() int {
	for i, c := range geometricCharacteristics {
		if c == t.characteristic {
			return i
		}
	}
	return 0
}

// SetCharacteristicIndex selects the characteristic from a combo index; out of range is ignored.
func (t *ModelToleranceTool) SetCharacteristicIndex(i int) {
	if i >= 0 && i < len(geometricCharacteristics) {
		t.characteristic = geometricCharacteristics[i]
	}
}

// SetValue / Value hold the tolerance zone size (database units).
func (t *ModelToleranceTool) SetValue(v float64) { t.value = v }
func (t *ModelToleranceTool) Value() float64     { return t.value }

// SetDatums / Datums hold the frame's datum references as typed, e.g. "A,B".
func (t *ModelToleranceTool) SetDatums(d string) { t.datums = d }
func (t *ModelToleranceTool) Datums() string     { return t.datums }

// SetLabel / Label hold the datum-feature letter.
func (t *ModelToleranceTool) SetLabel(l string) { t.label = strings.TrimSpace(l) }
func (t *ModelToleranceTool) Label() string     { return t.label }

// CanCommit requires the annotated geometry plus the mode's own input: a positive tolerance for
// a frame, a non-empty letter for a datum.
func (t *ModelToleranceTool) CanCommit() bool {
	if t.geometry == nil {
		return false
	}
	if t.datumMode {
		return t.label != ""
	}
	return t.value > 0
}

// Commit records the annotation as a feature in history and recomputes. The feature changes no
// geometry, so the recompute is what re-runs the tree with the annotation carried.
func (t *ModelToleranceTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = t.addTolerance(part.Features())
	if t.added == nil {
		return errors.New("model tolerance: nothing to annotate")
	}
	part.Recompute()
	s.recordEdit(part, t.Name())
	return nil
}

// addTolerance builds the annotation feature into engine fs — shared by Commit and the preview.
func (t *ModelToleranceTool) addTolerance(fs *feature.PartFeatures) *feature.PartFeature {
	key, ok := annotatedGeometryKey(t.geometry)
	if !ok {
		return nil
	}
	def := &feature.ModelToleranceDefinition{}
	if t.datumMode {
		def.Datums = []feature.DatumLabel{{GeometryKey: key, Label: t.label}}
	} else {
		def.Frames = []feature.ToleranceFrame{{
			GeometryKey: key, Characteristic: t.characteristic, Value: t.value, Datums: splitModelDatums(t.datums),
		}}
	}
	return feature.NewToleranceFeatures(fs).AddModelTolerance(def)
}

// annotatedGeometryKey is the reference key of the picked face or edge.
func annotatedGeometryKey(sel Selectable) ([]byte, bool) {
	switch h := sel.(type) {
	case FaceHandle:
		return h.Face.ReferenceKey(), true
	case EdgeHandle:
		return h.Edge.ReferenceKey(), true
	default:
		return nil, false
	}
}

// splitModelDatums parses the datum field ("A, B") into labels, dropping blanks so a trailing comma
// does not record an empty datum.
func splitModelDatums(s string) []string {
	var out []string
	for part := range strings.SplitSeq(s, ",") {
		if d := strings.TrimSpace(part); d != "" {
			out = append(out, d)
		}
	}
	return out
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *ModelToleranceTool) AddedFeature() *feature.PartFeature { return t.added }

// DraftFeature satisfies DraftPreviewable so the commit gate has a draft to inspect (#1626).
// A tolerance changes no geometry, so the gate can only ever find it healthy — the draft exists
// to keep the gate unskippable by construction.
func (t *ModelToleranceTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addTolerance(fs), nil
	})
}

// Prompt guides the user through the annotation steps.
func (t *ModelToleranceTool) Prompt(*Session) string {
	if t.geometry == nil {
		return "Click the face or edge to annotate"
	}
	if t.datumMode {
		return "Enter the datum letter, then click OK"
	}
	return "Choose the characteristic and tolerance, then click OK"
}

// Cancel is a no-op; the engine restores the ambient filter.
func (t *ModelToleranceTool) Cancel(*Session) {}
