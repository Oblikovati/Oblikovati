// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// This file holds the YAML codecs for pattern features (rectangular/circular/
// sketch-driven) and mirror. These replicate earlier features, recorded as program
// indices (feature ids are session-local, never persisted) and resolved back to the
// restored features on open. Mirror also references a mirror plane by reference key.

// RectPatternData replicates source features in a 2D grid.
type RectPatternData struct {
	Source []int `yaml:"source"`
	CountX int   `yaml:"countX"`
	CountY int   `yaml:"countY"`
}

// CircPatternData replicates source features around an axis.
type CircPatternData struct {
	Source []int   `yaml:"source"`
	Count  int     `yaml:"count"`
	Angle  float64 `yaml:"angle"`
}

// SketchDrivenPatternData places one occurrence of the source per sketch point.
type SketchDrivenPatternData struct {
	Source     []int `yaml:"source"`
	PointCount int   `yaml:"pointCount"`
}

// MirrorData reflects source features across a plane (a reference key).
type MirrorData struct {
	Source []int  `yaml:"source"`
	Plane  string `yaml:"plane"`
}

func restoreRectPattern(fs *PartFeatures, d *RectPatternData, restored []*PartFeature) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("rectangular-pattern feature is missing its payload")
	}
	src, err := resolveSources(d.Source, restored)
	if err != nil {
		return nil, err
	}
	NewPatternFeatures(fs).AddRectangular(src, constInt(d.CountX), constInt(d.CountY))
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
	NewPatternFeatures(fs).AddCircular(src, constInt(d.Count), constFloat(d.Angle))
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
	NewPatternFeatures(fs).AddSketchDriven(src, constInt(d.PointCount))
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
	NewPatternFeatures(fs).AddMirror(src, key)
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
