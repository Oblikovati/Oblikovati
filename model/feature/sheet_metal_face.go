// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// Sheet-metal base/secondary Face feature (M13-F02). A Face is the wall a sheet-metal part
// starts from: a closed sketch profile thickened by the active rule's constant thickness.
// The first Face is the base (a new body); a secondary Face joins the running sheet.
//
// The thickness is NOT stored on the feature — it is read live from the part's Thickness
// parameter at every recompute, so editing that parameter (via the rule) repropagates to
// every wall through the normal feature engine, and a restored part stays at one gauge
// without freezing a stale value. This is the defining sheet-metal invariant.

// sheetMetalThicknessParam is the well-known parameter the sheet-metal rule backs the
// material thickness with (compdef seeds it when the part enters the environment). Defined
// here too so the feature engine reads it without importing compdef (which imports feature).
const sheetMetalThicknessParam = "Thickness"

// SheetMetalFaceDefinition is the Face recipe: the profile to thicken, the side the material
// grows toward, and the boolean operation (new body for the base wall, join for a secondary
// wall). Thickness comes from the part rule at recompute time, not from here.
type SheetMetalFaceDefinition struct {
	Sketch       *sketch.Sketch
	ProfileIndex int
	Direction    ExtentDirection // material side: positive/negative/symmetric about the sketch plane
	Operation    ops.PartFeatureOperation
}

// SheetMetalFaceFeature thickens the profile into a wall each recompute.
type SheetMetalFaceFeature struct {
	def      *SheetMetalFaceDefinition
	featName string
}

// Definition returns the Face recipe.
func (f *SheetMetalFaceFeature) Definition() *SheetMetalFaceDefinition { return f.def }

// Kind identifies the feature for serialization and the model tree.
func (f *SheetMetalFaceFeature) Kind() string { return "sheet-metal-face" }

// Recompute resolves the profile, thickens it by the rule thickness on the chosen side, and
// applies the operation against the running bodies.
func (f *SheetMetalFaceFeature) Recompute(in Input) (Output, error) {
	profiles, err := resolveClosedProfiles(f.def.Sketch, []int{f.def.ProfileIndex}, "sheet-metal face")
	if err != nil {
		return Output{}, err
	}
	t, err := sheetThickness(in.Params)
	if err != nil {
		return Output{}, err
	}
	body := buildProfilePrisms(profiles, f.def.Sketch.Plane(), faceSpan(t, f.def.Direction), 0, f.featName)
	bodies, err := combine(in.Bodies, body, f.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// sheetThickness reads the live material thickness (database units) from the part's Thickness
// parameter, erroring when it is absent (not a sheet-metal part) or non-positive.
func sheetThickness(ps *param.Parameters) (float64, error) {
	if ps == nil {
		return 0, fmt.Errorf("sheet-metal face: no parameters available")
	}
	p, ok := ps.ByName(sheetMetalThicknessParam)
	if !ok {
		return 0, fmt.Errorf("sheet-metal face: no %q parameter (is this a sheet-metal part?)", sheetMetalThicknessParam)
	}
	t := p.ModelValue()
	if t <= 0 {
		return 0, fmt.Errorf("sheet-metal face: thickness must be positive, got %g", t)
	}
	return t, nil
}

// faceSpan returns the thickening span for the given material side: positive grows along the
// sketch normal, negative against it, symmetric splits the thickness across the plane.
func faceSpan(t float64, dir ExtentDirection) span {
	switch dir {
	case NegativeDir:
		return span{near: -t, far: 0}
	case SymmetricDir:
		return span{near: -t / 2, far: t / 2}
	default:
		return span{near: 0, far: t}
	}
}

// SheetMetalFaceFeatures adds Face features into the engine.
type SheetMetalFaceFeatures struct{ engine *PartFeatures }

// NewSheetMetalFaceFeatures binds the collection to a feature engine.
func NewSheetMetalFaceFeatures(engine *PartFeatures) *SheetMetalFaceFeatures {
	return &SheetMetalFaceFeatures{engine}
}

// Add appends a Face feature, naming it Face1, Face2, … .
func (c *SheetMetalFaceFeatures) Add(def *SheetMetalFaceDefinition) *PartFeature {
	f := &SheetMetalFaceFeature{def: def}
	pf := c.engine.Add(f)
	pf.SetName(c.engine.UniqueName("Face"))
	f.featName = pf.Name()
	return pf
}

// assert it satisfies the topo-building Feature contract.
var _ Feature = (*SheetMetalFaceFeature)(nil)
