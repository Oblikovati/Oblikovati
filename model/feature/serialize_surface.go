// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/model/sketch"
)

// This file holds the YAML codecs for the sketch-based surface features — boundary
// patch (fills closed sketch loops) and ruled surface (sweeps a profile's edges by a
// distance). Like extrude, they reference their input sketch by index via the
// SketchIndexer.

// BoundaryPatchData is a boundary patch's recipe: the closed sketch loops it fills.
type BoundaryPatchData struct {
	Loops []PatchLoopData `yaml:"loops"`
}

// PatchLoopData is one boundary loop: a sketch profile and its continuity condition.
type PatchLoopData struct {
	Sketch    int    `yaml:"sketch"`
	Profile   int    `yaml:"profile"`
	Condition string `yaml:"condition"`
}

// RuledSurfaceData is a ruled surface's recipe: a profile swept by straight rulings. Direction is
// the sweep ruling vector, DraftAngle the outward flare, Flip the ruling-side reversal (#1868).
type RuledSurfaceData struct {
	Sketch     int       `yaml:"sketch"`
	Profile    int       `yaml:"profile"`
	Type       string    `yaml:"type"`
	Distance   float64   `yaml:"distance"`
	Direction  []float64 `yaml:"direction,omitempty"`
	DraftAngle float64   `yaml:"draftAngle,omitempty"`
	Flip       bool      `yaml:"flip,omitempty"`
}

func serializeBoundaryPatch(def *BoundaryPatchDefinition, sk SketchIndexer) (*BoundaryPatchData, error) {
	if def.Loops == nil || def.Loops.Count() == 0 {
		return nil, fmt.Errorf("boundary patch has no loops")
	}
	out := &BoundaryPatchData{Loops: make([]PatchLoopData, def.Loops.Count())}
	for i := 0; i < def.Loops.Count(); i++ {
		loop := def.Loops.Item(i)
		idx, ok := sk.IndexOf(loop.Sketch)
		if !ok {
			return nil, fmt.Errorf("boundary patch loop %d references a sketch not in the part", i)
		}
		cond, err := patchConditionName(loop.Condition)
		if err != nil {
			return nil, err
		}
		out.Loops[i] = PatchLoopData{Sketch: idx, Profile: loop.ProfileIndex, Condition: cond}
	}
	return out, nil
}

func restoreBoundaryPatch(fs *PartFeatures, d *BoundaryPatchData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil || len(d.Loops) == 0 {
		return nil, fmt.Errorf("boundary-patch feature is missing its loops")
	}
	first, err := resolvePatchLoop(d.Loops[0], sk)
	if err != nil {
		return nil, err
	}
	pf := NewBoundaryPatchFeatures(fs).Add(first.skt, first.profile, first.cond)
	// Additional loops (outer + holes) append to the same patch definition.
	for i := 1; i < len(d.Loops); i++ {
		more, err := resolvePatchLoop(d.Loops[i], sk)
		if err != nil {
			return nil, err
		}
		pf.feature.(*BoundaryPatchFeature).def.Loops.Add(more.skt, more.profile, more.cond)
	}
	return pf, nil
}

// patchLoopInputs is the resolved (sketch, profile, condition) of one loop.
type patchLoopInputs struct {
	skt     *sketch.Sketch
	profile int
	cond    PatchCondition
}

// resolvePatchLoop resolves a serialized loop's sketch index and continuity name.
func resolvePatchLoop(loop PatchLoopData, sk SketchIndexer) (patchLoopInputs, error) {
	skt, ok := sk.At(loop.Sketch)
	if !ok {
		return patchLoopInputs{}, fmt.Errorf("boundary patch references sketch index %d, which does not exist", loop.Sketch)
	}
	cond, err := parsePatchCondition(loop.Condition)
	if err != nil {
		return patchLoopInputs{}, err
	}
	return patchLoopInputs{skt: skt, profile: loop.Profile, cond: cond}, nil
}

func serializeRuledSurface(def *RuledSurfaceDefinition, sk SketchIndexer) (*RuledSurfaceData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("ruled surface references a sketch not in the part")
	}
	kind, err := ruledTypeName(def.Type)
	if err != nil {
		return nil, err
	}
	d := &RuledSurfaceData{Sketch: idx, Profile: def.ProfileIndex, Type: kind, Distance: evalFloat(def.Distance), Flip: def.Flip}
	if def.Type == RuledSweep {
		d.Direction = encodeVec3(def.Direction)
	}
	if a := evalFloat(def.DraftAngle); a != 0 {
		d.DraftAngle = a
	}
	return d, nil
}

func restoreRuledSurface(fs *PartFeatures, d *RuledSurfaceData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("ruled-surface feature is missing its payload")
	}
	skt, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("ruled surface references sketch index %d, which does not exist", d.Sketch)
	}
	kind, err := parseRuledType(d.Type)
	if err != nil {
		return nil, err
	}
	def := &RuledSurfaceDefinition{
		Sketch: skt, ProfileIndex: d.Profile, Type: kind, Distance: constFloat(d.Distance),
		DraftAngle: constFloat(d.DraftAngle), Flip: d.Flip,
	}
	if len(d.Direction) == 3 {
		def.Direction = decodeVec3(d.Direction)
	}
	return NewRuledSurfaceFeatures(fs).AddRuled(def), nil
}

// patchConditionName / parsePatchCondition map the continuity condition to/from a name.
func patchConditionName(c PatchCondition) (string, error) {
	switch c {
	case PatchFree:
		return "free", nil
	case PatchTangent:
		return "tangent", nil
	case PatchCurvature:
		return "curvature", nil
	default:
		return "", fmt.Errorf("unknown patch condition %d", c)
	}
}

func parsePatchCondition(name string) (PatchCondition, error) {
	switch name {
	case "free":
		return PatchFree, nil
	case "tangent":
		return PatchTangent, nil
	case "curvature":
		return PatchCurvature, nil
	default:
		return 0, fmt.Errorf("unknown patch condition %q (want free|tangent|curvature)", name)
	}
}

// ruledTypeName / parseRuledType map the ruled-surface direction type to/from a name. "perpendicular"
// is the pre-#1868 spelling of the sweep type and still parses (back-compat) but serializes as "sweep".
func ruledTypeName(t RuledSurfaceType) (string, error) {
	switch t {
	case RuledNormal:
		return "normal", nil
	case RuledTangent:
		return "tangent", nil
	case RuledSweep:
		return "sweep", nil
	default:
		return "", fmt.Errorf("unknown ruled surface type %d", t)
	}
}

func parseRuledType(name string) (RuledSurfaceType, error) {
	switch name {
	case "normal":
		return RuledNormal, nil
	case "tangent":
		return RuledTangent, nil
	case "sweep", "perpendicular":
		return RuledSweep, nil
	default:
		return 0, fmt.Errorf("unknown ruled surface type %q (want normal|tangent|sweep)", name)
	}
}
