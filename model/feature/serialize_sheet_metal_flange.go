// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// SheetMetalFlangeData is the serialized form of a SheetMetalFlangeFeature: the picked edge
// (base64 reference key), the flange height, and the optional bend angle/radius (0 ⇒ default
// 90° / the rule's bend radius). Thickness is not stored — it is read live from the rule.
type SheetMetalFlangeData struct {
	Edge   string  `yaml:"edge"`
	Height float64 `yaml:"height"`
	Angle  float64 `yaml:"angle,omitempty"`  // 0 ⇒ 90° default
	Radius float64 `yaml:"radius,omitempty"` // 0 ⇒ use the rule's bend radius
	Flip   bool    `yaml:"flip,omitempty"`
	// BendPosition / HeightDatum place the wall (#1957); both absent ⇒ the bend at the picked edge
	// with the height measured from its tangent, which is what an older document meant.
	BendPosition   string  `yaml:"bendPosition,omitempty"`
	PositionOffset float64 `yaml:"positionOffset,omitempty"`
	HeightDatum    string  `yaml:"heightDatum,omitempty"`
}

// serializeSheetMetalFlange projects a flange recipe to its persisted form, freezing the
// height/angle/radius closures to their current values.
func serializeSheetMetalFlange(def *SheetMetalFlangeDefinition) *SheetMetalFlangeData {
	return &SheetMetalFlangeData{
		Edge:   encodeKey(def.EdgeKey),
		Height: evalFloat(def.Height),
		Angle:  evalFloat(def.Angle),
		Radius: evalFloat(def.Radius),
		Flip:   def.Flip,

		BendPosition:   BendPositionName(def.Position),
		PositionOffset: evalFloat(def.PositionOffset),
		HeightDatum:    HeightDatumName(def.HeightDatum),
	}
}

// restoreSheetMetalFlange rebuilds a flange feature, erroring on a missing payload or an
// undecodable edge key. A zero angle/radius restores as nil (default 90° / rule radius).
func restoreSheetMetalFlange(fs *PartFeatures, d *SheetMetalFlangeData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sheet-metal flange feature is missing its payload")
	}
	key, err := decodeKey(d.Edge)
	if err != nil {
		return nil, fmt.Errorf("sheet-metal flange edge key: %w", err)
	}
	position, ok := ParseBendPosition(d.BendPosition)
	if !ok {
		return nil, fmt.Errorf("sheet-metal flange: unknown bend position %q", d.BendPosition)
	}
	datum, ok := ParseHeightDatum(d.HeightDatum)
	if !ok {
		return nil, fmt.Errorf("sheet-metal flange: unknown height datum %q", d.HeightDatum)
	}
	def := &SheetMetalFlangeDefinition{
		EdgeKey:     key,
		Height:      constFloat(d.Height),
		Flip:        d.Flip,
		Position:    position,
		HeightDatum: datum,
	}
	if d.PositionOffset != 0 {
		def.PositionOffset = constFloat(d.PositionOffset)
	}
	if d.Angle != 0 {
		def.Angle = constFloat(d.Angle)
	}
	if d.Radius != 0 {
		def.Radius = constFloat(d.Radius)
	}
	return NewSheetMetalFlangeFeatures(fs).Add(def), nil
}
