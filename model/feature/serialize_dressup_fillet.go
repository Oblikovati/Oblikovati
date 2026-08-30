// SPDX-License-Identifier: GPL-2.0-only

package feature

import "oblikovati.org/api/types"

// Dress-up serialization — the FILLET data (M48 #2237 split of serialize_dressup.go). The YAML shape of a
// fillet's edge sets (constant/variable radius with intermediate points), corner/cross-section modes and
// the face/rule/full-round variants, plus their serialize/restore. The shared edge/face reference and
// anchor encoding lives in serialize_dressup.go.

// FilletSetData is one serialized fillet edge set: constant (Radius) or variable
// (StartRadius/EndRadius over one edge, plus optional intermediate RadiusPoints, #695).
type FilletSetData struct {
	Edges        []string                `yaml:"edges"`
	Radius       float64                 `yaml:"radius,omitempty"`
	StartRadius  float64                 `yaml:"startRadius,omitempty"`
	EndRadius    float64                 `yaml:"endRadius,omitempty"`
	RadiusPoints []FilletRadiusPointData `yaml:"radiusPoints,omitempty"`
}

// FilletRadiusPointData is one serialized intermediate radius stop on a variable fillet edge (#695).
type FilletRadiusPointData struct {
	T      float64 `yaml:"t"`
	Radius float64 `yaml:"radius"`
}

// cornerTypeOrZero returns the fillet corner-treatment id, defaulting to 0 (miter) for an absent
// payload or an older recipe.
func (d *EdgeDressData) cornerTypeOrZero() int32 {
	if d == nil {
		return 0
	}
	return d.CornerType
}

// crossSectionOrArc returns the fillet cross-section, defaulting to the arc for an absent/older recipe.
func (d *EdgeDressData) crossSectionOrArc() FilletCrossSection {
	if d == nil {
		return FilletArc
	}
	c, _ := types.ParseFilletCrossSection(d.CrossSection)
	return c
}

// crossSectionWire is the cross-section's recipe spelling, empty for the arc default so omitempty
// keeps older recipes byte-identical.
func crossSectionWire(c FilletCrossSection) string {
	if c.IsArc() {
		return ""
	}
	return string(c)
}

// rhoOrZero returns the conic fullness rho (0 ⇒ default) for an absent/older recipe.
func (d *EdgeDressData) rhoOrZero() float64 {
	if d == nil {
		return 0
	}
	return d.Rho
}

// serializeFilletSets encodes a fillet's edge sets (nil for the legacy single-set form).
func serializeFilletSets(sets []FilletEdgeSet) []FilletSetData {
	out := make([]FilletSetData, len(sets))
	for i, s := range sets {
		out[i] = FilletSetData{Edges: encodeKeys(s.EdgeKeys)}
		if s.variable() {
			out[i].StartRadius, out[i].EndRadius = evalFloat(s.StartRadius), evalFloat(s.EndRadius)
			out[i].RadiusPoints = serializeRadiusPoints(s.RadiusPoints)
			continue
		}
		out[i].Radius = evalFloat(s.Radius)
	}
	return out
}

// restoreFilletSets decodes the edge-set form back into definition sets.
func restoreFilletSets(sets []FilletSetData) ([]FilletEdgeSet, error) {
	out := make([]FilletEdgeSet, len(sets))
	for i, s := range sets {
		keys, err := decodeKeys(s.Edges)
		if err != nil {
			return nil, err
		}
		out[i] = FilletEdgeSet{EdgeKeys: keys}
		if s.Radius > 0 {
			out[i].Radius = constFloat(s.Radius)
			continue
		}
		out[i].StartRadius, out[i].EndRadius = constFloat(s.StartRadius), constFloat(s.EndRadius)
		out[i].RadiusPoints = restoreRadiusPoints(s.RadiusPoints)
	}
	return out, nil
}

// serializeRadiusPoints / restoreRadiusPoints round-trip a variable fillet's intermediate
// radius stops (#695).
func serializeRadiusPoints(pts []FilletRadiusPoint) []FilletRadiusPointData {
	if len(pts) == 0 {
		return nil
	}
	out := make([]FilletRadiusPointData, len(pts))
	for i, p := range pts {
		out[i] = FilletRadiusPointData{T: p.T, Radius: evalFloat(p.Radius)}
	}
	return out
}

func restoreRadiusPoints(pts []FilletRadiusPointData) []FilletRadiusPoint {
	if len(pts) == 0 {
		return nil
	}
	out := make([]FilletRadiusPoint, len(pts))
	for i, p := range pts {
		out[i] = FilletRadiusPoint{T: p.T, Radius: constFloat(p.Radius)}
	}
	return out
}

// FaceFilletData persists a face fillet (#694): the two face-set reference keys and the radius.
type FaceFilletData struct {
	FacesA []string `yaml:"facesA"`
	FacesB []string `yaml:"facesB"`
	Value  float64  `yaml:"value"`
	// Width sizes the blend by its CHORD instead of by Value, the rolling ball's radius (#1887).
	// Persisted separately so a reopened part still resolves the width against the angle its faces
	// meet at, rather than freezing whatever radius that came to on the day it was authored.
	Width float64 `yaml:"width,omitempty"`
}

// RuleFilletData persists a rule fillet (#486): the dihedral rule (wire spelling) and the radius.
type RuleFilletData struct {
	Rule  string  `yaml:"rule"`
	Value float64 `yaml:"value"`
}

// FullRoundData persists a full-round fillet (#694): the three face-set reference keys.
type FullRoundData struct {
	Side1  []string `yaml:"side1"`
	Center []string `yaml:"center"`
	Side2  []string `yaml:"side2"`
}
