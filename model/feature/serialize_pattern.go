// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati/math"
)

// This file holds the YAML codecs for pattern features (rectangular/circular/
// sketch-driven) and mirror. These replicate earlier features, recorded as program
// indices (feature ids are session-local, never persisted) and resolved back to the
// restored features on open, plus the occurrence geometry (grid step, axis, sketch
// points, mirror plane) so the placed copies regenerate on the next recompute.

// RectPatternData replicates source features in a 2D grid stepping by StepX/StepY.
type RectPatternData struct {
	Source []int     `yaml:"source"`
	CountX int       `yaml:"countX"`
	CountY int       `yaml:"countY"`
	StepX  []float64 `yaml:"stepX"`
	StepY  []float64 `yaml:"stepY"`
}

// CircPatternData replicates source features around an axis.
type CircPatternData struct {
	Source    []int     `yaml:"source"`
	Count     int       `yaml:"count"`
	Angle     float64   `yaml:"angle"`
	AxisPoint []float64 `yaml:"axisPoint"`
	AxisDir   []float64 `yaml:"axisDir"`
}

// SketchDrivenPatternData places one occurrence of the source per sketch point.
type SketchDrivenPatternData struct {
	Source []int       `yaml:"source"`
	Points [][]float64 `yaml:"points"`
}

// MirrorData reflects source features across a plane (a reference key + geometry).
type MirrorData struct {
	Source []int     `yaml:"source"`
	Plane  string    `yaml:"plane"`
	Origin []float64 `yaml:"origin"`
	Normal []float64 `yaml:"normal"`
}

// encodeVec3 / encodePoint3 serialize a 3-vector / point as [x,y,z]; encodePoints maps
// a point slice. decodeVec3 / decodePoint3 invert them (a missing/short slice → zero).
func encodeVec3(v math.Vector3) []float64  { return []float64{v.X, v.Y, v.Z} }
func encodePoint3(p math.Point3) []float64 { return []float64{p.X, p.Y, p.Z} }

func encodePoints(ps []math.Point3) [][]float64 {
	out := make([][]float64, len(ps))
	for i, p := range ps {
		out[i] = encodePoint3(p)
	}
	return out
}

func decodeVec3(s []float64) math.Vector3 {
	if len(s) < 3 {
		return math.V3(0, 0, 0)
	}
	return math.V3(s[0], s[1], s[2])
}

func decodePoint3(s []float64) math.Point3 {
	if len(s) < 3 {
		return math.P3(0, 0, 0)
	}
	return math.P3(s[0], s[1], s[2])
}

func decodePoints(s [][]float64) []math.Point3 {
	out := make([]math.Point3, len(s))
	for i, p := range s {
		out[i] = decodePoint3(p)
	}
	return out
}

func restoreRectPattern(fs *PartFeatures, d *RectPatternData, restored []*PartFeature) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("rectangular-pattern feature is missing its payload")
	}
	src, err := resolveSources(d.Source, restored)
	if err != nil {
		return nil, err
	}
	NewPatternFeatures(fs).AddRectangular(src, constInt(d.CountX), constInt(d.CountY), decodeVec3(d.StepX), decodeVec3(d.StepY))
	return lastFeature(fs), nil
}

func restoreCircPattern(fs *PartFeatures, d *CircPatternData, restored []*PartFeature) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("circular-pattern feature is missing its payload")
	}
	src, err := resolveSources(d.Source, restored)
	if err != nil {
		return nil, err
	}
	NewPatternFeatures(fs).AddCircular(src, constInt(d.Count), constFloat(d.Angle), decodePoint3(d.AxisPoint), decodeVec3(d.AxisDir))
	return lastFeature(fs), nil
}

func restoreSketchPattern(fs *PartFeatures, d *SketchDrivenPatternData, restored []*PartFeature) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sketch-driven-pattern feature is missing its payload")
	}
	src, err := resolveSources(d.Source, restored)
	if err != nil {
		return nil, err
	}
	points := decodePoints(d.Points)
	NewPatternFeatures(fs).AddSketchDriven(src, func() []math.Point3 { return points })
	return lastFeature(fs), nil
}

func restoreMirror(fs *PartFeatures, d *MirrorData, restored []*PartFeature) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("mirror feature is missing its payload")
	}
	src, err := resolveSources(d.Source, restored)
	if err != nil {
		return nil, err
	}
	key, err := decodeKey(d.Plane)
	if err != nil {
		return nil, err
	}
	NewPatternFeatures(fs).AddMirror(src, key, decodePoint3(d.Origin), decodeVec3(d.Normal))
	return lastFeature(fs), nil
}

// sourceIndices maps a pattern's source feature ids to their program positions for
// serialization, erroring if a source is not part of the program.
func sourceIndices(ids []ID, idx map[ID]int) ([]int, error) {
	out := make([]int, len(ids))
	for i, id := range ids {
		pos, ok := idx[id]
		if !ok {
			return nil, fmt.Errorf("pattern source feature id %d is not in the program", id)
		}
		out[i] = pos
	}
	return out, nil
}

// resolveSources maps program positions back to the restored features' ids, erroring
// on an out-of-range index (a source must precede the pattern that consumes it).
func resolveSources(indices []int, restored []*PartFeature) ([]ID, error) {
	out := make([]ID, len(indices))
	for i, pos := range indices {
		if pos < 0 || pos >= len(restored) {
			return nil, fmt.Errorf("pattern source index %d is out of range (%d features restored so far)", pos, len(restored))
		}
		out[i] = restored[pos].ID()
	}
	return out, nil
}

// lastFeature returns the most recently added feature — the pattern Add methods return
// the concrete feature, not the engine's PartFeature wrapper, so restore reads it back.
func lastFeature(fs *PartFeatures) *PartFeature { return fs.Item(fs.Count() - 1) }
