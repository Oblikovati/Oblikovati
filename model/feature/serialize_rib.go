// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// RibData is a rib's recipe: its open profile (a sketch + path index), wall thickness, signed
// depth along the sketch-plane normal, and the boolean operation joining it to the part.
type RibData struct {
	Sketch    int     `yaml:"sketch"`
	Profile   int     `yaml:"profile"`
	Thickness float64 `yaml:"thickness"`
	Depth     float64 `yaml:"depth,omitempty"`
	ToNext    bool    `yaml:"toNext,omitempty"` // extend to the existing material (#316)
	Operation string  `yaml:"operation"`
	// The wall's cross-section options (#1882). Each is omitted when it holds the pre-#1882
	// default, so an older document reads back as the symmetric, undrafted wall it described.
	ThickenSide   string  `yaml:"thickenSide,omitempty"` // "" | side1 | side2
	Draft         float64 `yaml:"draft,omitempty"`       // radians
	ThicknessAt   string  `yaml:"thicknessAt,omitempty"` // "" (sketch plane) | root
	ExtendProfile bool    `yaml:"extendProfile,omitempty"`
}

// ribThickenSideNames is the persisted spelling of each thicken side, both ways. The symmetric
// default is the empty string so it drops out of the YAML entirely.
var ribThickenSideNames = map[RibThickenSide]string{
	RibThickenSymmetric: "", RibThickenSide1: "side1", RibThickenSide2: "side2",
}

// ribThickenSideFromName reads a persisted thicken side; an unknown spelling is a precise error
// rather than a silent fall back to symmetric, which would move the wall off the path.
func ribThickenSideFromName(name string) (RibThickenSide, error) {
	for side, n := range ribThickenSideNames {
		if n == name {
			return side, nil
		}
	}
	return 0, fmt.Errorf("rib: unknown thickenSide %q in the recipe (want side1, side2 or omitted for symmetric)", name)
}

func serializeRib(def *RibDefinition, sk SketchIndexer) (*RibData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("rib references a sketch that is not in the part")
	}
	op, err := operationName(def.Operation)
	if err != nil {
		return nil, err
	}
	return &RibData{
		Sketch: idx, Profile: def.ProfileIndex,
		Thickness: evalFloat(def.Thickness), Depth: evalFloat(def.Depth),
		ToNext: def.ToNext, Operation: op,
		ThickenSide: ribThickenSideNames[def.ThickenSide], Draft: def.Draft,
		ThicknessAt: ribThicknessAtName(def.HoldThicknessAtRoot), ExtendProfile: def.ExtendProfile,
	}, nil
}

func restoreRib(fs *PartFeatures, d *RibData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("rib feature is missing its payload")
	}
	skt, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("rib references sketch index %d, which does not exist", d.Sketch)
	}
	op, err := parseOperation(d.Operation)
	if err != nil {
		return nil, err
	}
	side, err := ribThickenSideFromName(d.ThickenSide)
	if err != nil {
		return nil, err
	}
	def := &RibDefinition{
		Sketch: skt, ProfileIndex: d.Profile,
		Thickness: constFloat(d.Thickness), Depth: constFloat(d.Depth),
		ToNext: d.ToNext, Operation: op,
		ThickenSide: side, Draft: d.Draft,
		HoldThicknessAtRoot: d.ThicknessAt == ribThicknessAtRootName, ExtendProfile: d.ExtendProfile,
	}
	return NewRibFeatures(fs).AddDefinition(def), nil
}

// ribThicknessAtRootName is the persisted spelling for holding the thickness at the root; the
// sketch-plane default persists as the empty string.
const ribThicknessAtRootName = "root"

// ribThicknessAtName is the persisted spelling of the thickness plane.
func ribThicknessAtName(atRoot bool) string {
	if atRoot {
		return ribThicknessAtRootName
	}
	return ""
}
