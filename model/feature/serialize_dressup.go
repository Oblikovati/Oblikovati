// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"encoding/base64"
	"fmt"
	"sort"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

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
	// Fillet-only cross-section (M36-F08): "g2" or "conic"; absent/"arc" ⇒ the circular-arc
	// rolling-ball blend. Rho is the conic's fullness (0<ρ<1, 0.5 = parabola).
	CrossSection string  `yaml:"crossSection,omitempty"`
	Rho          float64 `yaml:"rho,omitempty"`
	// GeomEdges are edges selected by a serialized GEOMETRIC descriptor (ADR-0040), the path
	// an external author (the NX exporter) uses because it cannot mint Oblikovati lineage
	// keys. Empty for an Oblikovati-authored dress-up (which uses Edges).
	GeomEdges []GeomEdgeRefData `yaml:"geomEdges,omitempty"`
	// EdgeAnchors carry each picked edge's mint-time midpoint (keyed by its reference key) for
	// the geometric recovery tier (ADR-0043 P6b). Omitted for an older recipe, which then
	// recovers a lost reference by exact/ancestral binding only.
	EdgeAnchors []EdgeAnchorData `yaml:"edgeAnchors,omitempty"`
	// Chamfer-only (#1888): the face Value is measured on for the asymmetric modes, and the span of
	// each edge the bevel covers. Absent ⇒ the edge's own face order and the whole edge.
	ReferenceFace string  `yaml:"referenceFace,omitempty"`
	PartialStart  float64 `yaml:"partialStart,omitempty"`
	PartialLength float64 `yaml:"partialLength,omitempty"`
}

// EdgeAnchorData is the serialized mint-time anchor of one picked edge: its reference key
// (base64) and midpoint, used to disambiguate surviving siblings when the exact key is lost.
type EdgeAnchorData struct {
	Key      string    `yaml:"key"`
	Midpoint []float64 `yaml:"midpoint,flow"`
}

// FaceAnchorData is the serialized mint-time anchor of one picked face: its reference key (base64)
// and centroid, used to disambiguate surviving siblings when the exact key is lost (#1579).
type FaceAnchorData struct {
	Key      string    `yaml:"key"`
	Centroid []float64 `yaml:"centroid,flow"`
}

// GeomEdgeRefData is the serialized form of a geometric edge descriptor: the edge's
// midpoint and (sign-agnostic) direction in model space. It binds to a running-body edge
// at recompute via [topo.Body.FindEdgeByGeometry] (ADR-0040).
type GeomEdgeRefData struct {
	Midpoint  []float64 `yaml:"midpoint,flow"`
	Direction []float64 `yaml:"direction,omitempty,flow"`
}

// FaceDressData is a face-based dress-up (shell thickness / draft angle): the picked
// faces as reference keys plus the scalar value, and (draft only) the pull direction.
type FaceDressData struct {
	Faces         []string  `yaml:"faces"`
	Value         float64   `yaml:"value"`
	Pull          []float64 `yaml:"pull,omitempty"`          // draft pull direction (dx,dy,dz); absent ⇒ +Z
	NeutralOrigin []float64 `yaml:"neutralOrigin,omitempty"` // draft neutral (parting) plane origin (x,y,z); absent ⇒ implicit hinge
	NeutralNormal []float64 `yaml:"neutralNormal,omitempty"` // draft neutral plane normal (nx,ny,nz)
	// GeomFaces are faces selected by a serialized GEOMETRIC descriptor (ADR-0040), the
	// path the NX exporter uses since it cannot mint Oblikovati lineage keys. Empty for an
	// Oblikovati-authored dress-up (which uses Faces).
	GeomFaces []GeomFaceRefData `yaml:"geomFaces,omitempty"`
	// ShellDirection is the wall side for a shell ("outside"/"both"; absent ⇒ inside). Unused by
	// the other dress-ups. #1864.
	ShellDirection string `yaml:"shellDirection,omitempty"`
	// FaceThicknesses are a shell's per-face wall overrides, resolved at save time (#1864).
	// Absent ⇒ every retained face carries the uniform Value.
	FaceThicknesses []FaceThicknessData `yaml:"faceThicknesses,omitempty"`
}

