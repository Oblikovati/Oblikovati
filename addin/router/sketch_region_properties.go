// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/sketch"
)

// Region properties over the wire (M06-F08, #623): the full section property
// set of a closed profile, computed in model/sketch/region_properties.go.

// sketchRegionProperties serves wire.MethodSketchRegionProperties.
func sketchRegionProperties(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.RegionPropertiesArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, _, err := resolveSketch(s, raw) // RegionPropertiesArgs carries sketchIndex
	if err != nil {
		return nil, err
	}
	acc, err := regionAccuracy(in.Accuracy)
	if err != nil {
		return nil, err
	}
	profiles := sk.Profiles()
	if in.ProfileIndex < 0 || in.ProfileIndex >= profiles.Count() {
		return nil, fmt.Errorf("profile index %d out of range (sketch has %d profiles)",
			in.ProfileIndex, profiles.Count())
	}
	props, err := profiles.Item(in.ProfileIndex).RegionProperties(acc)
	if err != nil {
		return nil, err
	}
	return json.Marshal(regionPropertiesResult(props))
}

// sketch3DRegionProperties serves wire.MethodSketch3DRegionProperties.
func sketch3DRegionProperties(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.RegionPropertiesArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, _, err := resolveSketch3D(s, raw) // RegionPropertiesArgs carries sketchIndex
	if err != nil {
		return nil, err
	}
	acc, err := regionAccuracy(in.Accuracy)
	if err != nil {
		return nil, err
	}
	profiles := sk.Profiles3D()
	if in.ProfileIndex < 0 || in.ProfileIndex >= len(profiles) {
		return nil, fmt.Errorf("profile index %d out of range (3D sketch has %d profiles)",
			in.ProfileIndex, len(profiles))
	}
	props, err := profiles[in.ProfileIndex].RegionProperties(acc)
	if err != nil {
		return nil, err
	}
	return json.Marshal(regionPropertiesResult(props))
}

// regionAccuracy parses the requested accuracy; empty means the documented
// "high" default.
func regionAccuracy(spelling string) (types.Accuracy, error) {
	if spelling == "" {
		return types.AccuracyHigh, nil
	}
	acc, ok := types.ParseAccuracy(spelling)
	if !ok {
		return 0, fmt.Errorf("unknown accuracy %q (want low|medium|high|veryHigh)", spelling)
	}
	return acc, nil
}

// regionPropertiesResult renders the computed property set as its wire DTO.
func regionPropertiesResult(p *sketch.RegionProperties) wire.RegionPropertiesResult {
	cx, cy := p.Centroid()
	ixx, iyy, ixy := p.MomentsOfInertia()
	i1, i2 := p.PrincipalMoments()
	first, second := p.PrincipalAxes()
	return wire.RegionPropertiesResult{
		Area:             p.Area(),
		Perimeter:        p.Perimeter(),
		Centroid:         []float64{cx, cy},
		MomentsOfInertia: []float64{ixx, iyy, ixy},
		PrincipalMoments: []float64{i1, i2},
		RotationAngle:    p.RotationAngle(),
		PrincipalAxes: [][]float64{
			{float64(first.X), float64(first.Y)},
			{float64(second.X), float64(second.Y)},
		},
		Accuracy: p.Accuracy().String(),
	}
}
