// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// Serialized forms of the surface-editing features (M10-F01/F02): trim, extend, surface offset,
// mid-surface, stitch, and sculpt. These kinds were creatable but carried no serialization codec,
// so any part containing one failed to marshal — a save/undo silently refused to record the edit
// ("no serialization codec for feature kind …", #1416/#1617). Each parameter-driven distance is
// stored as its evaluated value and restored as a constant closure, the same convention the extrude
// and dress-up codecs use.

// TrimData is a surface trim's recipe: the cutting plane (origin+normal) and which side to keep.
type TrimData struct {
	Origin       []float64 `yaml:"origin"`
	Normal       []float64 `yaml:"normal"`
	KeepPositive bool      `yaml:"keepPositive"`
}

func serializeTrim(def *TrimDefinition) *TrimData {
	return &TrimData{Origin: encodePoint3(def.CutOrigin), Normal: encodeVec3(def.CutNormal), KeepPositive: def.KeepPositive}
}

func restoreTrim(fs *PartFeatures, d *TrimData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("trim feature is missing its payload")
	}
	return NewTrimFeatures(fs).AddByPlane(decodePoint3(d.Origin), decodeVec3(d.Normal), d.KeepPositive), nil
}

// ExtendData is a surface extend's recipe: the boundary edge (base64 reference key) and the
// outward distance.
type ExtendData struct {
	EdgeKey  string  `yaml:"edgeKey"`
	Distance float64 `yaml:"distance"`
}

func serializeExtend(def *ExtendDefinition) *ExtendData {
	return &ExtendData{EdgeKey: encodeKey(def.EdgeKey), Distance: evalFloat(def.Distance)}
}

func restoreExtend(fs *PartFeatures, d *ExtendData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("extend feature is missing its payload")
	}
	key, err := decodeKey(d.EdgeKey)
	if err != nil {
		return nil, fmt.Errorf("extend feature edge key: %w", err)
	}
	return NewExtendFeatures(fs).Add(key, constFloat(d.Distance)), nil
}

// SurfaceOffsetData is a surface offset's recipe: the offset distance along the face normal.
type SurfaceOffsetData struct {
	Distance float64 `yaml:"distance"`
}

func serializeSurfaceOffset(def *SurfaceOffsetDefinition) *SurfaceOffsetData {
	return &SurfaceOffsetData{Distance: evalFloat(def.Distance)}
}

func restoreSurfaceOffset(fs *PartFeatures, d *SurfaceOffsetData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("surface-offset feature is missing its payload")
	}
	return NewSurfaceOffsetFeatures(fs).AddByDistance(constFloat(d.Distance)), nil
}

// MidSurfaceData is a mid-surface's recipe: the maximum thin-wall thickness a face pair may have.
// The extracted per-pair thicknesses are a recompute result, not part of the recipe.
type MidSurfaceData struct {
	MaxThickness float64 `yaml:"maxThickness"`
}

func serializeMidSurface(def *MidSurfaceDefinition) *MidSurfaceData {
	return &MidSurfaceData{MaxThickness: def.MaxThickness}
}

func restoreMidSurface(fs *PartFeatures, d *MidSurfaceData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("mid-surface feature is missing its payload")
	}
	return NewMidSurfaceFeatures(fs).AddByThickness(d.MaxThickness), nil
}

// StitchData is a stitch/knit's recipe: the coincidence tolerance and whether to keep a closed
// quilt as a surface rather than promoting it to a solid.
type StitchData struct {
	Tolerance         float64 `yaml:"tolerance"`
	MaintainAsSurface bool    `yaml:"maintainAsSurface"`
}

func serializeStitch(def *StitchDefinition) *StitchData {
	return &StitchData{Tolerance: def.Tolerance, MaintainAsSurface: def.MaintainAsSurface}
}

func restoreStitch(fs *PartFeatures, d *StitchData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("stitch feature is missing its payload")
	}
	return NewStitchFeatures(fs).Add(d.Tolerance, d.MaintainAsSurface), nil
}

// SculptData is a sculpt's recipe: the boolean operation against existing material and the
// coincidence tolerance for closing the bounding surfaces.
type SculptData struct {
	Operation string  `yaml:"operation"`
	Tolerance float64 `yaml:"tolerance"`
}

func serializeSculpt(def *SculptDefinition) (*SculptData, error) {
	op, err := operationName(def.Operation)
	if err != nil {
		return nil, err
	}
	return &SculptData{Operation: op, Tolerance: def.Tolerance}, nil
}

func restoreSculpt(fs *PartFeatures, d *SculptData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sculpt feature is missing its payload")
	}
	op, err := parseOperation(d.Operation)
	if err != nil {
		return nil, fmt.Errorf("sculpt feature operation: %w", err)
	}
	return NewSculptFeatures(fs).Add(op, d.Tolerance), nil
}
