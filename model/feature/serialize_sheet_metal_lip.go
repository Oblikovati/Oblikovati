// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// SheetMetalLipData is the serialized form of a SheetMetalLipFeature: the edge reference key, the
// flange height + return length, the optional bend angle/radius (0 ⇒ default 90° / rule radius),
// and the flip. Thickness is read live from the rule.
type SheetMetalLipData struct {
	Edge         string  `yaml:"edge"`
	Height       float64 `yaml:"height"`
	ReturnLength float64 `yaml:"returnLength"`
	Angle        float64 `yaml:"angle,omitempty"`  // 0 ⇒ 90° default
	Radius       float64 `yaml:"radius,omitempty"` // 0 ⇒ rule bend radius
	Flip         bool    `yaml:"flip,omitempty"`
}

// serializeSheetMetalLip projects a lip recipe to its persisted form.
func serializeSheetMetalLip(def *SheetMetalLipDefinition) *SheetMetalLipData {
	return &SheetMetalLipData{
		Edge:         encodeKey(def.EdgeKey),
		Height:       evalFloat(def.Height),
		ReturnLength: evalFloat(def.ReturnLength),
		Angle:        evalFloat(def.Angle),
		Radius:       evalFloat(def.Radius),
		Flip:         def.Flip,
	}
}

// restoreSheetMetalLip rebuilds a lip feature, erroring on a missing payload. A zero angle/radius
// restores as nil (default 90° / rule radius).
func restoreSheetMetalLip(fs *PartFeatures, d *SheetMetalLipData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sheet-metal lip feature is missing its payload")
	}
	key, err := decodeKey(d.Edge)
	if err != nil {
		return nil, fmt.Errorf("sheet-metal lip edge: %w", err)
	}
	def := &SheetMetalLipDefinition{
		EdgeKey:      key,
		Height:       constFloat(d.Height),
		ReturnLength: constFloat(d.ReturnLength),
		Flip:         d.Flip,
	}
	if d.Angle != 0 {
		def.Angle = constFloat(d.Angle)
	}
	if d.Radius != 0 {
		def.Radius = constFloat(d.Radius)
	}
	return NewSheetMetalLipFeatures(fs).Add(def), nil
}