// GeomFaceRefData is the serialized form of a geometric face descriptor: the face's
// centroid and outward normal in model space. It binds to a running-body face at recompute
// via [topo.Body.FindFaceByGeometry] (ADR-0040).
type GeomFaceRefData struct {
	Centroid []float64 `yaml:"centroid,flow"`
	Normal   []float64 `yaml:"normal,omitempty,flow"`
}

// SnapFitData persists a cantilever snap-fit (#486): the beam and catch dimensions.
type SnapFitData struct {
	Length      float64 `yaml:"length"`
	Width       float64 `yaml:"width"`
	Thickness   float64 `yaml:"thickness"`
	CatchLength float64 `yaml:"catchLength"`
	CatchHeight float64 `yaml:"catchHeight"`
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
	// Offset and Length are the thread's axial window on the face (cm), resolved at save time
	// (Inventor ThreadOffset / ThreadDepth). Absent/0 length ⇒ the thread runs the full face.
	Offset float64 `yaml:"offset,omitempty"`
	Length float64 `yaml:"length,omitempty"`
	// LeftHanded reverses the thread sense (#1892). A flag, not a "handedness" name, so a
	// document written before the option existed reads back as the right-hand thread it was.
	LeftHanded  bool             `yaml:"leftHanded,omitempty"`
	FaceAnchors []FaceAnchorData `yaml:"faceAnchors,omitempty"`
}

// dressInputs is the decoded (keys, value) pair shared by edge/face dress-ups, plus any
// geometric edge descriptors (externally-authored selections, ADR-0040) and the mint-time
// edge anchors for geometric recovery (ADR-0043 P6b).
type dressInputs struct {
	keys      [][]byte
	value     float64
	geom      []topo.GeometricEdgeRef
	geomFaces []topo.GeometricFaceRef
	anchors   map[string]math.Point3
}

func requireEdgeDress(d *EdgeDressData, kind string) (dressInputs, error) {
	if d == nil {
		return dressInputs{}, fmt.Errorf("%s feature is missing its payload", kind)
	}
	keys, err := decodeKeys(d.Edges)
	if err != nil {
		return dressInputs{}, err
	}
	anchors, err := decodeEdgeAnchors(d.EdgeAnchors)
	if err != nil {
		return dressInputs{}, err
	}
	return dressInputs{keys: keys, value: d.Value, geom: decodeGeomEdges(d.GeomEdges), anchors: anchors}, nil
}

