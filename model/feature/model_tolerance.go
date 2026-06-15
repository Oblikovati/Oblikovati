// SPDX-License-Identifier: GPL-2.0-only

package feature

import "oblikovati.org/api/types"

// ModelTolerance (M20-F13 #866) is a GD&T carrier: a metadata feature that annotates model
// geometry with feature-control frames and datum labels. Unlike the other F13 features it
// changes no geometry — it passes the running body through and carries tolerance records that
// survive recompute and the .obk round trip (mirrors the cosmetic/annotation features).

// GeometricCharacteristic is the geometric-tolerance symbol of a feature-control frame; the
// canonical definition lives in api/types (ADR-0018).
type GeometricCharacteristic = types.GeometricCharacteristic

// ToleranceFrame is one feature-control frame: a geometric characteristic over a tolerance
// zone of size Value, applied to a model geometry reference, optionally relative to the named
// datums (e.g. ["A","B"]).
type ToleranceFrame struct {
	GeometryKey    []byte
	Characteristic GeometricCharacteristic
	Value          float64
	Datums         []string
}

// DatumLabel marks a model geometry reference as a datum feature (label "A", "B", …).
type DatumLabel struct {
	GeometryKey []byte
	Label       string
}

// ModelToleranceDefinition is the GD&T record set carried by the feature.
type ModelToleranceDefinition struct {
	Frames []ToleranceFrame
	Datums []DatumLabel
}

// ModelToleranceFeature annotates the model; it never alters the running body.
type ModelToleranceFeature struct{ def *ModelToleranceDefinition }

// Definition returns the GD&T record set.
func (m *ModelToleranceFeature) Definition() *ModelToleranceDefinition { return m.def }

// Kind implements [Feature].
func (m *ModelToleranceFeature) Kind() string { return "modelTolerance" }

// Recompute passes the running body state through unchanged — tolerances are metadata.
func (m *ModelToleranceFeature) Recompute(in Input) (Output, error) {
	return Output{Bodies: in.Bodies}, nil
}

// ToleranceFeatures adds GD&T model-tolerance carriers into the engine.
type ToleranceFeatures struct{ engine *PartFeatures }

// NewToleranceFeatures binds the collection to a feature engine.
func NewToleranceFeatures(engine *PartFeatures) *ToleranceFeatures { return &ToleranceFeatures{engine} }

// AddModelTolerance records a GD&T annotation set as a feature in history.
func (c *ToleranceFeatures) AddModelTolerance(def *ModelToleranceDefinition) *PartFeature {
	mt := &ModelToleranceFeature{def: def}
	pf := c.engine.Add(mt)
	pf.SetName(c.engine.UniqueName("Tolerance"))
	return pf
}
