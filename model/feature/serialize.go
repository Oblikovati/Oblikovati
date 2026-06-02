// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"encoding/base64"
	"fmt"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// This file serializes the ordered feature program into the git-friendly YAML recipe
// (ADR-0020) and rebuilds it on open, then the part recompute regenerates geometry.
// Features reference sketches by index (resolved through a [SketchIndexer]); features
// that reference realized topology (dress-up edges/faces by reference key) get their
// codecs — and the key-context persistence — as those kinds are added. Any feature
// kind without a codec makes Save error rather than drop the feature silently.

// FeatureData is the serializable form of one history feature. Exactly one payload
// pointer is set, matching Kind.
type FeatureData struct {
	Kind       string         `yaml:"kind"`
	Name       string         `yaml:"name,omitempty"`
	Suppressed bool           `yaml:"suppressed,omitempty"`
	Extrude    *ExtrudeData   `yaml:"extrude,omitempty"`
	Fillet     *EdgeDressData `yaml:"fillet,omitempty"`
	Chamfer    *EdgeDressData `yaml:"chamfer,omitempty"`
	Shell      *FaceDressData `yaml:"shell,omitempty"`
	Draft      *FaceDressData `yaml:"draft,omitempty"`
	Thread     *ThreadData    `yaml:"thread,omitempty"`
	Hole       *HoleData      `yaml:"hole,omitempty"`
	Boss       *BossData      `yaml:"boss,omitempty"`
	Combine    *CombineData   `yaml:"combine,omitempty"`

	RectPattern   *RectPatternData         `yaml:"rectangularPattern,omitempty"`
	CircPattern   *CircPatternData         `yaml:"circularPattern,omitempty"`
	SketchPattern *SketchDrivenPatternData `yaml:"sketchDrivenPattern,omitempty"`
	Mirror        *MirrorData              `yaml:"mirror,omitempty"`
}

// EdgeDressData is an edge-based dress-up (fillet radius / chamfer distance): the
// picked edges as reference keys plus the scalar value. The keys are base64 because
// they are opaque lineage-derived bytes — the one non-text field (ADR-0020); they
// re-bind to the regenerated edges after recompute (kernel topo FindEdgeByKey).
type EdgeDressData struct {
	Edges []string `yaml:"edges"`
	Value float64  `yaml:"value"`
}

// FaceDressData is a face-based dress-up (shell thickness / draft angle): the picked
// faces as reference keys plus the scalar value.
type FaceDressData struct {
	Faces []string `yaml:"faces"`
	Value float64  `yaml:"value"`
}

// ThreadData tags a single cylindrical face (reference key) with a thread designation.
type ThreadData struct {
	Face        string `yaml:"face"`
	Designation string `yaml:"designation"`
}

// ExtrudeData is an extrude's recipe: which sketch profile, the boolean operation, and
// the distance extent. The distance is the evaluated growth (a fixed value on reopen;
// parametric distance expressions arrive with the dimension-driven extent API).
type ExtrudeData struct {
	Sketch    int     `yaml:"sketch"`
	Profile   int     `yaml:"profile"`
	Operation string  `yaml:"operation"`
	Distance  float64 `yaml:"distance"`
	Taper     float64 `yaml:"taper,omitempty"`
}

// SketchIndexer maps between a sketch pointer and its index in the part, so a feature
// can record which sketch it consumes (marshal) and re-bind it (restore).
type SketchIndexer interface {
	IndexOf(*sketch.Sketch) (int, bool)
	At(int) (*sketch.Sketch, bool)
}

// MarshalRecipe projects the feature program into its serializable form, in history
// order, erroring on any feature kind without a codec (no silent loss).
func (fs *PartFeatures) MarshalRecipe(sk SketchIndexer) ([]FeatureData, error) {
	idx := fs.indexByID()
	out := make([]FeatureData, 0, fs.Count())
	for i := 0; i < fs.Count(); i++ {
		pf := fs.Item(i)
		fd, err := serializeFeature(pf, sk, idx)
		if err != nil {
			return nil, fmt.Errorf("feature %d (%s): %w", i, pf.Kind(), err)
		}
		out = append(out, fd)
	}
	return out, nil
}

// indexByID maps each feature's stable id to its position, so a pattern can record
// which earlier features it replicates as program indices (ids are not persisted).
func (fs *PartFeatures) indexByID() map[ID]int {
	m := make(map[ID]int, fs.Count())
	for i := 0; i < fs.Count(); i++ {
		m[fs.Item(i).ID()] = i
	}
	return m
}