// encodeEdgeAnchors / decodeEdgeAnchors round-trip the mint-time edge-anchor map (ADR-0043
// P6b): keys are base64-encoded reference keys (opaque lineage bytes, like Edges), values are
// midpoints. Empty for a dress-up authored before P6b or with no captured anchors.
func encodeEdgeAnchors(anchors map[string]math.Point3) []EdgeAnchorData {
	if len(anchors) == 0 {
		return nil
	}
	out := make([]EdgeAnchorData, 0, len(anchors))
	for k, p := range anchors {
		out = append(out, EdgeAnchorData{Key: encodeKey([]byte(k)), Midpoint: encodePoint3(p)})
	}
	// Deterministic order so the serialized document is stable across runs (maps iterate randomly).
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func decodeEdgeAnchors(data []EdgeAnchorData) (map[string]math.Point3, error) {
	if len(data) == 0 {
		return nil, nil
	}
	out := make(map[string]math.Point3, len(data))
	for _, d := range data {
		k, err := decodeKey(d.Key)
		if err != nil {
			return nil, err
		}
		out[string(k)] = decodePoint3(d.Midpoint)
	}
	return out, nil
}

// encodeFaceAnchors / decodeFaceAnchors round-trip the mint-time face-anchor map (#1579), the
// face counterpart of encodeEdgeAnchors: keys are base64 reference keys, values are centroids.
// Empty for a feature authored with no captured anchors or restored from a pre-#1579 recipe.
func encodeFaceAnchors(anchors map[string]math.Point3) []FaceAnchorData {
	if len(anchors) == 0 {
		return nil
	}
	out := make([]FaceAnchorData, 0, len(anchors))
	for k, p := range anchors {
		out = append(out, FaceAnchorData{Key: encodeKey([]byte(k)), Centroid: encodePoint3(p)})
	}
	// Deterministic order so the serialized document is stable across runs (maps iterate randomly).
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func decodeFaceAnchors(data []FaceAnchorData) (map[string]math.Point3, error) {
	if len(data) == 0 {
		return nil, nil
	}
	out := make(map[string]math.Point3, len(data))
	for _, d := range data {
		k, err := decodeKey(d.Key)
		if err != nil {
			return nil, err
		}
		out[string(k)] = decodePoint3(d.Centroid)
	}
	return out, nil
}

// encodeGeomEdges / decodeGeomEdges convert geometric edge descriptors to and from their
// serialized form. A descriptor with no direction (degenerate) still round-trips by
// midpoint alone.
func encodeGeomEdges(geom []topo.GeometricEdgeRef) []GeomEdgeRefData {
	if len(geom) == 0 {
		return nil
	}
	out := make([]GeomEdgeRefData, len(geom))
	for i, g := range geom {
		out[i] = GeomEdgeRefData{Midpoint: encodePoint3(g.Midpoint), Direction: encodeVec3(g.Direction)}
	}
	return out
}

func decodeGeomEdges(data []GeomEdgeRefData) []topo.GeometricEdgeRef {
	if len(data) == 0 {
		return nil
	}
	out := make([]topo.GeometricEdgeRef, len(data))
	for i, d := range data {
		out[i] = topo.GeometricEdgeRef{Midpoint: decodePoint3(d.Midpoint), Direction: decodeVec3(d.Direction)}
	}
	return out
}

func requireFaceDress(d *FaceDressData, kind string) (dressInputs, error) {
	if d == nil {
		return dressInputs{}, fmt.Errorf("%s feature is missing its payload", kind)
	}
	keys, err := decodeKeys(d.Faces)
	if err != nil {
		return dressInputs{}, err
	}
	return dressInputs{keys: keys, value: d.Value, geomFaces: decodeGeomFaces(d.GeomFaces)}, nil
}

// encodeGeomFaces / decodeGeomFaces convert geometric face descriptors to and from their
// serialized form (centroid + outward normal).
func encodeGeomFaces(geom []topo.GeometricFaceRef) []GeomFaceRefData {
	if len(geom) == 0 {
		return nil
	}
	out := make([]GeomFaceRefData, len(geom))
	for i, g := range geom {
		out[i] = GeomFaceRefData{Centroid: encodePoint3(g.Centroid), Normal: encodeVec3(g.Normal)}
	}
	return out
}

func decodeGeomFaces(data []GeomFaceRefData) []topo.GeometricFaceRef {
	if len(data) == 0 {
		return nil
	}
	out := make([]topo.GeometricFaceRef, len(data))
	for i, d := range data {
		out[i] = topo.GeometricFaceRef{Centroid: decodePoint3(d.Centroid), Normal: decodeVec3(d.Normal)}
	}
	return out
}

// encodeGeomFacePtr / decodeGeomFacePtr handle a single optional geometric face descriptor
// (a hole's placement face).
func encodeGeomFacePtr(g *topo.GeometricFaceRef) *GeomFaceRefData {
	if g == nil {
		return nil
	}
	return &GeomFaceRefData{Centroid: encodePoint3(g.Centroid), Normal: encodeVec3(g.Normal)}
}

func decodeGeomFacePtr(d *GeomFaceRefData) *topo.GeometricFaceRef {
	if d == nil {
		return nil
	}
	return &topo.GeometricFaceRef{Centroid: decodePoint3(d.Centroid), Normal: decodeVec3(d.Normal)}
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
