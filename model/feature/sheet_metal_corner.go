// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/topo"
)

// Sheet-metal Corner feature (M13-F02). A corner treatment chamfers or rounds the corner of a
// sheet-metal face — the through-thickness edge where two boundary edges of the wall meet.
// CornerChamfer cuts a flat across the corner; CornerRound rolls a fillet. Both reuse the
// part-modeling chamfer/fillet geometry (a corner is just that vertical edge), wrapped as a
// sheet-metal feature so it lives in the sheet-metal history and the flat pattern (F04).

// CornerTreatment discriminates how a sheet-metal corner is finished.
type CornerTreatment int

const (
	// CornerChamfer cuts a flat bevel across the corner (a triangular notch through the wall).
	CornerChamfer CornerTreatment = iota
	// CornerRound rolls a fillet around the corner.
	CornerRound
)

// ParseCornerTreatment resolves a wire spelling to a corner treatment.
func ParseCornerTreatment(s string) (CornerTreatment, bool) {
	switch s {
	case "chamfer":
		return CornerChamfer, true
	case "round":
		return CornerRound, true
	}
	return 0, false
}

// SheetMetalCornerDefinition is the corner recipe: the corner edges (through-thickness edge
// reference keys), the treatment, and its size (chamfer setback / round radius).
//
// A chamfer takes its variant from ChamferType (#1967): the single Size (equal setbacks), Size plus
// Angle (distance-and-angle), or Size plus Distance2 (two distances). A round may instead carry
// RoundSets — several edge groups each with its own radius, one feature — in which case EdgeKeys/Size
// are unused; a single-size round leaves RoundSets empty.
type SheetMetalCornerDefinition struct {
	EdgeKeys  [][]byte
	Treatment CornerTreatment
	Size      func() float64
	// ChamferType / Distance2 / Angle / FaceKey shape the chamfer variant. Angle is in radians.
	ChamferType types.ChamferType
	Distance2   func() float64
	Angle       func() float64
	FaceKey     []byte
	// RoundSets carries a multi-radius corner round; empty ⇒ the single EdgeKeys/Size round.
	RoundSets []CornerRoundSet
}

// CornerRoundSet is one radius group of a multi-set corner round (#1967).
type CornerRoundSet struct {
	EdgeKeys [][]byte
	Radius   func() float64
}

// SheetMetalCornerFeature finishes the sheet's corners each recompute.
type SheetMetalCornerFeature struct {
	def      *SheetMetalCornerDefinition
	featName string
}

// Definition returns the corner recipe.
func (f *SheetMetalCornerFeature) Definition() *SheetMetalCornerDefinition { return f.def }

// Kind identifies the feature for serialization and the model tree.
func (f *SheetMetalCornerFeature) Kind() string { return "sheet-metal-corner" }

// Recompute resolves the corner edges and chamfers or rounds them on the running sheet.
func (f *SheetMetalCornerFeature) Recompute(in Input) (Output, error) {
	body, err := lastBody(in, "sheet-metal corner")
	if err != nil {
		return Output{}, err
	}
	if f.def.Treatment == CornerRound {
		return f.roundCorners(in, body)
	}
	return f.chamferCorners(in, body)
}

// roundCorners rolls a fillet around each round edge set at that set's radius (#1967). A single set
// takes the simple per-radius path (the analytic cylinder-rim blend); several sets round in ONE
// kernel pass with per-edge radii, so the sets do not re-lineage each other's edges between passes.
func (f *SheetMetalCornerFeature) roundCorners(in Input, body *topo.Body) (Output, error) {
	sets := f.roundSets()
	if len(sets) == 0 {
		return Output{}, fmt.Errorf("sheet-metal corner: no corner edges selected")
	}
	if len(sets) == 1 {
		res, heals, err := f.roundOneSet(body, sets[0])
		if err != nil {
			return Output{}, err
		}
		return Output{Bodies: replaceBody(in.Bodies, body, res), Heals: heals}, nil
	}
	return f.roundManySets(in, body, sets)
}

// roundManySets rounds every set's edges at its own radius in one FilletEdgesCorner pass, healing
// each pick's key first (ADR-0043 P6) so a recovered reference reaches the kernel and the Output.
func (f *SheetMetalCornerFeature) roundManySets(in Input, body *topo.Body, sets []CornerRoundSet) (Output, error) {
	picks, err := cornerRoundPicks(sets)
	if err != nil {
		return Output{}, err
	}
	heals, err := healPickKeys(body, picks)
	if err != nil {
		return Output{}, err
	}
	work := planarizeFilletPicks(body, picks, f.featName)
	res, err := blend.FilletEdgesCornerDiag(work, picks, cornerStrategy(types.FilletCornerMiter), concaveFill(0), in.Diag)
	if err != nil {
		return Output{}, fmt.Errorf("sheet-metal corner round: %w", err)
	}
	return Output{Bodies: replaceBody(in.Bodies, body, res), Heals: heals}, nil
}

