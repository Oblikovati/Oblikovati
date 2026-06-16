// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/seq"
	"oblikovati.org/model/sketch"
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
	Kind           string              `yaml:"kind"`
	Name           string              `yaml:"name,omitempty"`
	Suppressed     bool                `yaml:"suppressed,omitempty"`
	Seq            uint64              `yaml:"seq,omitempty"` // global creation stamp; see model/seq
	Extrude        *ExtrudeData        `yaml:"extrude,omitempty"`
	Fillet         *EdgeDressData      `yaml:"fillet,omitempty"`
	Chamfer        *EdgeDressData      `yaml:"chamfer,omitempty"`
	Shell          *FaceDressData      `yaml:"shell,omitempty"`
	Draft          *FaceDressData      `yaml:"draft,omitempty"`
	Lip            *LipData            `yaml:"lip,omitempty"`
	Simplify       *SimplifyData       `yaml:"simplify,omitempty"`
	Unwrap         *UnwrapData         `yaml:"unwrap,omitempty"`
	MeshSolid      *MeshSolidData      `yaml:"meshSolid,omitempty"`
	ModelTolerance *ModelToleranceData `yaml:"modelTolerance,omitempty"`
	Grill          *GrillData          `yaml:"grill,omitempty"`
	Thread         *ThreadData         `yaml:"thread,omitempty"`
	Hole           *HoleData           `yaml:"hole,omitempty"`
	Boss           *BossData           `yaml:"boss,omitempty"`
	Rib            *RibData            `yaml:"rib,omitempty"`
	Emboss         *EmbossData         `yaml:"emboss,omitempty"`
	Combine        *CombineData        `yaml:"combine,omitempty"`
	SplitSolid     *SplitSolidData     `yaml:"splitSolid,omitempty"`
	DirectEdit     *DirectEditData     `yaml:"directEdit,omitempty"`

	RectPattern   *RectPatternData         `yaml:"rectangularPattern,omitempty"`
	CircPattern   *CircPatternData         `yaml:"circularPattern,omitempty"`
	SketchPattern *SketchDrivenPatternData `yaml:"sketchDrivenPattern,omitempty"`
	Mirror        *MirrorData              `yaml:"mirror,omitempty"`

	BoundaryPatch *BoundaryPatchData `yaml:"boundaryPatch,omitempty"`
	RuledSurface  *RuledSurfaceData  `yaml:"ruledSurface,omitempty"`
	FaceEdit      *FaceEditData      `yaml:"faceEdit,omitempty"`
	Thicken       *ThickenData       `yaml:"thicken,omitempty"`
	Revolve       *RevolveData       `yaml:"revolve,omitempty"`
	Coil          *CoilData          `yaml:"coil,omitempty"`
	Sweep         *SweepData         `yaml:"sweep,omitempty"`
	Loft          *LoftData          `yaml:"loft,omitempty"`
	Move          *MoveData          `yaml:"move,omitempty"`
	Bend          *BendData          `yaml:"bend,omitempty"`
	Decal         *DecalData         `yaml:"decal,omitempty"`
	Reference     *ReferenceData     `yaml:"reference,omitempty"`
	Client        *ClientData        `yaml:"client,omitempty"`
	Mark          *MarkData          `yaml:"mark,omitempty"`
	Finish        *FinishData        `yaml:"finish,omitempty"`
	Import        *ImportData        `yaml:"import,omitempty"`

	DerivedAssembly *DerivedAssemblyData `yaml:"derivedAssembly,omitempty"`
	DerivedPart     *DerivedPartData     `yaml:"derivedPart,omitempty"`
	Shrinkwrap      *ShrinkwrapData      `yaml:"shrinkwrap,omitempty"`

	SheetMetalFace   *SheetMetalFaceData   `yaml:"sheetMetalFace,omitempty"`   // M13-F02
	SheetMetalFlange *SheetMetalFlangeData `yaml:"sheetMetalFlange,omitempty"` // M13-F02
	SheetMetalHem    *SheetMetalHemData    `yaml:"sheetMetalHem,omitempty"`    // M13-F02
	SheetMetalBend   *SheetMetalBendData   `yaml:"sheetMetalBend,omitempty"`   // M13-F02
	SheetMetalFold   *SheetMetalFoldData   `yaml:"sheetMetalFold,omitempty"`   // M13-F02
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
	fd := FeatureData{Kind: pf.Kind(), Name: pf.name, Suppressed: pf.suppress, Seq: pf.seq}
	switch f := pf.feature.(type) {
	case *ExtrudeFeature:
		ed, err := serializeExtrude(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.Extrude = ed
	case *FilletFeature:
		if len(f.def.EdgeSets) > 0 {
			fd.Fillet = &EdgeDressData{Sets: serializeFilletSets(f.def.EdgeSets), CornerType: int32(f.def.CornerType)}
			break
		}
		fd.Fillet = &EdgeDressData{Edges: encodeKeys(f.def.EdgeKeys), Value: evalFloat(f.def.Radius), CornerType: int32(f.def.CornerType)}
	case *ChamferFeature:
		flat := f.def.FlatCorners
		fd.Chamfer = &EdgeDressData{
			Edges: encodeKeys(f.def.EdgeKeys), Value: evalFloat(f.def.Distance), FlatCorners: &flat,
			ChamferType: int32(f.def.Type), Value2: evalFloat(f.def.Distance2), Angle: evalFloat(f.def.Angle),
		}
	case *ShellFeature:
		fd.Shell = &FaceDressData{Faces: encodeKeys(f.def.RemovedFaceKeys), Value: evalFloat(f.def.Thickness)}
	case *FaceDraftFeature:
		p := f.def.PullDir
		fd.Draft = &FaceDressData{Faces: encodeKeys(f.def.FaceKeys), Value: evalFloat(f.def.Angle), Pull: []float64{p.X, p.Y, p.Z}}
	case *LipFeature:
		fd.Lip = &LipData{Edges: encodeKeys(f.def.EdgeKeys), Width: evalFloat(f.def.Width), Height: evalFloat(f.def.Height), Groove: f.def.Groove}
	case *SimplifyFeature:
		fd.Simplify = &SimplifyData{RemoveFaces: encodeKeys(f.def.RemoveFaceKeys), FillVoids: f.def.FillVoids}
	case *UnwrapFeature:
		fd.Unwrap = &UnwrapData{Face: encodeKey(f.def.FaceKey)}
	case *MeshSolidFeature:
		fd.MeshSolid = serializeMeshSolid(f.geom)
	case *ModelToleranceFeature:
		fd.ModelTolerance = serializeModelTolerance(f.def)
	case *GrillFeature:
		gd, err := serializeGrill(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.Grill = gd
	case *ThreadFeature:
		fd.Thread = &ThreadData{Face: encodeKey(f.def.FaceKey), Designation: f.def.Designation, Cut: f.def.Cut,
			Class: f.def.Class, Tapered: f.def.Tapered, ModelDiameter: threadModelDiameterName(f.def.ModelDiameter)}
	case *HoleFeature:
		h, err := serializeHole(f.def)
		if err != nil {
			return FeatureData{}, err
		}
		fd.Hole = h
	case *BossFeature:
		fd.Boss = &BossData{Face: encodeKey(f.def.PlacementFaceKey), Diameter: evalFloat(f.def.Diameter), Height: evalFloat(f.def.Height)}
	case *RibFeature:
		rd, err := serializeRib(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.Rib = rd
	case *EmbossFeature:
		ed, err := serializeEmboss(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.Emboss = ed
	case *SplitSolidFeature:
		sd, err := serializeSplitSolid(f.def)
		if err != nil {
			return FeatureData{}, err
		}
		fd.SplitSolid = sd
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
			Options: encodePatternOptions(f.def.Options),
		}
	case *CircularPatternFeature:
		src, err := sourceIndices(f.def.SourceFeatures, idx)
		if err != nil {
			return FeatureData{}, err
		}
		fd.CircPattern = &CircPatternData{
			Source: src, Count: evalInt(f.def.Count), Angle: evalFloat(f.def.Angle),
			AxisPoint: encodePoint3(f.def.AxisPoint), AxisDir: encodeVec3(f.def.AxisDir),
			Options: encodePatternOptions(f.def.Options),
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
	case *BendPartFeature:
		bd, err := serializeBend(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.Bend = bd
	case *SheetMetalFaceFeature:
		sm, err := serializeSheetMetalFace(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.SheetMetalFace = sm
	case *SheetMetalFlangeFeature:
		fd.SheetMetalFlange = serializeSheetMetalFlange(f.def)
	case *SheetMetalHemFeature:
		fd.SheetMetalHem = serializeSheetMetalHem(f.def)
	case *SheetMetalBendFeature:
		smb, err := serializeSheetMetalBend(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.SheetMetalBend = smb
	case *SheetMetalFoldFeature:
		smf, err := serializeSheetMetalFold(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.SheetMetalFold = smf
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
	case *ImportedBodyFeature:
		fd.Import = serializeImportedBody(f)
	case *DirectEditFeature:
		fd.DirectEdit = serializeDirectEdit(f.def)
	case *MoveFaceFeature:
		fd.FaceEdit = serializeMoveFace(f)
	case *FaceOffsetFeature:
		fd.FaceEdit = &FaceEditData{Faces: encodeKeys(f.FaceKeys()), Distance: f.Distance(),
			Approximation: approximationName(f.Approximation())}
	case *ReplaceFaceFeature:
		fd.FaceEdit = &FaceEditData{Faces: encodeKeys(f.FaceKeys()), Target: encodeKey(f.TargetKey())}
	case *ThickenFeature:
		fd.Thicken = &ThickenData{Value: f.Thickness(), Approximation: approximationName(f.Approximation())}
	case faceEditor:
		fd.FaceEdit = &FaceEditData{Faces: encodeKeys(f.FaceKeys())}
	case *DerivedAssemblyComponent:
		fd.DerivedAssembly = serializeDerivedAssembly(f)
	case *DerivedPartComponent:
		fd.DerivedPart = serializeDerivedPart(f)
	case *ShrinkwrapComponent:
		fd.Shrinkwrap = serializeShrinkwrap(f)
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
		corner := types.FilletCornerType(fd.Fillet.cornerTypeOrZero())
		if corner == 0 {
			corner = types.FilletCornerMiter // absent / older recipe ⇒ the miter default
		}
		if fd.Fillet != nil && len(fd.Fillet.Sets) > 0 {
			sets, err := restoreFilletSets(fd.Fillet.Sets)
			if err != nil {
				return nil, err
			}
			return du.AddFilletSetsCorner(sets, corner), nil
		}
		d, err := requireEdgeDress(fd.Fillet, "fillet")
		if err != nil {
			return nil, err
		}
		return du.AddFilletCorner(d.keys, constFloat(d.value), corner), nil
	case "chamfer":
		d, err := requireEdgeDress(fd.Chamfer, "chamfer")
		if err != nil {
			return nil, err
		}
		// Build the def directly so the stored flat-corner flag round-trips for EVERY mode
		// (the asymmetric builders default it to true, but a recipe carries the saved value).
		def := &ChamferDefinition{
			EdgeKeys: d.keys, Distance: constFloat(d.value),
			Type: types.ChamferType(fd.Chamfer.ChamferType), FlatCorners: chamferFlatCornersOr(fd.Chamfer.FlatCorners),
		}
		switch def.Type {
		case types.ChamferTwoDistances:
			def.Distance2 = constFloat(fd.Chamfer.Value2)
		case types.ChamferDistanceAndAngle:
			def.Angle = constFloat(fd.Chamfer.Angle)
		}
		return du.addChamfer(def), nil
	case "lip":
		return restoreLip(fs, fd.Lip)
	case "simplify":
		return restoreSimplify(fs, fd.Simplify)
	case "unwrap":
		return restoreUnwrap(fs, fd.Unwrap)
	case "mesh-solid":
		return restoreMeshSolid(fs, fd.MeshSolid)
	case "modelTolerance":
		return restoreModelTolerance(fs, fd.ModelTolerance)
	case "grill":
		return restoreGrill(fs, fd.Grill, sk)
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
		return du.AddDraftPull(d.keys, draftPull(fd.Draft.Pull), constFloat(d.value)), nil
	case "thread":
		if fd.Thread == nil {
			return nil, fmt.Errorf("thread feature is missing its payload")
		}
		key, err := decodeKey(fd.Thread.Face)
		if err != nil {
			return nil, err
		}
		md, err := threadModelDiameterOf(fd.Thread.ModelDiameter)
		if err != nil {
			return nil, err
		}
		return du.AddThreadDef(&ThreadDefinition{FaceKey: key, Designation: fd.Thread.Designation,
			Cut: fd.Thread.Cut, Class: fd.Thread.Class, Tapered: fd.Thread.Tapered, ModelDiameter: md}), nil
	case "hole":
		return restoreHole(fs, fd.Hole)
	case "boss":
		return restoreBoss(fs, fd.Boss)
	case "combine":
		return restoreCombine(fs, fd.Combine)
	case "directEdit":
		return restoreDirectEdit(fs, fd.DirectEdit)
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
	case "split", "move-face", "face-offset", "delete-face", "replace-face":
		return restoreFaceEdit(fs, fd.Kind, fd.FaceEdit)
	case "thicken":
		if fd.Thicken == nil {
			return nil, fmt.Errorf("thicken feature is missing its payload")
		}
		approx, err := approximationOf(fd.Thicken.Approximation)
		if err != nil {
			return nil, err
		}
		pf := NewModifyFeatures(fs).AddThicken(fd.Thicken.Value)
		pf.Definition().(*ThickenFeature).SetApproximation(approx)
		return pf, nil
	case "revolve":
		return restoreRevolve(fs, fd.Revolve, sk, work)
	case "coil":
		return restoreCoil(fs, fd.Coil, sk, work)
	case "sweep":
		return restoreSweep(fs, fd.Sweep, sk)
	case "loft":
		return restoreLoft(fs, fd.Loft, sk)
	case "rib":
		return restoreRib(fs, fd.Rib, sk)
	case "emboss":
		return restoreEmboss(fs, fd.Emboss, sk)
	case "splitSolid":
		return restoreSplitSolid(fs, fd.SplitSolid, work)
	case "move":
		return restoreMove(fs, fd.Move)
	case "bend-part":
		return restoreBend(fs, fd.Bend, sk)
	case "sheet-metal-face":
		return restoreSheetMetalFace(fs, fd.SheetMetalFace, sk)
	case "sheet-metal-flange":
		return restoreSheetMetalFlange(fs, fd.SheetMetalFlange)
	case "sheet-metal-hem":
		return restoreSheetMetalHem(fs, fd.SheetMetalHem)
	case "sheet-metal-bend":
		return restoreSheetMetalBend(fs, fd.SheetMetalBend, sk)
	case "sheet-metal-fold":
		return restoreSheetMetalFold(fs, fd.SheetMetalFold, sk)
	case "importedBody":
		return restoreImportedBody(fs, fd.Import)
	case "decal", "reference", "client", "mark", "finish":
		return restoreCosmetic(fs, fd)
	case "derivedAssembly":
		return restoreDerivedAssembly(fs, fd.DerivedAssembly)
	case "derived":
		return restoreDerivedPart(fs, fd.DerivedPart)
	case "shrinkwrap":
		return restoreShrinkwrap(fs, fd.Shrinkwrap)
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

// applyFeatureState restores the per-feature engine state (name, suppression, and the
// global creation stamp so a reopened document keeps its sketch/feature/work interleaving).
func applyFeatureState(pf *PartFeature, fd FeatureData) {
	if fd.Name != "" {
		pf.SetName(fd.Name)
	}
	if fd.Suppressed {
		pf.SetSuppressed(true)
	}
	seq.Restore(&pf.seq, fd.Seq)
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
