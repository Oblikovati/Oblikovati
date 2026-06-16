// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/geom"
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

// BendTransform is one bend an unfold/refold acts on: the bend line (a point + direction) and
// the swept fold angle. The flange feature folds the moving side by −angle about the line, so
// unfold rotates it by +angle and refold by −angle. Baked into the feature at creation from
// the recorded [BendPlacement] so Recompute stays self-contained (it never reaches back into
// the part's other features).
type BendTransform struct {
	LinePoint  math.Point3  // a point on the bend line
	LineDir    math.Vector3 // the bend line direction (the picked edge)
	BaseNormal math.Vector3 // the base sheet's outward normal (defines the split plane)
	Angle      float64      // swept fold angle (radians)
}

// unfoldBend flattens one bend: split at the bend-line plane, rotate the moving side by the
// signed angle about the bend line, union back. A positive angle unfolds a +angle-folded
// flange; refold passes the negated angle. Returns one watertight solid, erroring when the
// bend line does not divide the body in two.
func unfoldBend(body *topo.Body, bt BendTransform, signedAngle float64) (*topo.Body, error) {
	dir, err := math.UnitVector3FromVector(bt.LineDir)
	if err != nil {
		return nil, fmt.Errorf("sheet-metal unfold: bend line direction is degenerate: %v", bt.LineDir)
	}
	across, err := math.UnitVector3FromVector(bt.LineDir.Cross(bt.BaseNormal))
	if err != nil {
		return nil, fmt.Errorf("sheet-metal unfold: bend line is parallel to the base normal")
	}
	fixed, moving, err := splitAtBendLine(body, bt.LinePoint, across)
	if err != nil {
		return nil, err
	}
	rotated, err := ops.TransformBody(moving, math.Rotation4(signedAngle, dir, bt.LinePoint), identityLineage)
	if err != nil {
		return nil, fmt.Errorf("sheet-metal unfold: rotating the flange: %w", err)
	}
	merged, err := combine([]*topo.Body{fixed}, rotated, ops.Join)
	if err != nil {
		return nil, fmt.Errorf("sheet-metal unfold: rejoining the flange: %w", err)
	}
	if len(merged) != 1 {
		return nil, fmt.Errorf("sheet-metal unfold: flange rejoin produced %d bodies, want 1", len(merged))
	}
	return merged[0], nil
}

// splitAtBendLine cuts the body with the plane through the bend line whose normal is across
// (perpendicular to the line, in the base plane), returning the fixed side (base) and the
// moving side (the flange, on the +across side). It errors unless the plane divides the body.
func splitAtBendLine(body *topo.Body, linePoint math.Point3, across math.UnitVector3) (fixed, moving *topo.Body, err error) {
	plane, err := geom.NewPlane(linePoint, across.AsVector())
	if err != nil {
		return nil, nil, err
	}
	pieces, err := ops.SplitSolidByPlane(body, plane)
	if err != nil {
		return nil, nil, fmt.Errorf("sheet-metal unfold: splitting at the bend line: %w", err)
	}
	if len(pieces) != 2 {
		return nil, nil, fmt.Errorf("sheet-metal unfold: bend line does not divide the body (got %d pieces)", len(pieces))
	}
	at := across.AsVector().Dot(linePoint.AsVector())
	if across.AsVector().Dot(pieces[0].RangeBox().Center().AsVector()) > at {
		return pieces[1], pieces[0], nil
	}
	return pieces[0], pieces[1], nil
}

// identityLineage keeps lineage unchanged under a body transform (the flange's faces keep
// their reference keys, so a cut on the flat flange survives the refold transform).
func identityLineage(l topo.Lineage) topo.Lineage { return l }

// applyBends flattens (sign +1) or refolds (sign −1) every bend on the running body, in order.
func applyBends(in Input, bends []BendTransform, sign float64, what string) (Output, error) {
	body, err := lastBody(in, what)
	if err != nil {
		return Output{}, err
	}
	result := body
	for _, bt := range bends {
		result, err = unfoldBend(result, bt, sign*bt.Angle)
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
