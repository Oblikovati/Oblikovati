// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
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

	BoundaryPatch *BoundaryPatchData `yaml:"boundaryPatch,omitempty"`
	RuledSurface  *RuledSurfaceData  `yaml:"ruledSurface,omitempty"`
	FaceEdit      *FaceEditData      `yaml:"faceEdit,omitempty"`
	Revolve       *RevolveData       `yaml:"revolve,omitempty"`
	Coil          *CoilData          `yaml:"coil,omitempty"`
	Sweep         *SweepData         `yaml:"sweep,omitempty"`
	Loft          *LoftData          `yaml:"loft,omitempty"`
	Move          *MoveData          `yaml:"move,omitempty"`
	Decal         *DecalData         `yaml:"decal,omitempty"`
	Reference     *ReferenceData     `yaml:"reference,omitempty"`
	Client        *ClientData        `yaml:"client,omitempty"`
	Mark          *MarkData          `yaml:"mark,omitempty"`
	Finish        *FinishData        `yaml:"finish,omitempty"`
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
		fd.RectPattern = &RectPatternData{
			Source: src, CountX: evalInt(f.def.CountX), CountY: evalInt(f.def.CountY),
			StepX: encodeVec3(f.def.StepX), StepY: encodeVec3(f.def.StepY),
		}
	case *CircularPatternFeature:
		src, err := sourceIndices(f.def.SourceFeatures, idx)
		if err != nil {
			return FeatureData{}, err
		}
		fd.CircPattern = &CircPatternData{
			Source: src, Count: evalInt(f.def.Count), Angle: evalFloat(f.def.Angle),
			AxisPoint: encodePoint3(f.def.AxisPoint), AxisDir: encodeVec3(f.def.AxisDir),
		}
	case *SketchDrivenPatternFeature:
		src, err := sourceIndices(f.def.SourceFeatures, idx)
		if err != nil {
			return FeatureData{}, err
		}
		fd.SketchPattern = &SketchDrivenPatternData{Source: src, Points: encodePoints(callPoints(f.def.Points))}
	case *MirrorFeature:
		src, err := sourceIndices(f.def.SourceFeatures, idx)
		if err != nil {
			return FeatureData{}, err
		}
		fd.Mirror = &MirrorData{
			Source: src, Plane: encodeKey(f.def.MirrorPlaneKey),
			Origin: encodePoint3(f.def.Origin), Normal: encodeVec3(f.def.Normal),
		}
	case *BoundaryPatchFeature:
		bp, err := serializeBoundaryPatch(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.BoundaryPatch = bp
	case *RuledSurfaceFeature:
		rs, err := serializeRuledSurface(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.RuledSurface = rs
	case *RevolveFeature:
		rv, err := serializeRevolve(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.Revolve = rv
	case *CoilFeature:
		cd, err := serializeCoil(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.Coil = cd
	case *SweepFeature:
		sw, err := serializeSweep(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.Sweep = sw
	case *LoftFeature:
		lo, err := serializeLoft(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.Loft = lo
	case *MoveFeature:
		fd.Move = serializeMove(f.def)
	case *DecalFeature:
		fd.Decal = &DecalData{Face: encodeKey(f.def.FaceKey), Image: f.def.Image}
	case *ReferenceFeature:
		fd.Reference = &ReferenceData{Label: f.def.Label, Source: encodeKey(f.def.SourceKey)}
	case *ClientFeature:
		fd.Client = &ClientData{AddIn: f.def.AddInID, Attributes: f.def.Attributes}
	case *MarkFeature:
		fd.Mark = &MarkData{Faces: encodeKeys(f.def.FaceKeys), Text: f.def.Text}
	case *FinishFeature:
		fd.Finish = &FinishData{Faces: encodeKeys(f.def.FaceKeys), Spec: f.def.Spec}
	case *MoveFaceFeature:
		t := f.Translation()
		fd.FaceEdit = &FaceEditData{Faces: encodeKeys(f.FaceKeys()), Translation: []float64{t.X, t.Y, t.Z}}
	case *FaceOffsetFeature:
		fd.FaceEdit = &FaceEditData{Faces: encodeKeys(f.FaceKeys()), Distance: f.Distance()}
	case faceEditor:
		fd.FaceEdit = &FaceEditData{Faces: encodeKeys(f.FaceKeys())}
	default:
		return FeatureData{}, fmt.Errorf("no serialization codec for feature kind %q", pf.Kind())
	}
	return fd, nil
}

// ApplyRecipe rebuilds the feature program from its serialized form, in order. It
// tracks the features restored so far so a pattern/mirror can resolve the earlier
// features it replicates (recorded as program indices). The caller recomputes
// afterward to regenerate geometry.
func (fs *PartFeatures) ApplyRecipe(data []FeatureData, sk SketchIndexer, work *WorkGeometry) error {
	restored := make([]*PartFeature, 0, len(data))
	for i, fd := range data {
		pf, err := buildFeature(fs, fd, sk, restored, work)
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
func buildFeature(fs *PartFeatures, fd FeatureData, sk SketchIndexer, restored []*PartFeature, work *WorkGeometry) (*PartFeature, error) {
	du := NewDressUpFeatures(fs)
	switch fd.Kind {
	case "extrude":
		return requireExtrude(fs, fd.Extrude, sk, work)
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
	case "boundary-patch":
		return restoreBoundaryPatch(fs, fd.BoundaryPatch, sk)
	case "ruled-surface":
		return restoreRuledSurface(fs, fd.RuledSurface, sk)
	case "split", "move-face", "face-offset", "delete-face", "replace-face", "thicken":
		return restoreFaceEdit(fs, fd.Kind, fd.FaceEdit)
	case "revolve":
		return restoreRevolve(fs, fd.Revolve, sk, work)
	case "coil":
		return restoreCoil(fs, fd.Coil, sk, work)
	case "sweep":
		return restoreSweep(fs, fd.Sweep, sk)
	case "loft":
		return restoreLoft(fs, fd.Loft, sk)
	case "move":
		return restoreMove(fs, fd.Move)
	case "decal", "reference", "client", "mark", "finish":
		return restoreCosmetic(fs, fd)
	default:
		return nil, fmt.Errorf("no restore codec for feature kind %q", fd.Kind)
	}
}

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
