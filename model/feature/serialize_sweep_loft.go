// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
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
	// The definition union (M08 PBI-094, #314):
	Orientation   string                  `yaml:"orientation,omitempty"` // SweepProfileOrientation spelling
	AlignVector   []float64               `yaml:"alignVector,omitempty"`
	Taper         float64                 `yaml:"taper,omitempty"`
	TwistStations []SweepTwistStationData `yaml:"twistStations,omitempty"`
	Rail          [][]float64             `yaml:"rail,omitempty"`
	Scaling       string                  `yaml:"scaling,omitempty"` // SweepProfileScaling spelling
	GuideFace     []byte                  `yaml:"guideFace,omitempty"`
	SolidTool     *int                    `yaml:"solidTool,omitempty"` // running-body index of the tool
}

// SweepTwistStationData is one persisted twist station.
type SweepTwistStationData struct {
	T     float64 `yaml:"t"`
	Angle float64 `yaml:"angle"`
}

// LoftSectionData is one section of a loft: a profile on a sketch; a point (apex) section
// (Point = model [x,y,z]); or a body-face section (FaceKey = the face reference key, resolved
// against the running bodies). Sketch is -1 when the section carries no sketch (a face section,
// or a bare 3D point).
type LoftSectionData struct {
	Sketch  int       `yaml:"sketch"`
	Profile int       `yaml:"profile"`
	Point   []float64 `yaml:"point,omitempty"`
	FaceKey []byte    `yaml:"faceKey,omitempty"`
}

// LoftEndData is a loft end-section condition (omitted entirely when Free). Angle is in radians.
type LoftEndData struct {
	Condition string  `yaml:"condition"`
	Angle     float64 `yaml:"angle,omitempty"`
	Impact    float64 `yaml:"impact,omitempty"`
	Reversed  bool    `yaml:"reversed,omitempty"`
}

// LoftAreaStopData is one area-graph control point (t, scale).
type LoftAreaStopData struct {
	T     float64 `yaml:"t"`
	Scale float64 `yaml:"scale"`
}

// LoftData is a loft's recipe. First/Last persist the end conditions (nil when Free); Rails,
// Centerline, AreaGraph and MapCurves persist the guides (model-space polylines / scale stops,
// evaluated like the sweep path).
type LoftData struct {
	Sections   []LoftSectionData  `yaml:"sections"`
	Rails      [][][]float64      `yaml:"rails,omitempty"`
	Centerline [][]float64        `yaml:"centerline,omitempty"`
	AreaGraph  []LoftAreaStopData `yaml:"areaGraph,omitempty"`
	MapCurves  [][][]float64      `yaml:"mapCurves,omitempty"`
	Closed     bool               `yaml:"closed,omitempty"`
	Operation  string             `yaml:"operation"`
	First      *LoftEndData       `yaml:"first,omitempty"`
	Last       *LoftEndData       `yaml:"last,omitempty"`
}

func serializeSweep(def *SweepDefinition, sk SketchIndexer) (*SweepData, error) {
	idx := -1 // a solid sweep carries no profile sketch
	if def.Sketch != nil {
		var ok bool
		if idx, ok = sk.IndexOf(def.Sketch); !ok {
			return nil, fmt.Errorf("sweep references a sketch that is not in the part")
		}
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
	d := &SweepData{
		Sketch: idx, Profile: def.ProfileIndex, Path: encodePoints(path.Points()),
		Closed: path.IsClosed(), Twist: evalFloat(def.Twist), Operation: op,
	}
	return d, serializeSweepUnion(d, def)
}

// serializeSweepUnion persists the M08 union fields (#314).
func serializeSweepUnion(d *SweepData, def *SweepDefinition) error {
	if def.Orientation != 0 {
		d.Orientation = def.Orientation.String()
	}
	if float64(def.AlignVector.Length()) > 0 {
		d.AlignVector = []float64{float64(def.AlignVector.X), float64(def.AlignVector.Y), float64(def.AlignVector.Z)}
	}
	d.Taper = evalFloat(def.Taper)
	for _, st := range def.TwistStations {
		d.TwistStations = append(d.TwistStations, SweepTwistStationData(st))
	}
	if def.GuideRail != nil {
		rail := def.GuideRail()
		if rail == nil {
			return fmt.Errorf("sweep guide rail provider returned nothing")
		}
		d.Rail = encodePoints(rail.Points())
	}
	if def.Scaling != 0 {
		d.Scaling = def.Scaling.String()
	}
	d.GuideFace = def.GuideFaceKey
	d.SolidTool = def.SolidToolIndex
	return nil
}

func restoreSweep(fs *PartFeatures, d *SweepData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sweep feature is missing its payload")
	}
	var skt *sketch.Sketch
	if d.Sketch >= 0 { // -1 = a solid sweep (no profile sketch)
		var ok bool
		if skt, ok = sk.At(d.Sketch); !ok {
			return nil, fmt.Errorf("sweep references sketch index %d, which does not exist", d.Sketch)
		}
	}
	op, err := parseOperation(d.Operation)
	if err != nil {
		return nil, err
	}
	path := sketch.NewPath3D(point3DChain(decodePoints(d.Path)), d.Closed)
	def := &SweepDefinition{
		Sketch: skt, ProfileIndex: d.Profile,
		Path:      func() *sketch.Path3D { return path },
		Twist:     constFloat(d.Twist),
		Operation: op,
	}
	if err := restoreSweepUnion(def, d); err != nil {
		return nil, err
	}
	return NewSweepFeatures(fs).AddDefinition(def), nil
}

