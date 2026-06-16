// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"encoding/base64"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// draftPull reads a serialized pull direction [dx,dy,dz], defaulting to +Z when absent
// (older recipes / the common Z-up mould pull).
func draftPull(p []float64) math.Vector3 {
	if len(p) != 3 {
		return math.V3(0, 0, 1)
	}
	return math.V3(p[0], p[1], p[2])
}

// This file holds the YAML codecs for the dress-up family (fillet/chamfer/shell/draft/
// thread) and the reference-key + scalar helpers shared across feature codecs. Edge and
// face reference keys are opaque lineage bytes, so they are base64-encoded (ADR-0020)
// and re-bind to the regenerated topology after recompute.

// EdgeDressData is an edge-based dress-up (fillet radius / chamfer distance): the
// picked edges as reference keys plus the scalar value. FlatCorners is chamfer-only and a
// pointer so an absent value (older recipes, or a fillet) is distinguishable from an
// explicit false; absent restores as the flat-corner default (see chamferFlatCornersOr).
// Sets is fillet-only: the edge-set form (#323) — when present it carries the whole recipe
// and Edges/Value stay empty.
type EdgeDressData struct {
	Edges       []string        `yaml:"edges,omitempty"`
	Value       float64         `yaml:"value,omitempty"`
	FlatCorners *bool           `yaml:"flatCorners,omitempty"`
	Sets        []FilletSetData `yaml:"sets,omitempty"`
	// Chamfer-only mode (M20-F03): the setback mode and its second input. ChamferType 0 (or
	// absent, in older recipes) ⇒ the equal-distance default; Value2 is the second distance
	// (twoDistances); Angle is the chamfer-face angle in radians (distanceAndAngle).
	ChamferType int32   `yaml:"chamferType,omitempty"`
	Value2      float64 `yaml:"value2,omitempty"`
	Angle       float64 `yaml:"angle,omitempty"`
	// Fillet-only: the shared-corner treatment. FilletCornerType 0 (or absent) ⇒ the miter default.
	CornerType int32 `yaml:"cornerType,omitempty"`
}

// FilletSetData is one serialized fillet edge set: constant (Radius) or variable
// (StartRadius/EndRadius over one edge).
type FilletSetData struct {
	Edges       []string `yaml:"edges"`
	Radius      float64  `yaml:"radius,omitempty"`
	StartRadius float64  `yaml:"startRadius,omitempty"`
	EndRadius   float64  `yaml:"endRadius,omitempty"`
}

// cornerTypeOrZero returns the fillet corner-treatment id, defaulting to 0 (miter) for an absent
// payload or an older recipe.
func (d *EdgeDressData) cornerTypeOrZero() int32 {
	if d == nil {
		return 0
	}
	return d.CornerType
}

// serializeFilletSets encodes a fillet's edge sets (nil for the legacy single-set form).
func serializeFilletSets(sets []FilletEdgeSet) []FilletSetData {
	out := make([]FilletSetData, len(sets))
	for i, s := range sets {
		out[i] = FilletSetData{Edges: encodeKeys(s.EdgeKeys)}
		if s.variable() {
			out[i].StartRadius, out[i].EndRadius = evalFloat(s.StartRadius), evalFloat(s.EndRadius)
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
	}
	return out, nil
}

// FaceDressData is a face-based dress-up (shell thickness / draft angle): the picked
// faces as reference keys plus the scalar value, and (draft only) the pull direction.
type FaceDressData struct {
	Faces []string  `yaml:"faces"`
	Value float64   `yaml:"value"`
	Pull  []float64 `yaml:"pull,omitempty"` // draft pull direction (dx,dy,dz); absent ⇒ +Z
}

// ThreadData tags a single cylindrical face (reference key) with a thread designation, plus
// the #325 parity fields: the tolerance class, the tapered (pipe) flag, and which thread
// diameter the modeled face represents (wire spelling; absent = major).
type ThreadData struct {
	Face          string `yaml:"face"`
	Designation   string `yaml:"designation"`
	Cut           bool   `yaml:"cut,omitempty"`
	Class         string `yaml:"class,omitempty"`
	Tapered       bool   `yaml:"tapered,omitempty"`
	ModelDiameter string `yaml:"modelDiameter,omitempty"`
}

// dressInputs is the decoded (keys, value) pair shared by edge/face dress-ups.
type dressInputs struct {
	keys  [][]byte
	value float64
}

func requireEdgeDress(d *EdgeDressData, kind string) (dressInputs, error) {
	if d == nil {
		return dressInputs{}, fmt.Errorf("%s feature is missing its payload", kind)
	}
	keys, err := decodeKeys(d.Edges)
	if err != nil {
		return dressInputs{}, err
	}
	return dressInputs{keys: keys, value: d.Value}, nil
}

func requireFaceDress(d *FaceDressData, kind string) (dressInputs, error) {
	if d == nil {
		return dressInputs{}, fmt.Errorf("%s feature is missing its payload", kind)
	}
	keys, err := decodeKeys(d.Faces)
	if err != nil {
		return dressInputs{}, err
	}
	return dressInputs{keys: keys, value: d.Value}, nil
}

// encodeKeys / decodeKeys base64-encode reference keys (opaque lineage bytes) so they
// stay valid text in the YAML document (ADR-0020); they re-bind after recompute.
func encodeKeys(keys [][]byte) []string {
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = encodeKey(k)
	}
	return out
}

func decodeKeys(encoded []string) ([][]byte, error) {
	out := make([][]byte, len(encoded))
	for i, e := range encoded {
		k, err := decodeKey(e)
		if err != nil {
			return nil, err
		}
		out[i] = k
	}
	return out, nil
}

func encodeKey(k []byte) string { return base64.StdEncoding.EncodeToString(k) }

func decodeKey(s string) ([]byte, error) {
	k, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("reference key %q is not valid base64: %w", s, err)
	}
	return k, nil
}

// evalFloat reads a feature's scalar (a closure, typically a parameter); a nil closure
// reads as 0. constFloat is its inverse for restore — a fixed value (parametric values
// arrive with the dimension-driven API, like extrude distance).
func evalFloat(fn func() float64) float64 {
	if fn == nil {
		return 0
	}
	return fn()
}

func constFloat(v float64) func() float64 { return func() float64 { return v } }

// chamferFlatCornersOr reads a serialized chamfer's flat-corner flag, defaulting an absent
// value (older recipes) to the flat-corner default so reopening matches a fresh chamfer.
func chamferFlatCornersOr(p *bool) bool {
	if p == nil {
		return true
	}
	return *p
}

// threadModelDiameterName encodes a thread's model-diameter choice as its wire spelling
// (empty for the zero value, so older recipes stay byte-identical).
func threadModelDiameterName(md types.ModelDiameterFromThread) string {
	if md == 0 {
		return ""
	}
	return md.String()
}

// threadModelDiameterOf decodes the wire spelling back (absent = zero value, meaning major).
func threadModelDiameterOf(s string) (types.ModelDiameterFromThread, error) {
	if s == "" {
		return 0, nil
	}
	md, ok := types.ParseModelDiameterFromThread(s)
	if !ok {
		return 0, fmt.Errorf("thread: unknown modelDiameter %q (want major/minor/pitch/tapDrill)", s)
	}
	return md, nil
}
