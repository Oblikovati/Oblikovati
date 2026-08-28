// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// Coil-feature serialization (M48 #2238 split of serialize_work.go). The YAML shape and serialize/restore
// of a coil feature — profile, axis, pitch/height/revolutions rows, handedness and the start/end
// conditions. Shares the work-geometry reference helpers in serialize_work.go.

// CoilData is a coil's recipe: the sketch profile, the helix axis (a WorkRef), the
// pitch (per revolution), the number of revolutions, the taper, and the operation.
type CoilData struct {
	Sketch      int     `yaml:"sketch"`
	Profile     int     `yaml:"profile"`
	Axis        string  `yaml:"axis"`
	Pitch       float64 `yaml:"pitch,omitempty"`
	Revolutions float64 `yaml:"revolutions,omitempty"`
	Height      float64 `yaml:"height,omitempty"` // two-of-three shape spec (#316)
	Taper       float64 `yaml:"taper,omitempty"`
	Operation   string  `yaml:"operation"`
	// Variable-pitch rail + end conditions (M06-F09, #624).
	PitchRows []CoilPitchRowData `yaml:"pitchRows,omitempty"`
	StartEnd  *CoilEndData       `yaml:"startEnd,omitempty"`
	EndEnd    *CoilEndData       `yaml:"endEnd,omitempty"`
	// Handedness and the flat-spiral flavour (#1883). LeftHanded is persisted as a flag rather
	// than the enum so an existing document (no key) reads back as right-handed, the old default.
	LeftHanded bool `yaml:"leftHanded,omitempty"`
	Spiral     bool `yaml:"spiral,omitempty"`
}

// CoilPitchRowData is one persisted pitch station.
type CoilPitchRowData struct {
	Pitch      float64 `yaml:"pitch"`
	Revolution float64 `yaml:"revolution"`
}

// CoilEndData is one persisted flat-end condition (radians).
type CoilEndData struct {
	TransitionAngle float64 `yaml:"transitionAngle,omitempty"`
	FlatAngle       float64 `yaml:"flatAngle,omitempty"`
}

func serializeCoil(def *CoilDefinition, sk SketchIndexer) (*CoilData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("coil references a sketch that is not in the part")
	}
	if def.Axis == nil {
		return nil, fmt.Errorf("coil has no axis")
	}
	op, err := operationName(def.Operation)
	if err != nil {
		return nil, err
	}
	d := &CoilData{
		Sketch: idx, Profile: def.ProfileIndex, Axis: string(def.Axis.Key()),
		Pitch: evalFloat(def.Pitch), Revolutions: evalFloat(def.Revolutions),
		Height: evalFloat(def.Height), Taper: def.Taper, Operation: op,
		LeftHanded: def.Handedness == LeftHandedCoil, Spiral: def.Spiral,
	}
	for _, r := range def.PitchRows {
		d.PitchRows = append(d.PitchRows, CoilPitchRowData(r))
	}
	d.StartEnd = coilEndData(def.StartEnd)
	d.EndEnd = coilEndData(def.EndEnd)
	return d, nil
}

// coilEndData persists a flat end (nil for natural).
func coilEndData(c CoilEndCondition) *CoilEndData {
	if !c.Flat {
		return nil
	}
	return &CoilEndData{TransitionAngle: c.TransitionAngle, FlatAngle: c.FlatAngle}
}

func restoreCoil(fs *PartFeatures, d *CoilData, sk SketchIndexer, work *WorkGeometry) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("coil feature is missing its payload")
	}
	skt, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("coil references sketch index %d, which does not exist", d.Sketch)
	}
	if work == nil {
		return nil, fmt.Errorf("coil needs the part's work geometry to resolve its axis")
	}
	axis, err := work.axis(WorkRef(d.Axis))
	if err != nil {
		return nil, fmt.Errorf("coil axis: %w", err)
	}
	op, err := parseOperation(d.Operation)
	if err != nil {
		return nil, err
	}
	def := &CoilDefinition{
		Sketch: skt, ProfileIndex: d.Profile, Axis: axis,
		Pitch: constFloat(d.Pitch), Revolutions: constFloat(d.Revolutions),
		Height: constFloat(d.Height), Taper: d.Taper, Operation: op,
		Handedness: coilHandednessFromData(d.LeftHanded), Spiral: d.Spiral,
	}
	pf := NewCoilFeatures(fs).AddDefinition(def)
	for _, r := range d.PitchRows {
		def.PitchRows = append(def.PitchRows, CoilPitchRow(r))
	}
	def.StartEnd = coilEndFromData(d.StartEnd)
	def.EndEnd = coilEndFromData(d.EndEnd)
	return pf, nil
}

// coilHandednessFromData rebuilds the winding sense from the persisted flag (#1883).
func coilHandednessFromData(left bool) CoilHandedness {
	if left {
		return LeftHandedCoil
	}
	return RightHandedCoil
}

// coilEndFromData rebuilds a persisted flat end (nil stays natural).
func coilEndFromData(d *CoilEndData) CoilEndCondition {
	if d == nil {
		return CoilEndCondition{}
	}
	return CoilEndCondition{Flat: true, TransitionAngle: d.TransitionAngle, FlatAngle: d.FlatAngle}
}