// cornerRoundPicks flattens the round edge sets into per-edge radius picks, erroring on a
// non-positive radius.
func cornerRoundPicks(sets []CornerRoundSet) ([]blend.EdgeFilletRadii, error) {
	var picks []blend.EdgeFilletRadii
	for _, set := range sets {
		radius := evalFloat(set.Radius)
		if radius <= 0 {
			return nil, fmt.Errorf("sheet-metal corner round: radius must be positive, got %g", radius)
		}
		for _, k := range set.EdgeKeys {
			picks = append(picks, blend.EdgeFilletRadii{Key: k, R0: radius, R1: radius})
		}
	}
	return picks, nil
}

// roundSets returns the round's edge sets — the explicit RoundSets, or the single EdgeKeys/Size set.
func (f *SheetMetalCornerFeature) roundSets() []CornerRoundSet {
	if len(f.def.RoundSets) > 0 {
		return f.def.RoundSets
	}
	if len(f.def.EdgeKeys) == 0 {
		return nil
	}
	return []CornerRoundSet{{EdgeKeys: f.def.EdgeKeys, Radius: f.def.Size}}
}

// roundOneSet fillets one edge set on the running body at its radius. It heals the keys before the
// kernel pass (blend.FilletEdges re-resolves by exact key) so a recovered reference reaches the Output.
func (f *SheetMetalCornerFeature) roundOneSet(body *topo.Body, set CornerRoundSet) (*topo.Body, []ReferenceHeal, error) {
	radius := evalFloat(set.Radius)
	if radius <= 0 {
		return nil, nil, fmt.Errorf("sheet-metal corner round: radius must be positive, got %g", radius)
	}
	edges, heals, err := resolveEdges(body, set.EdgeKeys, nil)
	if err != nil {
		return nil, nil, err
	}
	res, err := blend.FilletEdges(body, currentKeys(edges), radius)
	if err != nil {
		return nil, nil, fmt.Errorf("sheet-metal corner round: %w", err)
	}
	return res, heals, nil
}

// chamferCorners cuts a flat bevel across the corner edges, its two setbacks taken from the chamfer
// variant — equal distance, distance-and-angle, or two distances (#1967).
func (f *SheetMetalCornerFeature) chamferCorners(in Input, body *topo.Body) (Output, error) {
	if len(f.def.EdgeKeys) == 0 {
		return Output{}, fmt.Errorf("sheet-metal corner: no corner edges selected")
	}
	d1, d2, err := chamferSetbackValues(f.def.ChamferType, evalFloat(f.def.Size), evalFloat(f.def.Distance2), evalFloat(f.def.Angle))
	if err != nil {
		return Output{}, fmt.Errorf("sheet-metal corner chamfer: %w", err)
	}
	if d1 <= 0 || d2 <= 0 {
		return Output{}, fmt.Errorf("sheet-metal corner chamfer: setbacks must be positive, got %g and %g", d1, d2)
	}
	edges, heals, err := resolveEdges(body, f.def.EdgeKeys, nil)
	if err != nil {
		return Output{}, err
	}
	work, edges := planarizeForEdges(body, edges, f.featName)
	tools, err := chamferWedges(edges, d1, d2, chamferRun{}, f.featName)
	if err != nil {
		return Output{}, fmt.Errorf("sheet-metal corner chamfer: %w", err)
	}
	res, err := cutAll(work, tools, in.Diag)
	if err != nil {
		return Output{}, fmt.Errorf("sheet-metal corner chamfer: %w", err)
	}
	return Output{Bodies: replaceBody(in.Bodies, body, res), Heals: heals}, nil
}

// SheetMetalCornerFeatures adds corner features into the engine.
type SheetMetalCornerFeatures struct{ engine *PartFeatures }

// NewSheetMetalCornerFeatures binds the collection to a feature engine.
func NewSheetMetalCornerFeatures(engine *PartFeatures) *SheetMetalCornerFeatures {
	return &SheetMetalCornerFeatures{engine}
}

// Add appends a corner feature, naming it Corner1, Corner2, … .
func (c *SheetMetalCornerFeatures) Add(def *SheetMetalCornerDefinition) *PartFeature {
	f := &SheetMetalCornerFeature{def: def}
	pf := c.engine.Add(f)
	pf.SetName(c.engine.UniqueName("Corner"))
	f.featName = pf.Name()
	return pf
}

var _ Feature = (*SheetMetalCornerFeature)(nil)
