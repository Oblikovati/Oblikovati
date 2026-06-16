// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/param"
)

// Sheet-metal Hem feature (M13-F02). A hem folds the material at an edge back on itself —
// the reinforced/safe edge on a finished panel. Geometrically it is a flange bent through
// ~180°, so it reuses the flange's cross-section band (buildFlangeSolid at a half-turn): the
// wall curls over the bend radius and runs back across the parent. The hem TYPE sets the
// bend radius: a closed hem folds tight (radius ≈ half the thickness, the wall nearly flat
// against the sheet); an open hem leaves a rounded loop of the given gap (radius = gap/2).

// HemType discriminates the hem geometry.
type HemType int

const (
	// ClosedHem folds the material flat back on itself (a tight radius ≈ t/2).
	ClosedHem HemType = iota
	// OpenHem leaves a rounded loop of the configured gap (radius = gap/2).
	OpenHem
)

// hemFoldAngle is the half-turn every hem folds through.
const hemFoldAngle = stdmath.Pi

// SheetMetalHemDefinition is the hem recipe: the edge to hem, the hem length (how far the
// folded-back wall runs), the type, an open hem's gap, and a flip to fold to the other side.
type SheetMetalHemDefinition struct {
	EdgeKey []byte
	Length  func() float64
	Type    HemType
	Gap     func() float64 // open-hem loop gap; ignored for a closed hem
	Flip    bool
}

// SheetMetalHemFeature folds a hem onto the sheet each recompute.
type SheetMetalHemFeature struct {
	def       *SheetMetalHemDefinition
	featName  string
	placement *BendPlacement // resolved bend geometry from the last recompute (for the flat pattern)
}

// Definition returns the hem recipe.
func (f *SheetMetalHemFeature) Definition() *SheetMetalHemDefinition { return f.def }

// Kind identifies the feature for serialization and the model tree.
func (f *SheetMetalHemFeature) Kind() string { return "sheet-metal-hem" }

// Recompute resolves the edge and folds a 180° hem onto the sheet at the type's radius.
func (f *SheetMetalHemFeature) Recompute(in Input) (Output, error) {
	body, err := lastBody(in, "sheet-metal hem")
	if err != nil {
		return Output{}, err
	}
	t, radius, length, err := f.hemDims(in.Params)
	if err != nil {
		return Output{}, err
	}
	edges, err := resolveEdges(body, [][]byte{f.def.EdgeKey})
	if err != nil {
		return Output{}, err
	}
	wall, placement, err := buildFlangeSolid(edges[0], t, radius, length, hemFoldAngle, f.def.Flip, f.featName)
	if err != nil {
		return Output{}, err
	}
	bodies, err := combine(in.Bodies, wall, ops.Join)
	if err != nil {
		return Output{}, err
	}
	f.placement = &placement // record the resolved bend for the flat pattern (M13-F04)
	return Output{Bodies: bodies}, nil
}

// Placement returns the resolved bend geometry captured by the last successful recompute,
// for the flat pattern to lay this hem out as a tab. ok is false before the first recompute.
func (f *SheetMetalHemFeature) Placement() (BendPlacement, bool) {
	if f.placement == nil {
		return BendPlacement{}, false
	}
	return *f.placement, true
}

// hemDims reads the live thickness and resolves the hem's bend radius and fold-back length,
// erroring if either is non-positive.
func (f *SheetMetalHemFeature) hemDims(ps *param.Parameters) (thickness, radius, length float64, err error) {
	thickness, err = sheetThickness(ps)
	if err != nil {
		return 0, 0, 0, err
	}
	radius = f.hemRadius(thickness)
	length = evalFloat(f.def.Length)
	if radius <= 0 || length <= 0 {
		return 0, 0, 0, fmt.Errorf("sheet-metal hem: radius/length must be positive (r=%g l=%g)", radius, length)
	}
	return thickness, radius, length, nil
}

// hemRadius returns the inside bend radius for the hem type: a closed hem folds at half the
// material thickness; an open hem at half its gap (defaulting to the thickness when unset).
func (f *SheetMetalHemFeature) hemRadius(thickness float64) float64 {
	if f.def.Type == OpenHem {
		if g := evalFloat(f.def.Gap); g > 0 {
			return g / 2
		}
		return thickness
	}
	return thickness / 2
}

// BendSpecs reports the single 180° fold a hem introduces, for the flat pattern. The hem
// radius is gauge-derived (a closed hem folds at half the thickness), so it is always
// resolved here from the passed thickness rather than deferred to the rule's default.
func (f *SheetMetalHemFeature) BendSpecs(thickness float64) []BendSpec {
	return []BendSpec{{Angle: hemFoldAngle, Radius: f.hemRadius(thickness)}}
}

// SheetMetalHemFeatures adds hem features into the engine.
type SheetMetalHemFeatures struct{ engine *PartFeatures }

// NewSheetMetalHemFeatures binds the collection to a feature engine.
func NewSheetMetalHemFeatures(engine *PartFeatures) *SheetMetalHemFeatures {
	return &SheetMetalHemFeatures{engine}
}

// Add appends a hem feature, naming it Hem1, Hem2, … .
func (c *SheetMetalHemFeatures) Add(def *SheetMetalHemDefinition) *PartFeature {
	f := &SheetMetalHemFeature{def: def}
	pf := c.engine.Add(f)
	pf.SetName(c.engine.UniqueName("Hem"))
	f.featName = pf.Name()
	return pf
}

// ParseHemType resolves a wire spelling to a hem type.
func ParseHemType(s string) (HemType, bool) {
	switch s {
	case "", "closed":
		return ClosedHem, true
	case "open":
		return OpenHem, true
	}
	return 0, false
}

var _ Feature = (*SheetMetalHemFeature)(nil)
