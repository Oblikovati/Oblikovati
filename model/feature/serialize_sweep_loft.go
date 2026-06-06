// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati/math"
	"oblikovati/model/sketch"
)

// YAML codecs for sweep and loft. A sweep persists its profile (sketch+index), its
// 3D path points (model space), the twist, and the operation. A loft persists its
// ordered sections (each a sketch index + profile index), the closed flag, and the
// operation. The path's 3D points are stored directly (a 3D-sketch path model is a
// later refinement).

// SweepData is a sweep's recipe.
type SweepData struct {
	Sketch    int         `yaml:"sketch"`
	Profile   int         `yaml:"profile"`
	Path      [][]float64 `yaml:"path"`
	Closed    bool        `yaml:"closed,omitempty"`
	Twist     float64     `yaml:"twist,omitempty"`
	Operation string      `yaml:"operation"`
}

// LoftSectionData is one section of a loft: a profile on a sketch, or — when Point is set — an
// apex (a point section). Point is model-space [x,y,z]; Sketch still gives the apex's plane (the
// tangent plane a TangentToPlane condition domes against).
type LoftSectionData struct {
	Sketch  int       `yaml:"sketch"`
	Profile int       `yaml:"profile"`
	Point   []float64 `yaml:"point,omitempty"`
}

// LoftEndData is a loft end-section condition (omitted entirely when Free). Angle is in radians.
type LoftEndData struct {
	Condition string  `yaml:"condition"`
	Angle     float64 `yaml:"angle,omitempty"`
	Impact    float64 `yaml:"impact,omitempty"`
	Reversed  bool    `yaml:"reversed,omitempty"`
}

// LoftData is a loft's recipe. First/Last persist the end conditions (nil when Free).
type LoftData struct {
	Sections  []LoftSectionData `yaml:"sections"`
	Closed    bool              `yaml:"closed,omitempty"`
	Operation string            `yaml:"operation"`
	First     *LoftEndData      `yaml:"first,omitempty"`
	Last      *LoftEndData      `yaml:"last,omitempty"`
}

func serializeSweep(def *SweepDefinition, sk SketchIndexer) (*SweepData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("sweep references a sketch that is not in the part")
	}
	if def.Path == nil {
		return nil, fmt.Errorf("sweep has no path")
	}
	path := def.Path() // serialize the current path geometry (evaluated, like other args)
	if path == nil {
		return nil, fmt.Errorf("sweep has no path")
	}
	op, err := operationName(def.Operation)
	if err != nil {
		return nil, err
	}
	return &SweepData{
		Sketch: idx, Profile: def.ProfileIndex, Path: encodePoints(path.Points()),
		Closed: path.IsClosed(), Twist: evalFloat(def.Twist), Operation: op,
	}, nil
}

func restoreSweep(fs *PartFeatures, d *SweepData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sweep feature is missing its payload")
	}
	skt, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("sweep references sketch index %d, which does not exist", d.Sketch)
	}
	op, err := parseOperation(d.Operation)
	if err != nil {
		return nil, err
	}
	path := sketch.NewPath3D(point3DChain(decodePoints(d.Path)), d.Closed)
	twist := d.Twist
	return NewSweepFeatures(fs).Add(skt, d.Profile, path, func() float64 { return twist }, op), nil
}

func serializeLoft(def *LoftDefinition, sk SketchIndexer) (*LoftData, error) {
	sections := make([]LoftSectionData, len(def.Sections))
	for i, s := range def.Sections {
		idx, ok := sk.IndexOf(s.Sketch)
		if !ok {
			return nil, fmt.Errorf("loft section %d references a sketch that is not in the part", i)
		}
		sd := LoftSectionData{Sketch: idx, Profile: s.ProfileIndex}
		if s.IsPoint() {
			sd.Point = []float64{s.Point.X, s.Point.Y, s.Point.Z}
		}
		sections[i] = sd
	}
	op, err := operationName(def.Operation)
	if err != nil {
		return nil, err
	}
	first, last := def.First, def.Last
	if def.LiveEnds != nil {
		first, last = def.LiveEnds() // capture the current (evaluated) end values, like sweep's twist
	}
	return &LoftData{Sections: sections, Closed: def.Closed, Operation: op, First: encodeLoftEnd(first), Last: encodeLoftEnd(last)}, nil
}

// encodeLoftEnd serializes an end condition, returning nil for a Free end (the default) so a
// plain loft persists no condition keys.
func encodeLoftEnd(e LoftEnd) *LoftEndData {
	if e.Condition.IsFree() {
		return nil
	}
	return &LoftEndData{Condition: string(e.Condition), Angle: e.Angle, Impact: e.Impact, Reversed: e.Reversed}
}

// decodeLoftEnd is encodeLoftEnd's inverse (a nil payload is a Free end).
func decodeLoftEnd(d *LoftEndData) LoftEnd {
	if d == nil {
		return LoftEnd{}
	}
	return LoftEnd{Condition: LoftCondition(d.Condition), Angle: d.Angle, Impact: d.Impact, Reversed: d.Reversed}
}

func restoreLoft(fs *PartFeatures, d *LoftData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("loft feature is missing its payload")
	}
	sections := make([]LoftSection, len(d.Sections))
	for i, s := range d.Sections {
		skt, ok := sk.At(s.Sketch)
		if !ok {
			return nil, fmt.Errorf("loft section %d references sketch index %d, which does not exist", i, s.Sketch)
		}
		ls := LoftSection{Sketch: skt, ProfileIndex: s.Profile}
		if len(s.Point) == 3 {
			p := math.P3(s.Point[0], s.Point[1], s.Point[2])
			ls.Point = &p
		}
		sections[i] = ls
	}
	op, err := parseOperation(d.Operation)
	if err != nil {
		return nil, err
	}
	return NewLoftFeatures(fs).AddConditioned(sections, d.Closed, op, decodeLoftEnd(d.First), decodeLoftEnd(d.Last)), nil
}

// point3DChain wraps model points as sketch 3D points for a restored sweep path.
func point3DChain(pts []math.Point3) []*sketch.Point3D {
	out := make([]*sketch.Point3D, len(pts))
	for i, p := range pts {
		out[i] = sketch.NewPoint3D(p)
	}
	return out
}
