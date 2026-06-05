// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// This file holds the YAML codecs for the placed/boolean solid features: holes,
// bosses (both placed on a face by reference key, like the dress-up family) and
// combine (a boolean between two running bodies by index). Keys re-bind to the
// regenerated topology after recompute; body indices address the running state.

// HoleData is a hole's recipe: the placement face (reference key), diameter/depth,
// the hole geometry type, and optional tap data.
type HoleData struct {
	Face            string  `yaml:"face"`
	Diameter        float64 `yaml:"diameter"`
	Depth           float64 `yaml:"depth"`
	ThroughAll      bool    `yaml:"throughAll,omitempty"`
	CounterDiameter float64 `yaml:"counterDiameter,omitempty"`
	CounterDepth    float64 `yaml:"counterDepth,omitempty"`
	CounterAngle    float64 `yaml:"counterAngle,omitempty"`
	PointAngle      float64 `yaml:"pointAngle,omitempty"`
	Type            string  `yaml:"type"`
	Tapped          bool    `yaml:"tapped,omitempty"`
	Designation     string  `yaml:"designation,omitempty"`
}

// BossData is a boss's recipe: a raised cylinder on a placement face.
type BossData struct {
	Face     string  `yaml:"face"`
	Diameter float64 `yaml:"diameter"`
	Height   float64 `yaml:"height"`
}

// CombineData booleans two running bodies (by index) under an operation.
type CombineData struct {
	Target    int    `yaml:"target"`
	Tool      int    `yaml:"tool"`
	Operation string `yaml:"operation"`
}

func serializeHole(def *HoleDefinition) (*HoleData, error) {
	kind, err := holeTypeName(def.Type)
	if err != nil {
		return nil, err
	}
	return &HoleData{
		Face:            encodeKey(def.PlacementFaceKey),
		Diameter:        evalFloat(def.Diameter),
		Depth:           evalFloat(def.Depth),
		ThroughAll:      def.ThroughAll,
		CounterDiameter: evalFloat(def.CounterDiameter),
		CounterDepth:    evalFloat(def.CounterDepth),
		CounterAngle:    evalFloat(def.CounterAngle),
		PointAngle:      evalFloat(def.PointAngle),
		Type:            kind,
		Tapped:          def.Tap.Tapped,
		Designation:     def.Tap.Designation,
	}, nil
}

func restoreHole(fs *PartFeatures, h *HoleData) (*PartFeature, error) {
	if h == nil {
		return nil, fmt.Errorf("hole feature is missing its payload")
	}
	key, err := decodeKey(h.Face)
	if err != nil {
		return nil, err
	}
	holeType, err := parseHoleType(h.Type)
	if err != nil {
		return nil, err
	}
	holes := NewHoleFeatures(fs)
	var pf *PartFeature
	switch {
	case h.Tapped:
		pf = holes.AddTapped(key, constFloat(h.Diameter), constFloat(h.Depth), h.Designation)
	case holeType == CounterboreHole:
		pf = holes.AddCounterbore(key, constFloat(h.Diameter), constFloat(h.Depth), constFloat(h.CounterDiameter), constFloat(h.CounterDepth))
	case holeType == CountersinkHole:
		pf = holes.AddCountersink(key, constFloat(h.Diameter), constFloat(h.Depth), constFloat(h.CounterDiameter), constFloat(h.CounterAngle))
	case h.ThroughAll:
		pf = holes.AddDrilledThrough(key, constFloat(h.Diameter))
	default:
		pf = holes.AddDrilled(key, constFloat(h.Diameter), constFloat(h.Depth))
	}
	def := pf.feature.(*HoleFeature).def
	def.Type = holeType
	def.ThroughAll = h.ThroughAll
	def.PointAngle = constFloat(h.PointAngle)
	return pf, nil
}

func restoreBoss(fs *PartFeatures, b *BossData) (*PartFeature, error) {
	if b == nil {
		return nil, fmt.Errorf("boss feature is missing its payload")
	}
	key, err := decodeKey(b.Face)
	if err != nil {
		return nil, err
	}
	return NewBossFeatures(fs).Add(key, constFloat(b.Diameter), constFloat(b.Height)), nil
}

func restoreCombine(fs *PartFeatures, c *CombineData) (*PartFeature, error) {
	if c == nil {
		return nil, fmt.Errorf("combine feature is missing its payload")
	}
	op, err := parseOperation(c.Operation)
	if err != nil {
		return nil, err
	}
	return NewModifyFeatures(fs).AddCombine(c.Target, c.Tool, op), nil
}

// holeTypeName / parseHoleType map the hole geometry type to/from a stable name.
func holeTypeName(t HoleType) (string, error) {
	switch t {
	case DrilledHole:
		return "drilled", nil
	case CounterboreHole:
		return "counterbore", nil
	case CountersinkHole:
		return "countersink", nil
	default:
		return "", fmt.Errorf("unknown hole type %d", t)
	}
}

func parseHoleType(name string) (HoleType, error) {
	switch name {
	case "drilled":
		return DrilledHole, nil
	case "counterbore":
		return CounterboreHole, nil
	case "countersink":
		return CountersinkHole, nil
	default:
		return 0, fmt.Errorf("unknown hole type %q (want drilled|counterbore|countersink)", name)
	}
}