func serializeFeature(pf *PartFeature, sk SketchIndexer, idx map[ID]int) (FeatureData, error) {
	fd := FeatureData{Kind: pf.Kind(), Name: pf.name, Suppressed: pf.suppress}
	switch f := pf.feature.(type) {
	case *ExtrudeFeature:
		ed, err := serializeExtrude(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.Extrude = ed
	case *FilletFeature:
		fd.Fillet = &EdgeDressData{Edges: encodeKeys(f.def.EdgeKeys), Value: evalFloat(f.def.Radius)}
	case *ChamferFeature:
		fd.Chamfer = &EdgeDressData{Edges: encodeKeys(f.def.EdgeKeys), Value: evalFloat(f.def.Distance)}
	case *ShellFeature:
		fd.Shell = &FaceDressData{Faces: encodeKeys(f.def.RemovedFaceKeys), Value: evalFloat(f.def.Thickness)}
	case *FaceDraftFeature:
		fd.Draft = &FaceDressData{Faces: encodeKeys(f.def.FaceKeys), Value: evalFloat(f.def.Angle)}
	case *ThreadFeature:
		fd.Thread = &ThreadData{Face: encodeKey(f.def.FaceKey), Designation: f.def.Designation}
	case *HoleFeature:
		h, err := serializeHole(f.def)
		if err != nil {
			return FeatureData{}, err
		}
		fd.Hole = h
	case *BossFeature:
		fd.Boss = &BossData{Face: encodeKey(f.def.PlacementFaceKey), Diameter: evalFloat(f.def.Diameter), Height: evalFloat(f.def.Height)}
	case *CombineFeature:
		op, err := operationName(f.def.Operation)
		if err != nil {
			return FeatureData{}, err
		}
		fd.Combine = &CombineData{Target: f.def.TargetIndex, Tool: f.def.ToolIndex, Operation: op}
	case *RectangularPatternFeature:
		src, err := sourceIndices(f.def.SourceFeatures, idx)
		if err != nil {
			return FeatureData{}, err
		}
		fd.RectPattern = &RectPatternData{Source: src, CountX: evalInt(f.def.CountX), CountY: evalInt(f.def.CountY)}
	case *CircularPatternFeature:
		src, err := sourceIndices(f.def.SourceFeatures, idx)
		if err != nil {
			return FeatureData{}, err
		}
		fd.CircPattern = &CircPatternData{Source: src, Count: evalInt(f.def.Count), Angle: evalFloat(f.def.Angle)}
	case *SketchDrivenPatternFeature:
		src, err := sourceIndices(f.def.SourceFeatures, idx)
		if err != nil {
			return FeatureData{}, err
		}
		fd.SketchPattern = &SketchDrivenPatternData{Source: src, PointCount: evalInt(f.def.PointCount)}
	case *MirrorFeature:
		src, err := sourceIndices(f.def.SourceFeatures, idx)
		if err != nil {
			return FeatureData{}, err
		}
		fd.Mirror = &MirrorData{Source: src, Plane: encodeKey(f.def.MirrorPlaneKey)}
	default:
		return FeatureData{}, fmt.Errorf("no serialization codec for feature kind %q", pf.Kind())
	}
	return fd, nil
}

func serializeExtrude(def *ExtrudeDefinition, sk SketchIndexer) (*ExtrudeData, error) {
	if def.Extent.Type != DistanceExtent {
		return nil, fmt.Errorf("only distance extents are serializable (got extent type %d)", def.Extent.Type)
	}
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("extrude references a sketch that is not in the part")
	}
	op, err := operationName(def.Operation)
	if err != nil {
		return nil, err
	}
	return &ExtrudeData{
		Sketch:    idx,
		Profile:   def.ProfileIndex,
		Operation: op,
		Distance:  def.Extent.distance(),
		Taper:     def.Taper,
	}, nil
}

// ApplyRecipe rebuilds the feature program from its serialized form, in order. It
// tracks the features restored so far so a pattern/mirror can resolve the earlier
// features it replicates (recorded as program indices). The caller recomputes
// afterward to regenerate geometry.
func (fs *PartFeatures) ApplyRecipe(data []FeatureData, sk SketchIndexer) error {
	restored := make([]*PartFeature, 0, len(data))
	for i, fd := range data {
		pf, err := buildFeature(fs, fd, sk, restored)
		if err != nil {
			return fmt.Errorf("feature %d (%s): %w", i, fd.Kind, err)
		}
		applyFeatureState(pf, fd)
		restored = append(restored, pf)
	}
	return nil
}