// restoreSweepUnion rebuilds the M08 union fields (#314).
func restoreSweepUnion(def *SweepDefinition, d *SweepData) error {
	if d.Orientation != "" {
		o, ok := types.ParseSweepProfileOrientation(d.Orientation)
		if !ok {
			return fmt.Errorf("sweep: unknown orientation %q", d.Orientation)
		}
		def.Orientation = o
	}
	if len(d.AlignVector) == 3 {
		def.AlignVector = math.V3(math.Scalar(d.AlignVector[0]), math.Scalar(d.AlignVector[1]), math.Scalar(d.AlignVector[2]))
	}
	def.Taper = constFloat(d.Taper)
	for _, st := range d.TwistStations {
		def.TwistStations = append(def.TwistStations, SweepTwistStation(st))
	}
	if len(d.Rail) > 0 {
		rail := sketch.NewPath3D(point3DChain(decodePoints(d.Rail)), false)
		def.GuideRail = func() *sketch.Path3D { return rail }
	}
	if d.Scaling != "" {
		sc, ok := types.ParseSweepProfileScaling(d.Scaling)
		if !ok {
			return fmt.Errorf("sweep: unknown scaling %q", d.Scaling)
		}
		def.Scaling = sc
	}
	def.GuideFaceKey = d.GuideFace
	def.SolidToolIndex = d.SolidTool
	return nil
}


func serializeLoft(def *LoftDefinition, sk SketchIndexer) (*LoftData, error) {
	sections := make([]LoftSectionData, len(def.Sections))
	for i, s := range def.Sections {
		idx := -1 // a face section (and a bare 3D point) carries no sketch
		if s.Sketch != nil {
			var ok bool
			if idx, ok = sk.IndexOf(s.Sketch); !ok {
				return nil, fmt.Errorf("loft section %d references a sketch that is not in the part", i)
			}
		}
		sd := LoftSectionData{Sketch: idx, Profile: s.ProfileIndex, FaceKey: s.FaceKey}
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
	var rails [][][]float64
	for _, r := range def.Rails {
		if r == nil {
			continue
		}
		if pts := r(); len(pts) >= 2 {
			rails = append(rails, encodePoints(pts))
		}
	}
	var centerline [][]float64
	if def.Centerline != nil {
		if pts := def.Centerline(); len(pts) >= 2 {
			centerline = encodePoints(pts)
		}
	}
	var area []LoftAreaStopData
	for _, st := range def.AreaGraph {
		area = append(area, LoftAreaStopData{T: st.T, Scale: st.Scale})
	}
	var maps [][][]float64
	for _, m := range def.MapCurves {
		if m == nil {
			continue
		}
		if pts := m(); len(pts) >= 2 {
			maps = append(maps, encodePoints(pts))
		}
	}
	return &LoftData{Sections: sections, Rails: rails, Centerline: centerline, AreaGraph: area, MapCurves: maps, Closed: def.Closed, Operation: op, First: encodeLoftEnd(first), Last: encodeLoftEnd(last)}, nil
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
		ls := LoftSection{ProfileIndex: s.Profile, FaceKey: s.FaceKey}
		if s.Sketch >= 0 { // a face section (and a bare 3D point) has no sketch (-1)
			skt, ok := sk.At(s.Sketch)
			if !ok {
				return nil, fmt.Errorf("loft section %d references sketch index %d, which does not exist", i, s.Sketch)
			}
			ls.Sketch = skt
		}
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
	first, last := decodeLoftEnd(d.First), decodeLoftEnd(d.Last)
	var g LoftGuideSet
	for _, rp := range d.Rails {
		pts := decodePoints(rp)
		g.Rails = append(g.Rails, func() []math.Point3 { return pts })
	}
	if len(d.Centerline) >= 2 {
		cl := decodePoints(d.Centerline)
		g.Centerline = func() []math.Point3 { return cl }
	}
	for _, st := range d.AreaGraph {
		g.AreaGraph = append(g.AreaGraph, LoftAreaStop{T: st.T, Scale: st.Scale})
	}
	for _, mp := range d.MapCurves {
		pts := decodePoints(mp)
		g.MapCurves = append(g.MapCurves, func() []math.Point3 { return pts })
	}
	return NewLoftFeatures(fs).AddGuided(sections, d.Closed, op, first, last, g), nil
}

// point3DChain wraps model points as sketch 3D points for a restored sweep path.
func point3DChain(pts []math.Point3) []*sketch.Point3D {
	out := make([]*sketch.Point3D, len(pts))
	for i, p := range pts {
		out[i] = sketch.NewPoint3D(p)
	}
	return out
}
