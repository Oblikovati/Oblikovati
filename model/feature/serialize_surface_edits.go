// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

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

// ExtendData is a surface extend's recipe (#1878): the boundary edges (base64 reference keys), the
// outward distance or a target plane (extend-to-plane), and the continuity mode. EdgeKey is the
// legacy single-edge field, read when Edges is empty so pre-#1878 recipes restore unchanged.
type ExtendData struct {
	EdgeKey      string    `yaml:"edgeKey,omitempty"`
	Edges        []string  `yaml:"edges,omitempty"`
	Distance     float64   `yaml:"distance,omitempty"`
	TargetOrigin []float64 `yaml:"targetOrigin,omitempty"` // extend-to-plane target (origin + normal)
	TargetNormal []float64 `yaml:"targetNormal,omitempty"`
	Natural      bool      `yaml:"natural,omitempty"`
}

func serializeExtend(def *ExtendDefinition) *ExtendData {
	d := &ExtendData{Edges: encodeKeys(def.EdgeKeys), Distance: evalFloat(def.Distance), Natural: def.Natural}
	if p := def.TargetPlane; p != nil {
		o, n := p.Origin, p.Normal()
		d.TargetOrigin = []float64{float64(o.X), float64(o.Y), float64(o.Z)}
		d.TargetNormal = []float64{float64(n.X), float64(n.Y), float64(n.Z)}
	}
	return d
}

func restoreExtend(fs *PartFeatures, d *ExtendData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("extend feature is missing its payload")
	}
	keys, err := extendEdgeKeys(d)
	if err != nil {
		return nil, err
	}
	def := &ExtendDefinition{EdgeKeys: keys, Distance: constFloat(d.Distance), Natural: d.Natural}
	if len(d.TargetNormal) == 3 && len(d.TargetOrigin) == 3 {
		pl, err := geom.NewPlane(math.P3(d.TargetOrigin[0], d.TargetOrigin[1], d.TargetOrigin[2]), math.V3(d.TargetNormal[0], d.TargetNormal[1], d.TargetNormal[2]))
		if err != nil {
			return nil, fmt.Errorf("extend feature target plane: %w", err)
		}
		def.TargetPlane = &pl
	}
	return NewExtendFeatures(fs).AddExtend(def), nil
}

// extendEdgeKeys decodes the multi-edge Edges list, falling back to the legacy single EdgeKey.
func extendEdgeKeys(d *ExtendData) ([][]byte, error) {
	if len(d.Edges) > 0 {
		return decodeKeys(d.Edges)
	}
	key, err := decodeKey(d.EdgeKey)
	if err != nil {
		return nil, fmt.Errorf("extend feature edge key: %w", err)
	}
	return [][]byte{key}, nil
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

// MidSurfaceData is a mid-surface's recipe (#1885): the auto-pairing thickness range, optional
// input-body selection, and optional manual face-key pairs. The extracted per-pair thicknesses are
// a recompute result, not part of the recipe.
type MidSurfaceData struct {
	MaxThickness float64     `yaml:"maxThickness"`
	MinThickness float64     `yaml:"minThickness,omitempty"`
	BodyIndices  []int       `yaml:"bodyIndices,omitempty"`
	Pairs        [][2]string `yaml:"pairs,omitempty"` // base64 face-key pairs
}

func serializeMidSurface(def *MidSurfaceDefinition) *MidSurfaceData {
	d := &MidSurfaceData{MaxThickness: def.MaxThickness, MinThickness: def.MinThickness, BodyIndices: def.BodyIndices}
	for _, pr := range def.Pairs {
		d.Pairs = append(d.Pairs, [2]string{encodeKey(pr[0]), encodeKey(pr[1])})
	}
	return d
}

func restoreMidSurface(fs *PartFeatures, d *MidSurfaceData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("mid-surface feature is missing its payload")
	}
	pairs, err := decodeKeyPairs(d.Pairs)
	if err != nil {
		return nil, fmt.Errorf("mid-surface feature pairs: %w", err)
	}
	return NewMidSurfaceFeatures(fs).AddMidSurface(&MidSurfaceDefinition{
		MaxThickness: d.MaxThickness, MinThickness: d.MinThickness, BodyIndices: d.BodyIndices, Pairs: pairs,
	}), nil
}

// decodeKeyPairs decodes a list of base64 face-key pairs.
func decodeKeyPairs(encoded [][2]string) ([][2][]byte, error) {
	if len(encoded) == 0 {
		return nil, nil
	}
	out := make([][2][]byte, len(encoded))
	for i, pr := range encoded {
		a, err := decodeKey(pr[0])
		if err != nil {
			return nil, err
		}
		b, err := decodeKey(pr[1])
		if err != nil {
			return nil, err
		}
		out[i] = [2][]byte{a, b}
	}
	return out, nil
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
	Operation     string  `yaml:"operation"`
	Tolerance     float64 `yaml:"tolerance"`
	Directions    []bool  `yaml:"directions,omitempty"`    // #1881: per-surface keep-positive
	BodyIndices   []int   `yaml:"bodyIndices,omitempty"`   // #1881: bounding-surface selection
	AffectedIndex *int    `yaml:"affectedIndex,omitempty"` // #1881: join/cut target
}

func serializeSculpt(def *SculptDefinition) (*SculptData, error) {
	op, err := operationName(def.Operation)
	if err != nil {
		return nil, err
	}
	return &SculptData{
		Operation: op, Tolerance: def.Tolerance,
		Directions: def.Directions, BodyIndices: def.BodyIndices, AffectedIndex: def.AffectedIndex,
	}, nil
}

func restoreSculpt(fs *PartFeatures, d *SculptData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sculpt feature is missing its payload")
	}
	op, err := parseOperation(d.Operation)
	if err != nil {
		return nil, fmt.Errorf("sculpt feature operation: %w", err)
	}
	return NewSculptFeatures(fs).AddSculpt(&SculptDefinition{
		Operation: op, Tolerance: d.Tolerance,
		Directions: d.Directions, BodyIndices: d.BodyIndices, AffectedIndex: d.AffectedIndex,
	}), nil
}
