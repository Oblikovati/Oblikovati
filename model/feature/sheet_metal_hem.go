// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// Sheet-metal Hem feature (M13-F02, types per #1956). A hem folds the material at an edge back on
// itself — the reinforced/safe edge on a finished panel. Geometrically it is a chain of bends and
// straight runs (see sheet_metal_band.go) extruded along the picked edge and unioned onto the
// sheet. The four types are Inventor's HemTypeEnum, and two of them are radius/angle-driven rather
// than length-driven:
//
//   - single   — one half-turn, then the leg runs back over the parent (gap + length).
//   - double   — a half-turn, a leg, then a SECOND half-turn curling the other way so the free
//     edge stacks on top of the first leg instead of folding down into the parent (gap + length).
//   - rolled   — a plain curl of the given radius through the given sweep, no straight leg.
//   - teardrop — a curl past a half-turn, then a straight tail that closes back onto the sheet;
//     the tail length is derived from the radius and angle, which is why Inventor asks for
//     neither a gap nor a length here.

// HemType discriminates the hem geometry (Inventor's HemTypeEnum).
type HemType int

const (
	// SingleHem folds the material back on itself once. The gap is the clear distance between the
	// returning leg and the parent, so the inside radius is half of it; with no gap it folds tight
	// at half the material thickness. This is the value the legacy "closed"/"open" spellings map
	// to, and its ordinal is unchanged so an existing recipe still reads as a single hem.
	SingleHem HemType = iota
	// DoubleHem folds back twice, stacking three layers of material at the edge.
	DoubleHem
	// RolledHem curls the edge through a sweep angle at a given radius — a rolled safe edge.
	RolledHem
	// TeardropHem curls past a half-turn and runs back to the sheet, leaving a near-closed loop.
	TeardropHem
)

// hemFoldAngle is the half-turn a single or double hem folds through.
const hemFoldAngle = stdmath.Pi

// SheetMetalHemDefinition is the hem recipe: the edge to hem, the type, and the dimensions that
// type takes — length + gap for the folded types, radius + angle for the curled ones. Flip folds
// to the other side of the sheet.
type SheetMetalHemDefinition struct {
	EdgeKey []byte
	Length  func() float64 // how far the folded-back leg runs; single/double only
	Type    HemType
	Gap     func() float64 // clear gap of the fold (radius = gap/2); single/double only
	Radius  func() float64 // curl radius; rolled/teardrop only
	Angle   func() float64 // curl sweep (radians); rolled/teardrop only
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

// Recompute folds the hem's bend chain onto the sheet and relieves the corner it forms.
func (f *SheetMetalHemFeature) Recompute(in Input) (Output, error) {
	wall, placement, heals, err := f.foldHem(in)
	if err != nil {
		return Output{}, err
	}
	bodies, err := combine(in, wall, ops.Join)
	if err != nil {
		return Output{}, err
	}
	f.placement = &placement // record the resolved bend for the flat pattern (M13-F04)
	// A hem meeting an earlier wall at a corner forms a junction that has to be relieved too, not
	// only a flange-to-flange one (#2072): its fold and the neighbour's want the same corner
	// material. The hem spans its whole edge, so it needs no bend relief at the ends — only the
	// corner cut where it meets a bend already placed.
	bodies, err = cutCornerRelief(bodies, placement, in, in.Transition, featOr(f.featName, "hem"))
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies, Heals: heals}, nil
}

// foldHem resolves the hem's edge, live thickness and bend chain, and builds the folded wall solid
// plus the bend placement the flat pattern lays out and any reference heals from resolving the edge.
func (f *SheetMetalHemFeature) foldHem(in Input) (*topo.Body, BendPlacement, []ReferenceHeal, error) {
	body, err := lastBody(in, "sheet-metal hem")
	if err != nil {
		return nil, BendPlacement{}, nil, err
	}
	t, err := sheetThickness(in.Params)
	if err != nil {
		return nil, BendPlacement{}, nil, err
	}
	steps, err := f.hemSteps(t)
	if err != nil {
		return nil, BendPlacement{}, nil, err
	}
	edges, heals, err := resolveEdges(body, [][]byte{f.def.EdgeKey}, nil)
	if err != nil {
		return nil, BendPlacement{}, nil, err
	}
	wall, placement, err := buildFoldedSolid(edges[0], t, steps, f.def.Flip, f.featName)
	if err != nil {
		return nil, BendPlacement{}, nil, err
	}
	return wall, placement, heals, nil
}

// Placement returns the resolved bend geometry captured by the last successful recompute,
// for the flat pattern to lay this hem out as a tab. ok is false before the first recompute.
func (f *SheetMetalHemFeature) Placement() (BendPlacement, bool) {
	if f.placement == nil {
		return BendPlacement{}, false
	}
	return *f.placement, true
}

// hemSteps builds the bend/run chain for the hem's type at the live material thickness.
func (f *SheetMetalHemFeature) hemSteps(thickness float64) ([]bendRun, error) {
	switch f.def.Type {
	case SingleHem, DoubleHem:
		return f.foldedSteps(thickness)
	case RolledHem:
		r, a, err := f.curlDims()
		return []bendRun{{Angle: a, Radius: r}}, err
	case TeardropHem:
		return f.teardropSteps(thickness)
	default:
		return nil, fmt.Errorf("sheet-metal hem: unknown type %d", f.def.Type)
	}
}

