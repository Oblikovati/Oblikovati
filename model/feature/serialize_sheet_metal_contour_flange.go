// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// SheetMetalContourFlangeData is the serialized form of a SheetMetalContourFlangeFeature: the
// picked edge (base64 reference key), the profile sketch index, and the flip. Thickness is
// read live from the rule.
type SheetMetalContourFlangeData struct {
	Edge    string `yaml:"edge"`
	Profile int    `yaml:"profile"`
	Flip    bool   `yaml:"flip,omitempty"`
	// Width is how much of the edge the swept wall covers (#1958); absent ⇒ the whole edge.
	Width *FlangeWidthData `yaml:"width,omitempty"`
}

// serializeSheetMetalContourFlange projects a contour-flange recipe to its persisted form,
// erroring if its profile sketch is not in the part.
func serializeSheetMetalContourFlange(def *SheetMetalContourFlangeDefinition, sk SketchIndexer) (*SheetMetalContourFlangeData, error) {
	idx, ok := sk.IndexOf(def.Profile)
	if !ok {
		return nil, fmt.Errorf("sheet-metal contour flange references a profile sketch that is not in the part")
	}
	return &SheetMetalContourFlangeData{Edge: encodeKey(def.EdgeKey), Profile: idx, Flip: def.Flip,
		Width: serializeFlangeWidth(def.Width)}, nil
}

// restoreSheetMetalContourFlange rebuilds a contour-flange feature, erroring on a missing
// payload, an undecodable edge key, or an unknown profile sketch.
func restoreSheetMetalContourFlange(fs *PartFeatures, d *SheetMetalContourFlangeData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sheet-metal contour flange feature is missing its payload")
	}
	key, err := decodeKey(d.Edge)
	if err != nil {
		return nil, fmt.Errorf("sheet-metal contour flange edge key: %w", err)
	}
	profile, ok := sk.At(d.Profile)
	if !ok {
		return nil, fmt.Errorf("sheet-metal contour flange references sketch %d which is not in the part", d.Profile)
	}
	width, err := restoreFlangeWidth(d.Width)
	if err != nil {
		return nil, err
	}
	return NewSheetMetalContourFlangeFeatures(fs).Add(&SheetMetalContourFlangeDefinition{
		EdgeKey: key, Profile: profile, Flip: d.Flip, Width: width,
	}), nil
}
