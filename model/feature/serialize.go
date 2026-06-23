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
	FaceFillet     *FaceFilletData     `yaml:"faceFillet,omitempty"`
	FullRound      *FullRoundData      `yaml:"fullRound,omitempty"`
	RuleFillet     *RuleFilletData     `yaml:"ruleFillet,omitempty"`
	SnapFit        *SnapFitData        `yaml:"snapFit,omitempty"`
	Rest           *RestData           `yaml:"rest,omitempty"`
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
	DeleteBody     *DeleteBodyData     `yaml:"deleteBody,omitempty"`
	SplitSolid     *SplitSolidData     `yaml:"splitSolid,omitempty"`
	DirectEdit     *DirectEditData     `yaml:"directEdit,omitempty"`

	RectPattern   *RectPatternData         `yaml:"rectangularPattern,omitempty"`
	CircPattern   *CircPatternData         `yaml:"circularPattern,omitempty"`
	SketchPattern *SketchDrivenPatternData `yaml:"sketchDrivenPattern,omitempty"`
	Mirror        *MirrorData              `yaml:"mirror,omitempty"`

	BoundaryPatch    *BoundaryPatchData    `yaml:"boundaryPatch,omitempty"`
	RuledSurface     *RuledSurfaceData     `yaml:"ruledSurface,omitempty"`
	Rebuild          *RebuildData          `yaml:"rebuild,omitempty"`
	ControlPointEdit *ControlPointEditData `yaml:"controlPointEdit,omitempty"`
	NurbsPlane       *NurbsPlaneData       `yaml:"nurbsPlane,omitempty"`
	Match            *MatchData            `yaml:"match,omitempty"`
	ExtendSurface    *ExtendSurfaceData    `yaml:"extendSurface,omitempty"`
	Untrim           *UntrimData           `yaml:"untrim,omitempty"`
	FillSurface      *FillSurfaceData      `yaml:"fillSurface,omitempty"`
	FaceEdit         *FaceEditData         `yaml:"faceEdit,omitempty"`
	Thicken          *ThickenData          `yaml:"thicken,omitempty"`
	Revolve          *RevolveData          `yaml:"revolve,omitempty"`
	Coil             *CoilData             `yaml:"coil,omitempty"`
	Sweep            *SweepData            `yaml:"sweep,omitempty"`
	Loft             *LoftData             `yaml:"loft,omitempty"`
	Move             *MoveData             `yaml:"move,omitempty"`
	Bend             *BendData             `yaml:"bend,omitempty"`
	Decal            *DecalData            `yaml:"decal,omitempty"`
	Reference        *ReferenceData        `yaml:"reference,omitempty"`
	Client           *ClientData           `yaml:"client,omitempty"`
	Mark             *MarkData             `yaml:"mark,omitempty"`
	Finish           *FinishData           `yaml:"finish,omitempty"`
	Import           *ImportData           `yaml:"import,omitempty"`

	DerivedAssembly *DerivedAssemblyData `yaml:"derivedAssembly,omitempty"`
	DerivedPart     *DerivedPartData     `yaml:"derivedPart,omitempty"`
	Shrinkwrap      *ShrinkwrapData      `yaml:"shrinkwrap,omitempty"`

	SheetMetalFace          *SheetMetalFaceData          `yaml:"sheetMetalFace,omitempty"`          // M13-F02
	SheetMetalFlange        *SheetMetalFlangeData        `yaml:"sheetMetalFlange,omitempty"`        // M13-F02
	SheetMetalHem           *SheetMetalHemData           `yaml:"sheetMetalHem,omitempty"`           // M13-F02
	SheetMetalBend          *SheetMetalBendData          `yaml:"sheetMetalBend,omitempty"`          // M13-F02
	SheetMetalFold          *SheetMetalFoldData          `yaml:"sheetMetalFold,omitempty"`          // M13-F02
	SheetMetalCorner        *SheetMetalCornerData        `yaml:"sheetMetalCorner,omitempty"`        // M13-F02
	SheetMetalContourFlange *SheetMetalContourFlangeData `yaml:"sheetMetalContourFlange,omitempty"` // M13-F02
	SheetMetalLoftedFlange  *SheetMetalLoftedFlangeData  `yaml:"sheetMetalLoftedFlange,omitempty"`  // M13-F02
	SheetMetalContourRoll   *SheetMetalContourRollData   `yaml:"sheetMetalContourRoll,omitempty"`   // M13-F02
	SheetMetalCornerSeam    *SheetMetalCornerSeamData    `yaml:"sheetMetalCornerSeam,omitempty"`    // M13-F02
	SheetMetalCut           *SheetMetalCutData           `yaml:"sheetMetalCut,omitempty"`           // M13-F03
	SheetMetalRip           *SheetMetalRipData           `yaml:"sheetMetalRip,omitempty"`           // M13-F03
	SheetMetalPunch         *SheetMetalPunchData         `yaml:"sheetMetalPunch,omitempty"`         // M13-F03
	SheetMetalLip           *SheetMetalLipData           `yaml:"sheetMetalLip,omitempty"`           // M13-F03
	SheetMetalCosmeticBend  *SheetMetalCosmeticBendData  `yaml:"sheetMetalCosmeticBend,omitempty"`  // M13-F03
	SheetMetalUnfold        *SheetMetalUnfoldData        `yaml:"sheetMetalUnfold,omitempty"`        // M13-F04
	SheetMetalRefold        *SheetMetalRefoldData        `yaml:"sheetMetalRefold,omitempty"`        // M13-F04
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
			fd.Fillet = &EdgeDressData{Sets: serializeFilletSets(f.def.EdgeSets), CornerType: int32(f.def.CornerType), CrossSection: crossSectionWire(f.def.CrossSection), Rho: f.def.Rho}
			break
		}
		fd.Fillet = &EdgeDressData{Edges: encodeKeys(f.def.EdgeKeys), Value: evalFloat(f.def.Radius), CornerType: int32(f.def.CornerType), CrossSection: crossSectionWire(f.def.CrossSection), Rho: f.def.Rho, GeomEdges: encodeGeomEdges(f.def.GeomEdges)}
	case *FaceFilletFeature:
		fd.FaceFillet = &FaceFilletData{FacesA: encodeKeys(f.def.FaceKeysA), FacesB: encodeKeys(f.def.FaceKeysB), Value: evalFloat(f.def.Radius)}
	case *FullRoundFilletFeature:
		fd.FullRound = &FullRoundData{Side1: encodeKeys(f.def.Side1Keys), Center: encodeKeys(f.def.CenterKeys), Side2: encodeKeys(f.def.Side2Keys)}
	case *RuleFilletFeature:
		fd.RuleFillet = &RuleFilletData{Rule: f.def.Rule.String(), Value: evalFloat(f.def.Radius)}
	case *SnapFitFeature:
		fd.SnapFit = &SnapFitData{
			Length: evalFloat(f.def.Length), Width: evalFloat(f.def.Width), Thickness: evalFloat(f.def.Thickness),
			CatchLength: evalFloat(f.def.CatchLength), CatchHeight: evalFloat(f.def.CatchHeight),
		}
	case *ChamferFeature:
		flat := f.def.FlatCorners
		fd.Chamfer = &EdgeDressData{
			Edges: encodeKeys(f.def.EdgeKeys), Value: evalFloat(f.def.Distance), FlatCorners: &flat,
			ChamferType: int32(f.def.Type), Value2: evalFloat(f.def.Distance2), Angle: evalFloat(f.def.Angle),
			GeomEdges: encodeGeomEdges(f.def.GeomEdges),
		}
	case *ShellFeature:
		fd.Shell = &FaceDressData{Faces: encodeKeys(f.def.RemovedFaceKeys), Value: evalFloat(f.def.Thickness), GeomFaces: encodeGeomFaces(f.def.GeomFaces)}
	case *FaceDraftFeature:
		p := f.def.PullDir
		fd.Draft = &FaceDressData{Faces: encodeKeys(f.def.FaceKeys), Value: evalFloat(f.def.Angle), Pull: []float64{p.X, p.Y, p.Z}, GeomFaces: encodeGeomFaces(f.def.GeomFaces)}
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
	case *RestFeature:
		rd, err := serializeRest(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.Rest = rd
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
	case *DeleteBodyFeature:
		fd.DeleteBody = serializeDeleteBody(f.def)
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
	case *RebuildFeature:
		fd.Rebuild = serializeRebuild(f.def)
	case *ControlPointEditFeature:
		fd.ControlPointEdit = serializeControlPointEdit(f.def)
	case *NurbsPlaneFeature:
		fd.NurbsPlane = serializeNurbsPlane(f.def)
	case *MatchFeature:
		fd.Match = serializeMatch(f.def)
	case *ExtendSurfaceFeature:
		fd.ExtendSurface = serializeExtendSurface(f.def)
	case *UntrimFeature:
		fd.Untrim = &UntrimData{}
	case *FillFeature:
		fd.FillSurface = serializeFillSurface(f.def)
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
	case *SheetMetalCosmeticBendFeature:
		smcb, err := serializeSheetMetalCosmeticBend(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.SheetMetalCosmeticBend = smcb
	case *SheetMetalRipFeature:
		smr, err := serializeSheetMetalRip(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.SheetMetalRip = smr
	case *SheetMetalPunchFeature:
		smp, err := serializeSheetMetalPunch(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.SheetMetalPunch = smp
	case *SheetMetalLipFeature:
		fd.SheetMetalLip = serializeSheetMetalLip(f.def)
	case *SheetMetalFoldFeature:
		smf, err := serializeSheetMetalFold(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.SheetMetalFold = smf
	case *SheetMetalCornerFeature:
		fd.SheetMetalCorner = serializeSheetMetalCorner(f.def)
	case *SheetMetalContourFlangeFeature:
		smcf, err := serializeSheetMetalContourFlange(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.SheetMetalContourFlange = smcf
	case *SheetMetalLoftedFlangeFeature:
		smlf, err := serializeSheetMetalLoftedFlange(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.SheetMetalLoftedFlange = smlf
	case *SheetMetalContourRollFeature:
		smcr, err := serializeSheetMetalContourRoll(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.SheetMetalContourRoll = smcr
	case *SheetMetalCornerSeamFeature:
		fd.SheetMetalCornerSeam = serializeSheetMetalCornerSeam(f.def)
	case *SheetMetalCutFeature:
		smc, err := serializeSheetMetalCut(f.def, sk)
		if err != nil {
			return FeatureData{}, err
		}
		fd.SheetMetalCut = smc
	case *SheetMetalUnfoldFeature:
		fd.SheetMetalUnfold = &SheetMetalUnfoldData{Bends: serializeBendTransforms(f.def.Bends)}
	case *SheetMetalRefoldFeature:
		fd.SheetMetalRefold = &SheetMetalRefoldData{Bends: serializeBendTransforms(f.def.Bends)}
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
		cross := fd.Fillet.crossSectionOrArc()
		rho := fd.Fillet.rhoOrZero()
		if fd.Fillet != nil && len(fd.Fillet.Sets) > 0 {
			sets, err := restoreFilletSets(fd.Fillet.Sets)
			if err != nil {
				return nil, err
			}
			return du.addFillet(&FilletDefinition{EdgeSets: sets, CornerType: corner, CrossSection: cross, Rho: rho}), nil
		}
		d, err := requireEdgeDress(fd.Fillet, "fillet")
		if err != nil {
			return nil, err
		}
		return du.addFillet(&FilletDefinition{
			EdgeKeys: d.keys, GeomEdges: d.geom, Radius: constFloat(d.value), CornerType: corner,
			CrossSection: cross, Rho: rho,
		}), nil
	case "face-fillet":
		if fd.FaceFillet == nil {
			return nil, fmt.Errorf("face-fillet feature is missing its payload")
		}
		a, err := decodeKeys(fd.FaceFillet.FacesA)
		if err != nil {
			return nil, err
		}
		b, err := decodeKeys(fd.FaceFillet.FacesB)
		if err != nil {
			return nil, err
		}
		return du.AddFaceFillet(a, b, constFloat(fd.FaceFillet.Value)), nil
	case "full-round-fillet":
		if fd.FullRound == nil {
			return nil, fmt.Errorf("full-round-fillet feature is missing its payload")
		}
		s1, err := decodeKeys(fd.FullRound.Side1)
		if err != nil {
			return nil, err
		}
		ctr, err := decodeKeys(fd.FullRound.Center)
		if err != nil {
			return nil, err
		}
		s2, err := decodeKeys(fd.FullRound.Side2)
		if err != nil {
			return nil, err
		}
		return du.AddFullRoundFillet(s1, ctr, s2), nil
	case "rule-fillet":
		if fd.RuleFillet == nil {
			return nil, fmt.Errorf("rule-fillet feature is missing its payload")
		}
		rule, ok := ParseRuleFilletRule(fd.RuleFillet.Rule)
		if !ok {
			return nil, fmt.Errorf("rule-fillet: unknown rule %q", fd.RuleFillet.Rule)
		}
		return du.AddRuleFillet(rule, constFloat(fd.RuleFillet.Value)), nil
	case "snap-fit":
		if fd.SnapFit == nil {
			return nil, fmt.Errorf("snap-fit feature is missing its payload")
		}
		d := fd.SnapFit
		return NewPlasticFeatures(fs).AddCantileverSnapFit(
			constFloat(d.Length), constFloat(d.Width), constFloat(d.Thickness),
			constFloat(d.CatchLength), constFloat(d.CatchHeight)), nil
	case "chamfer":
		d, err := requireEdgeDress(fd.Chamfer, "chamfer")
		if err != nil {
			return nil, err
		}
		// Build the def directly so the stored flat-corner flag round-trips for EVERY mode
		// (the asymmetric builders default it to true, but a recipe carries the saved value).
		def := &ChamferDefinition{
			EdgeKeys: d.keys, GeomEdges: d.geom, Distance: constFloat(d.value),
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
		return du.addShell(&ShellDefinition{
			RemovedFaceKeys: d.keys, GeomFaces: d.geomFaces, Thickness: constFloat(d.value),
		}), nil
	case "draft":
		d, err := requireFaceDress(fd.Draft, "draft")
		if err != nil {
			return nil, err
		}
		return du.addFaceDraft(&FaceDraftDefinition{
			FaceKeys: d.keys, GeomFaces: d.geomFaces, PullDir: draftPull(fd.Draft.Pull), Angle: constFloat(d.value),
		}), nil
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
	case "delete-body":
		return restoreDeleteBody(fs, fd.DeleteBody)
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
	case "rebuild-surface":
		return restoreRebuild(fs, fd.Rebuild)
	case "control-point-edit":
		return restoreControlPointEdit(fs, fd.ControlPointEdit)
	case "nurbs-plane":
		return restoreNurbsPlane(fs, fd.NurbsPlane)
	case "match-surface":
		return restoreMatch(fs, fd.Match)
	case "extend-surface":
		return restoreExtendSurface(fs, fd.ExtendSurface)
	case "untrim-surface":
		return restoreUntrim(fs, fd.Untrim)
	case "fill-surface":
		return restoreFillSurface(fs, fd.FillSurface)
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
	case "rest":
		return restoreRest(fs, fd.Rest, sk)
	case "splitSolid":
		return restoreSplitSolid(fs, fd.SplitSolid, work)
	case "move":
		return restoreMove(fs, fd.Move)
	case kindBendPart:
		return restoreBend(fs, fd.Bend, sk)
	case "sheet-metal-face":
		return restoreSheetMetalFace(fs, fd.SheetMetalFace, sk)
	case "sheet-metal-flange":
		return restoreSheetMetalFlange(fs, fd.SheetMetalFlange)
	case "sheet-metal-hem":
		return restoreSheetMetalHem(fs, fd.SheetMetalHem)
	case "sheet-metal-bend":
		return restoreSheetMetalBend(fs, fd.SheetMetalBend, sk)
	case "sheet-metal-cosmetic-bend":
		return restoreSheetMetalCosmeticBend(fs, fd.SheetMetalCosmeticBend, sk)
	case "sheet-metal-rip":
		return restoreSheetMetalRip(fs, fd.SheetMetalRip, sk)
	case "sheet-metal-punch":
		return restoreSheetMetalPunch(fs, fd.SheetMetalPunch, sk)
	case "sheet-metal-lip":
		return restoreSheetMetalLip(fs, fd.SheetMetalLip)
	case "sheet-metal-fold":
		return restoreSheetMetalFold(fs, fd.SheetMetalFold, sk)
	case "sheet-metal-corner":
		return restoreSheetMetalCorner(fs, fd.SheetMetalCorner)
	case "sheet-metal-contour-flange":
		return restoreSheetMetalContourFlange(fs, fd.SheetMetalContourFlange, sk)
	case "sheet-metal-lofted-flange":
		return restoreSheetMetalLoftedFlange(fs, fd.SheetMetalLoftedFlange, sk)
	case "sheet-metal-contour-roll":
		return restoreSheetMetalContourRoll(fs, fd.SheetMetalContourRoll, sk)
	case "sheet-metal-corner-seam":
		return restoreSheetMetalCornerSeam(fs, fd.SheetMetalCornerSeam)
	case "sheet-metal-cut":
		return restoreSheetMetalCut(fs, fd.SheetMetalCut, sk)
	case "sheet-metal-unfold":
		return restoreSheetMetalUnfold(fs, fd.SheetMetalUnfold)
	case "sheet-metal-refold":
		return restoreSheetMetalRefold(fs, fd.SheetMetalRefold)
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
