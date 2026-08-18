// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
)

// SheetMetalCornerData is the serialized form of a SheetMetalCornerFeature: the corner edges
// (base64 reference keys), the treatment, its size, and — the variant settings (#1967) — the
// chamfer type / second distance / angle / reference face, and the round's per-radius edge sets.
type SheetMetalCornerData struct {
	Edges       []string             `yaml:"edges"`
	Treatment   int32                `yaml:"treatment,omitempty"`
	Size        float64              `yaml:"size"`
	ChamferType int32                `yaml:"chamferType,omitempty"`
	Distance2   float64              `yaml:"distance2,omitempty"`
	Angle       float64              `yaml:"angle,omitempty"`
	FaceKey     string               `yaml:"faceKey,omitempty"`
	RoundSets   []CornerRoundSetData `yaml:"roundSets,omitempty"`
}

// CornerRoundSetData is one persisted round edge set: its edges and radius.
type CornerRoundSetData struct {
	Edges  []string `yaml:"edges"`
	Radius float64  `yaml:"radius"`
}

// serializeSheetMetalCorner projects a corner recipe to its persisted form.
func serializeSheetMetalCorner(def *SheetMetalCornerDefinition) *SheetMetalCornerData {
	return &SheetMetalCornerData{
		Edges:       encodeKeys(def.EdgeKeys),
		Treatment:   int32(def.Treatment),
		Size:        evalFloat(def.Size),
		ChamferType: int32(def.ChamferType),
		Distance2:   evalFloat(def.Distance2),
		Angle:       evalFloat(def.Angle),
		FaceKey:     encodeKey(def.FaceKey),
		RoundSets:   serializeRoundSets(def.RoundSets),
	}
}

// serializeRoundSets projects the round edge sets to their persisted form.
func serializeRoundSets(sets []CornerRoundSet) []CornerRoundSetData {
	if len(sets) == 0 {
		return nil
	}
	out := make([]CornerRoundSetData, len(sets))
	for i, s := range sets {
		out[i] = CornerRoundSetData{Edges: encodeKeys(s.EdgeKeys), Radius: evalFloat(s.Radius)}
	}
	return out
}

// restoreSheetMetalCorner rebuilds a corner feature, erroring on a missing payload or an
// undecodable edge key.
func restoreSheetMetalCorner(fs *PartFeatures, d *SheetMetalCornerData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sheet-metal corner feature is missing its payload")
	}
	keys, err := decodeKeys(d.Edges)
	if err != nil {
		return nil, fmt.Errorf("sheet-metal corner edge keys: %w", err)
	}
	face, err := decodeKey(d.FaceKey)
	if err != nil {
		return nil, fmt.Errorf("sheet-metal corner face key: %w", err)
	}
	sets, err := restoreRoundSets(d.RoundSets)
	if err != nil {
		return nil, err
	}
	return NewSheetMetalCornerFeatures(fs).Add(&SheetMetalCornerDefinition{
		EdgeKeys: keys, Treatment: CornerTreatment(d.Treatment), Size: constFloat(d.Size),
		ChamferType: types.ChamferType(d.ChamferType), Distance2: constFloat(d.Distance2),
		Angle: constFloat(d.Angle), FaceKey: face, RoundSets: sets,
	}), nil
}

// restoreRoundSets rebuilds the round edge sets, erroring on an undecodable edge key.
func restoreRoundSets(data []CornerRoundSetData) ([]CornerRoundSet, error) {
	if len(data) == 0 {
		return nil, nil
	}
	sets := make([]CornerRoundSet, len(data))
	for i, d := range data {
		keys, err := decodeKeys(d.Edges)
		if err != nil {
			return nil, fmt.Errorf("sheet-metal corner round set %d edge keys: %w", i, err)
		}
		radius := d.Radius
		sets[i] = CornerRoundSet{EdgeKeys: keys, Radius: constFloat(radius)}
	}
	return sets, nil
}
