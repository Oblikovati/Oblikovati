// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
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
type SheetMetalCornerDefinition struct {
	EdgeKeys  [][]byte
	Treatment CornerTreatment
	Size      func() float64
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
	size := evalFloat(f.def.Size)
	if size <= 0 {
		return Output{}, fmt.Errorf("sheet-metal corner: size must be positive, got %g", size)
	}
	if len(f.def.EdgeKeys) == 0 {
		return Output{}, fmt.Errorf("sheet-metal corner: no corner edges selected")
	}
	if f.def.Treatment == CornerRound {
		return f.roundCorners(in, body, size)
	}
	return f.chamferCorners(in, body, size)
}

// roundCorners rolls a fillet of the given radius around the corner edges.
func (f *SheetMetalCornerFeature) roundCorners(in Input, body *topo.Body, radius float64) (Output, error) {
	// Heal the keys before the kernel pass (ops.FilletEdges re-resolves by exact key), so a
	// recovered reference is addressed by its live key and the heal reaches the Output (P6).
	edges, heals, err := resolveEdges(body, f.def.EdgeKeys, nil)
	if err != nil {
		return Output{}, err
	}
	res, err := ops.FilletEdges(body, currentKeys(edges), radius)
	if err != nil {
		return Output{}, fmt.Errorf("sheet-metal corner round: %w", err)
	}
	return Output{Bodies: replaceBody(in.Bodies, body, res), Heals: heals}, nil
}

// chamferCorners cuts a flat bevel of the given setback across the corner edges.
func (f *SheetMetalCornerFeature) chamferCorners(in Input, body *topo.Body, setback float64) (Output, error) {
	edges, heals, err := resolveEdges(body, f.def.EdgeKeys, nil)
	if err != nil {
		return Output{}, err
	}
	work, edges := planarizeForEdges(body, edges, f.featName)
	tools, err := chamferWedges(edges, setback, setback, f.featName)
	if err != nil {
		return Output{}, fmt.Errorf("sheet-metal corner chamfer: %w", err)
	}
	res, err := cutAll(work, tools)
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
