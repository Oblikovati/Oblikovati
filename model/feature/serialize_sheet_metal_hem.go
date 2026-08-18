// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// SheetMetalHemData is the serialized form of a SheetMetalHemFeature: the picked edge (base64
// reference key), the type and the dimensions that type takes. Thickness is read live from the
// rule, so it is not stored.
//
// The type is persisted as a NAME (#1956). It used to be the enum's ordinal, which the four
// Inventor types could not extend without changing what an existing ordinal means — ordinal 1 was
// the open hem and is now the double. LegacyType carries that old field: it is only ever read, and
// only when Type is absent, so a document written before the names decodes as the single hem both
// old ordinals meant.
type SheetMetalHemData struct {
	Edge       string  `yaml:"edge"`
	Length     float64 `yaml:"length,omitempty"`
	Type       string  `yaml:"hemType,omitempty"`
	LegacyType int32   `yaml:"type,omitempty"`
	Gap        float64 `yaml:"gap,omitempty"`
	Radius     float64 `yaml:"radius,omitempty"`
	Angle      float64 `yaml:"angle,omitempty"`
	Flip       bool    `yaml:"flip,omitempty"`
}

// serializeSheetMetalHem projects a hem recipe to its persisted form.
func serializeSheetMetalHem(def *SheetMetalHemDefinition) *SheetMetalHemData {
	return &SheetMetalHemData{
		Edge:   encodeKey(def.EdgeKey),
		Length: evalFloat(def.Length),
		Type:   HemTypeName(def.Type),
		Gap:    evalFloat(def.Gap),
		Radius: evalFloat(def.Radius),
		Angle:  evalFloat(def.Angle),
		Flip:   def.Flip,
	}
}

// restoreSheetMetalHem rebuilds a hem feature, erroring on a missing payload, an undecodable edge
// key, or a hem type this build does not know (which must not quietly become a single hem).
func restoreSheetMetalHem(fs *PartFeatures, d *SheetMetalHemData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sheet-metal hem feature is missing its payload")
	}
	key, err := decodeKey(d.Edge)
	if err != nil {
		return nil, fmt.Errorf("sheet-metal hem edge key: %w", err)
	}
	hemType, ok := hemTypeFromData(d)
	if !ok {
		return nil, fmt.Errorf("sheet-metal hem: unknown type %q", d.Type)
	}
	def := &SheetMetalHemDefinition{EdgeKey: key, Type: hemType, Flip: d.Flip}
	setHemDims(def, d)
	return NewSheetMetalHemFeatures(fs).Add(def), nil
}

// hemTypeFromData reads the hem type, falling back to the ordinal an older document carries.
// Both old ordinals — closed (0) and open (1) — were single hems; what set them apart was the gap,
// which is still stored and still says it.
func hemTypeFromData(d *SheetMetalHemData) (HemType, bool) {
	if d.Type == "" && d.LegacyType != 0 {
		return SingleHem, true
	}
	return ParseHemType(d.Type)
}

// setHemDims attaches only the dimensions the recipe actually carries, so an absent one stays nil
// and keeps its type's default (a hem with no gap folds tight) rather than becoming a zero.
func setHemDims(def *SheetMetalHemDefinition, d *SheetMetalHemData) {
	for _, f := range []struct {
		value float64
		dst   *func() float64
	}{{d.Length, &def.Length}, {d.Gap, &def.Gap}, {d.Radius, &def.Radius}, {d.Angle, &def.Angle}} {
		if f.value != 0 {
			*f.dst = constFloat(f.value)
		}
	}
}
