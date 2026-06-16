// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// SheetMetalCutData is the serialized form of a SheetMetalCutFeature: the profile (sketch
// index + region index), the cut side, an optional depth (0 ⇒ through all), and the
// across-bend flag.
type SheetMetalCutData struct {
	Sketch     int     `yaml:"sketch"`
	Profile    int     `yaml:"profile"`
	Direction  int32   `yaml:"direction,omitempty"`
	Distance   float64 `yaml:"distance,omitempty"` // 0 ⇒ through all
	AcrossBend bool    `yaml:"acrossBend,omitempty"`
}

// serializeSheetMetalCut projects a cut recipe to its persisted form, erroring if its sketch
// is not in the part.
func serializeSheetMetalCut(def *SheetMetalCutDefinition, sk SketchIndexer) (*SheetMetalCutData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("sheet-metal cut references a sketch that is not in the part")
	}
	return &SheetMetalCutData{
		Sketch: idx, Profile: def.ProfileIndex, Direction: int32(def.Direction),
		Distance: evalFloat(def.Distance), AcrossBend: def.AcrossBend,
	}, nil
}

// restoreSheetMetalCut rebuilds a cut feature, erroring on a missing payload or unknown
// sketch. A zero distance restores as nil (cut through all).
func restoreSheetMetalCut(fs *PartFeatures, d *SheetMetalCutData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sheet-metal cut feature is missing its payload")
	}
	s, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("sheet-metal cut references sketch %d which is not in the part", d.Sketch)
	}
	def := &SheetMetalCutDefinition{Sketch: s, ProfileIndex: d.Profile, Direction: ExtentDirection(d.Direction), AcrossBend: d.AcrossBend}
	if d.Distance != 0 {
		def.Distance = constFloat(d.Distance)
	}
	return NewSheetMetalCutFeatures(fs).Add(def), nil
}
