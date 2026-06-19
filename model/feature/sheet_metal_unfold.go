// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Sheet-metal Unfold/Refold geometry (M13-F04, #377). Unfold flattens a folded bend by a
// rigid body transform: it splits the body at the bend line, rotates the moving (flange) side
// back into the base plane, and unions it on. Refold is the inverse rotation. Because both are
// body transforms — not rebuilds — any feature applied while flat (a cut on the developed
// flange) is part of the body topology and is carried back on refold, so cut-while-flat works
// without a separate folded↔flat entity map (the bend arc flattens approximately: cuts away
// from the bend are exact).

// BendTransform is one bend an unfold/refold acts on: the bend line, its fold frame, and the
// developed dimensions. Unfold develops the bend region flat (unrolls it about the neutral
// fibre); refold rolls it back. Baked into the feature at creation from the recorded
// [BendPlacement] + the rule's developed length, so Recompute stays self-contained (it never
// reaches back into the part's other features).
type BendTransform struct {
	LinePoint math.Point3  // a point on the inner-edge bend line
	LineDir   math.Vector3 // the bend line direction (the picked edge)
	Up        math.Vector3 // fold normal — the bend-axis centre is the bend line + Up·Radius
	Out       math.Vector3 // in-plane direction toward the flange
	Angle     float64      // swept fold angle (radians)
	Radius    float64      // inside bend radius (cm)
	Thickness float64      // material gauge (cm)
	Neutral   float64      // neutral-fibre radius (cm) — the developed length per radian of bend
}

// unfoldBend develops one bend: it applies the development point map to the whole body via
// [ops.DeformBody], so any cut placed while flat moves with its vertices through the bend. sign
// > 0 unrolls the bend flat (unfold); sign < 0 rolls it back (refold). Returns one watertight
// solid.
func unfoldBend(body *topo.Body, bt BendTransform, sign float64, what string) (*topo.Body, error) {
	dev, err := newBendDevelop(bt)
	if err != nil {
		return nil, err
	}
	fn := dev.foldedToFlat
	if sign < 0 {
		fn = dev.flatToFolded
	}
	out, err := ops.DeformBody(body, fn, identityLineage)
	if err != nil {
		return nil, fmt.Errorf(errCtxWrap, what, err)
	}
	return out, nil
}

// identityLineage keeps lineage unchanged under a body transform (every face keeps its
// reference key, so a cut on the flat flange survives the develop/refold map).
func identityLineage(l topo.Lineage) topo.Lineage { return l }

// applyBends develops (sign +1, unfold) or refolds (sign −1) every bend on the running body, in
// order.
func applyBends(in Input, bends []BendTransform, sign float64, what string) (Output, error) {
	body, err := lastBody(in, what)
	if err != nil {
		return Output{}, err
	}
	result := body
	for _, bt := range bends {
		result, err = unfoldBend(result, bt, sign, what)
		if err != nil {
			return Output{}, err
		}
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result)}, nil
}

// SheetMetalUnfoldDefinition lists the bends to flatten. It is baked from the part's recorded
// bend placements when the feature is created, so the feature never reaches back into the
// part's other features at recompute.
type SheetMetalUnfoldDefinition struct{ Bends []BendTransform }

// SheetMetalUnfoldFeature flattens the listed bends each recompute, leaving the part flat so
// later features (e.g. a cut) operate in developed space.
type SheetMetalUnfoldFeature struct {
	def *SheetMetalUnfoldDefinition
}

// Definition returns the unfold recipe.
func (f *SheetMetalUnfoldFeature) Definition() *SheetMetalUnfoldDefinition { return f.def }

// Kind identifies the feature for serialization and the model tree.
func (f *SheetMetalUnfoldFeature) Kind() string { return "sheet-metal-unfold" }

// Recompute flattens every listed bend (rotates each moving flange by +angle into the base
// plane).
func (f *SheetMetalUnfoldFeature) Recompute(in Input) (Output, error) {
	return applyBends(in, f.def.Bends, +1, "sheet-metal unfold")
}

// SheetMetalRefoldDefinition lists the bends to refold — the inverse of an earlier unfold.
type SheetMetalRefoldDefinition struct{ Bends []BendTransform }

// SheetMetalRefoldFeature refolds the listed bends each recompute, restoring the folded part
// (and carrying any edits made while flat, because the refold is a body transform).
type SheetMetalRefoldFeature struct {
	def *SheetMetalRefoldDefinition
}

// Definition returns the refold recipe.
func (f *SheetMetalRefoldFeature) Definition() *SheetMetalRefoldDefinition { return f.def }

// Kind identifies the feature for serialization and the model tree.
func (f *SheetMetalRefoldFeature) Kind() string { return "sheet-metal-refold" }

// Recompute refolds every listed bend (rotates each moving flange by −angle back out of the
// base plane).
func (f *SheetMetalRefoldFeature) Recompute(in Input) (Output, error) {
	return applyBends(in, f.def.Bends, -1, "sheet-metal refold")
}

// SheetMetalUnfoldFeatures and SheetMetalRefoldFeatures add the paired features into the
// engine.
type SheetMetalUnfoldFeatures struct{ engine *PartFeatures }
type SheetMetalRefoldFeatures struct{ engine *PartFeatures }

// NewSheetMetalUnfoldFeatures binds the unfold collection to a feature engine.
func NewSheetMetalUnfoldFeatures(engine *PartFeatures) *SheetMetalUnfoldFeatures {
	return &SheetMetalUnfoldFeatures{engine}
}

// NewSheetMetalRefoldFeatures binds the refold collection to a feature engine.
func NewSheetMetalRefoldFeatures(engine *PartFeatures) *SheetMetalRefoldFeatures {
	return &SheetMetalRefoldFeatures{engine}
}

// Add appends an unfold feature, naming it Unfold1, Unfold2, … .
func (c *SheetMetalUnfoldFeatures) Add(def *SheetMetalUnfoldDefinition) *PartFeature {
	f := &SheetMetalUnfoldFeature{def: def}
	pf := c.engine.Add(f)
	pf.SetName(c.engine.UniqueName("Unfold"))
	return pf
}

// Add appends a refold feature, naming it Refold1, Refold2, … .
func (c *SheetMetalRefoldFeatures) Add(def *SheetMetalRefoldDefinition) *PartFeature {
	f := &SheetMetalRefoldFeature{def: def}
	pf := c.engine.Add(f)
	pf.SetName(c.engine.UniqueName("Refold"))
	return pf
}

var (
	_ Feature = (*SheetMetalUnfoldFeature)(nil)
	_ Feature = (*SheetMetalRefoldFeature)(nil)
)