// buildFeature reconstructs one feature from its payload, erroring on an unknown kind
// or a missing payload (no silent loss). Dress-up edge/face keys re-bind to the
// regenerated topology on the next recompute (kernel topo FindEdgeByKey/FindFaceByKey);
// patterns resolve their source features from restored (the features built so far).
func buildFeature(fs *PartFeatures, fd FeatureData, sk SketchIndexer, restored []*PartFeature) (*PartFeature, error) {
	du := NewDressUpFeatures(fs)
	switch fd.Kind {
	case "extrude":
		return requireExtrude(fs, fd.Extrude, sk)
	case "fillet":
		d, err := requireEdgeDress(fd.Fillet, "fillet")
		if err != nil {
			return nil, err
		}
		return du.AddFillet(d.keys, constFloat(d.value)), nil
	case "chamfer":
		d, err := requireEdgeDress(fd.Chamfer, "chamfer")
		if err != nil {
			return nil, err
		}
		return du.AddChamfer(d.keys, constFloat(d.value)), nil
	case "shell":
		d, err := requireFaceDress(fd.Shell, "shell")
		if err != nil {
			return nil, err
		}
		return du.AddShell(d.keys, constFloat(d.value)), nil
	case "draft":
		d, err := requireFaceDress(fd.Draft, "draft")
		if err != nil {
			return nil, err
		}
		return du.AddDraft(d.keys, constFloat(d.value)), nil
	case "thread":
		if fd.Thread == nil {
			return nil, fmt.Errorf("thread feature is missing its payload")
		}
		key, err := decodeKey(fd.Thread.Face)
		if err != nil {
			return nil, err
		}
		return du.AddThread(key, fd.Thread.Designation), nil
	case "hole":
		return restoreHole(fs, fd.Hole)
	case "boss":
		return restoreBoss(fs, fd.Boss)
	case "combine":
		return restoreCombine(fs, fd.Combine)
	case "rectangular-pattern":
		return restoreRectPattern(fs, fd.RectPattern, restored)
	case "circular-pattern":
		return restoreCircPattern(fs, fd.CircPattern, restored)
	case "sketch-driven-pattern":
		return restoreSketchPattern(fs, fd.SketchPattern, restored)
	case "mirror":
		return restoreMirror(fs, fd.Mirror, restored)
	default:
		return nil, fmt.Errorf("no restore codec for feature kind %q", fd.Kind)
	}
}

// requireExtrude restores an extrude, erroring on a missing payload.
func requireExtrude(fs *PartFeatures, ed *ExtrudeData, sk SketchIndexer) (*PartFeature, error) {
	if ed == nil {
		return nil, fmt.Errorf("extrude feature is missing its payload")
	}
	return restoreExtrude(fs, ed, sk)
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

func restoreExtrude(fs *PartFeatures, ed *ExtrudeData, sk SketchIndexer) (*PartFeature, error) {
	skt, ok := sk.At(ed.Sketch)
	if !ok {
		return nil, fmt.Errorf("extrude references sketch index %d, which does not exist", ed.Sketch)
	}
	op, err := parseOperation(ed.Operation)
	if err != nil {
		return nil, err
	}
	dist := ed.Distance
	pf := NewExtrudeFeatures(fs).AddByDistanceExtent(skt, ed.Profile, op, func() float64 { return dist })
	if ed.Taper != 0 {
		pf.feature.(*ExtrudeFeature).def.Taper = ed.Taper
	}
	return pf, nil
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

// evalFloat reads a dress-up's scalar (a closure, typically a parameter); a nil closure
// reads as 0. constFloat is its inverse for restore — a fixed value (parametric
// dress-up values arrive with the dimension-driven API, like extrude distance).
func evalFloat(fn func() float64) float64 {
	if fn == nil {
		return 0
	}
	return fn()
}

func constFloat(v float64) func() float64 { return func() float64 { return v } }

// evalInt / constInt are the integer counterparts for pattern counts.
func evalInt(fn func() int) int {
	if fn == nil {
		return 0
	}
	return fn()
}

func constInt(v int) func() int { return func() int { return v } }

// applyFeatureState restores the per-feature engine state (name, suppression).
func applyFeatureState(pf *PartFeature, fd FeatureData) {
	if fd.Name != "" {
		pf.SetName(fd.Name)
	}
	if fd.Suppressed {
		pf.SetSuppressed(true)
	}
}

// operationName / parseOperation map the boolean operation to/from a stable name.
func operationName(op ops.PartFeatureOperation) (string, error) {
	switch op {
	case ops.Join:
		return "join", nil
	case ops.Cut:
		return "cut", nil
	case ops.Intersect:
		return "intersect", nil
	case ops.NewBody:
		return "newBody", nil
	default:
		return "", fmt.Errorf("unknown feature operation %d", op)
	}
}

func parseOperation(name string) (ops.PartFeatureOperation, error) {
	switch name {
	case "join":
		return ops.Join, nil
	case "cut":
		return ops.Cut, nil
	case "intersect":
		return ops.Intersect, nil
	case "newBody":
		return ops.NewBody, nil
	default:
		return 0, fmt.Errorf("unknown feature operation %q (want join|cut|intersect|newBody)", name)
	}
}
