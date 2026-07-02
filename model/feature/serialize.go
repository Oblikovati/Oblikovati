// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

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
	BridgeSurface    *BridgeSurfaceData    `yaml:"bridgeSurface,omitempty"`
	NetworkSurface   *NetworkSurfaceData   `yaml:"networkSurface,omitempty"`
	FairSurface      *FairSurfaceData      `yaml:"fairSurface,omitempty"`
	FitSurface       *FitSurfaceData       `yaml:"fitSurface,omitempty"`
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

// serializeFeature projects one feature into its FeatureData via the kind's registered codec (see
// serialize_registry.go). A kind without a codec errors rather than dropping the feature silently.
func serializeFeature(pf *PartFeature, sk SketchIndexer, idx map[ID]int) (FeatureData, error) {
	return serializeFeatureWith(featureCodecs, pf, sk, idx)
}

// serializeFeatureWith consults exactly the codec set it is handed — the injection
// seam proving nothing falls back to a global (#1617, audit B6).
func serializeFeatureWith(codecs featureCodecSet, pf *PartFeature, sk SketchIndexer, idx map[ID]int) (FeatureData, error) {
	c, ok := codecs[pf.Kind()]
	if !ok {
		return FeatureData{}, fmt.Errorf("no serialization codec for feature kind %q", pf.Kind())
	}
	fd := FeatureData{Kind: pf.Kind(), Name: pf.name, Suppressed: pf.suppress, Seq: pf.seq}
	if err := c.encode(&fd, pf.feature, sk, idx); err != nil {
		return FeatureData{}, err
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

// buildFeature reconstructs one feature from its payload via the kind's registered codec (see
// serialize_registry.go), erroring on an unknown kind or a missing payload (no silent loss). Dress-up
// edge/face keys re-bind to the regenerated topology on the next recompute (kernel topo
// FindEdgeByKey/FindFaceByKey); patterns resolve their source features from restored (the features
// built so far).
func buildFeature(fs *PartFeatures, fd FeatureData, sk SketchIndexer, restored []*PartFeature, work *WorkGeometry) (*PartFeature, error) {
	return buildFeatureWith(featureCodecs, fs, fd, sk, restored, work)
}

// buildFeatureWith is the decode half of the injection seam (#1617).
func buildFeatureWith(codecs featureCodecSet, fs *PartFeatures, fd FeatureData, sk SketchIndexer, restored []*PartFeature, work *WorkGeometry) (*PartFeature, error) {
	c, ok := codecs[fd.Kind]
	if !ok {
		return nil, fmt.Errorf("no restore codec for feature kind %q", fd.Kind)
	}
	return c.decode(&restoreContext{fs: fs, sk: sk, restored: restored, work: work}, fd)
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