// foldedSteps builds the single or double fold. The double's second bend is NEGATIVE — it curls
// the opposite way, which is what stacks its leg on top of the first instead of folding it down
// through the parent — and folds tight, since only the outer fold has to clear the doubled
// material the gap is measured across.
func (f *SheetMetalHemFeature) foldedSteps(thickness float64) ([]bendRun, error) {
	radius, leg := f.foldRadius(thickness), evalFloat(f.def.Length)
	if radius <= 0 || leg <= 0 {
		return nil, fmt.Errorf("sheet-metal hem: radius/length must be positive (r=%g l=%g)", radius, leg)
	}
	steps := []bendRun{{Angle: hemFoldAngle, Radius: radius, Run: leg}}
	if f.def.Type == DoubleHem {
		steps = append(steps, bendRun{Angle: -hemFoldAngle, Radius: thickness / 2, Run: leg})
	}
	return steps, nil
}

// teardropSteps curls the edge and then runs straight back to the parent's face. The tail is
// DERIVED: it is the run that brings the material's near surface back to the sheet, which is why
// the teardrop takes no length. A sweep of a half-turn or less never heads back, and a full turn
// has already closed, so both are refused rather than run off to infinity.
func (f *SheetMetalHemFeature) teardropSteps(thickness float64) ([]bendRun, error) {
	radius, angle, err := f.curlDims()
	if err != nil {
		return nil, err
	}
	if angle <= hemFoldAngle || angle >= 2*stdmath.Pi {
		return nil, fmt.Errorf("sheet-metal hem: a teardrop's sweep must be more than a half-turn and "+
			"less than a full one so the tail closes back onto the sheet, got %g rad", angle)
	}
	return []bendRun{{Angle: angle, Radius: radius, Run: teardropTail(radius, angle, thickness)}}, nil
}

// teardropTail is the straight run that brings the tail's end down onto the parent's face.
//
// The centreline leaves the curl at height radius − (radius+t/2)·cos(angle) and descends at
// sin(angle) per unit run. What has to reach the face is the LOWEST corner of the tail's end, and
// that sits half a thickness below the centreline only in proportion to how far the end face has
// tilted — |cos(angle)| — because the material's cross-section has rotated with it. Measuring to
// the centreline instead leaves the tail hanging half a thickness clear of the sheet, which is a
// teardrop that does not close.
func teardropTail(radius, angle, thickness float64) float64 {
	half := thickness / 2
	rise := radius - (radius+half)*stdmath.Cos(angle)
	return (half*stdmath.Abs(stdmath.Cos(angle)) - rise) / stdmath.Sin(angle)
}

// curlDims reads the radius and sweep the curled types are driven by.
func (f *SheetMetalHemFeature) curlDims() (radius, angle float64, err error) {
	radius, angle = evalFloat(f.def.Radius), evalFloat(f.def.Angle)
	if radius <= 0 || angle <= 0 {
		return 0, 0, fmt.Errorf("sheet-metal hem: a rolled or teardrop hem is driven by its curl, so "+
			"radius and angle must both be positive (r=%g a=%g rad)", radius, angle)
	}
	return radius, angle, nil
}

// foldRadius returns the inside bend radius of a single or double hem's outer fold: half the clear
// gap, or half the material thickness (folded tight) when no gap is given.
func (f *SheetMetalHemFeature) foldRadius(thickness float64) float64 {
	if g := evalFloat(f.def.Gap); g > 0 {
		return g / 2
	}
	return thickness / 2
}

// BendSpecs reports the folds a hem introduces, for the flat pattern. A folded hem's radius is
// gauge-derived (a tight fold is half the thickness), so it is always resolved here from the
// passed thickness rather than deferred to the rule's default; a curled hem's comes from its own
// radius. It reports the folds even when a dimension the SOLID needs is missing — the developed
// length depends on the bends, not on how far the leg then runs.
func (f *SheetMetalHemFeature) BendSpecs(thickness float64) []BendSpec {
	if f.def.Type == RolledHem || f.def.Type == TeardropHem {
		return []BendSpec{{Angle: evalFloat(f.def.Angle), Radius: evalFloat(f.def.Radius)}}
	}
	specs := []BendSpec{{Angle: hemFoldAngle, Radius: f.foldRadius(thickness)}}
	if f.def.Type == DoubleHem {
		specs = append(specs, BendSpec{Angle: hemFoldAngle, Radius: thickness / 2})
	}
	return specs
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

// hemTypeNames is the stable wire/recipe vocabulary for the hem types — one source shared by the
// op registry and the .obk codec so they cannot drift.
var hemTypeNames = map[HemType]string{
	SingleHem: "single", DoubleHem: "double", RolledHem: "rolled", TeardropHem: "teardrop",
}

// HemTypeName renders a hem type as its stable name.
func HemTypeName(t HemType) string { return hemTypeNames[t] }

// ParseHemType resolves a wire spelling to a hem type. "closed" and "open" are the spellings this
// feature shipped with, before the types were named after Inventor's: both are single hems, and
// what set them apart — how tight the fold is — is the GAP, which is still what says it.
func ParseHemType(s string) (HemType, bool) {
	switch s {
	case "", "closed", "open":
		return SingleHem, true
	}
	for t, n := range hemTypeNames {
		if n == s {
			return t, true
		}
	}
	return 0, false
}

var _ Feature = (*SheetMetalHemFeature)(nil)
