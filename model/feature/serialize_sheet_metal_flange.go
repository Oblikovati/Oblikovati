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
	// Width is how much of the edge the wall covers (#1958); absent ⇒ the whole edge.
	Width *FlangeWidthData `yaml:"width,omitempty"`
}

// FlangeWidthData persists a flange's width extent and the distances it takes (#1958).
type FlangeWidthData struct {
	Type    string  `yaml:"type"`
	Width   float64 `yaml:"width,omitempty"`
	Offset  float64 `yaml:"offset,omitempty"`
	Offset2 float64 `yaml:"offset2,omitempty"`
}

// serializeFlangeWidth persists a width extent, or nothing at all for the full-edge default so an
// existing recipe is untouched.
func serializeFlangeWidth(w FlangeWidth) *FlangeWidthData {
	if w.Type == WidthFullEdge {
		return nil
	}
	return &FlangeWidthData{Type: WidthExtentName(w.Type), Width: evalFloat(w.Width),
		Offset: evalFloat(w.Offset), Offset2: evalFloat(w.Offset2)}
}

// restoreFlangeWidth rebuilds a width extent; an unknown type is an error rather than a silent
// full-edge wall.
func restoreFlangeWidth(d *FlangeWidthData) (FlangeWidth, error) {
	if d == nil {
		return FlangeWidth{}, nil
	}
	t, ok := ParseWidthExtent(d.Type)
	if !ok {
		return FlangeWidth{}, fmt.Errorf("sheet-metal flange: unknown width extent %q", d.Type)
	}
	w := FlangeWidth{Type: t}
	for _, f := range []struct {
		value float64
		dst   *func() float64
	}{{d.Width, &w.Width}, {d.Offset, &w.Offset}, {d.Offset2, &w.Offset2}} {
		if f.value != 0 {
			*f.dst = constFloat(f.value)
		}
	}
	return w, nil
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
		Width:          serializeFlangeWidth(def.Width),
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
	width, err := restoreFlangeWidth(d.Width)
	if err != nil {
		return nil, err
	}
	def := &SheetMetalFlangeDefinition{
		EdgeKey:     key,
		Width:       width,
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
