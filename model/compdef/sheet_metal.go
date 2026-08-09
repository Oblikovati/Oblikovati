// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"encoding/hex"
	"fmt"

	"oblikovati.org/api/types"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sheetmetal"
)

// Sheet-metal environment attachment (M13-F01). A sheet-metal part is a normal part whose
// content carries a [sheetmetal.Rule] and exposes the Thickness/BendRadius parameters the
// rule reads, so a parameter edit repropagates to every wall and bend.

// Default sheet-metal rule seed (database units, cm): 1 mm material, 1 mm inside bend radius
// — a common light-gauge starting point the user then restyles.
const (
	defaultSheetThickness  = 0.1 // cm (1 mm)
	defaultSheetBendRadius = 0.1 // cm (1 mm)
)

// thicknessParamName and bendRadiusParamName are the well-known parameter names the rule
// binds to; they appear in the parameter table so the user can drive them by expression.
const (
	thicknessParamName  = "Thickness"
	bendRadiusParamName = "BendRadius"
)

// IsSheetMetal reports whether this part is in the sheet-metal environment.
func (d *PartComponentDefinition) IsSheetMetal() bool { return d.sheetMetal != nil }

// SheetMetal returns the active sheet-metal rule, or nil when the part is not sheet metal.
func (d *PartComponentDefinition) SheetMetal() *sheetmetal.Rule { return d.sheetMetal }

// EnableSheetMetal puts the part into the sheet-metal environment with a default rule,
// creating the Thickness and BendRadius parameters that back it (idempotent: a part already
// in the environment keeps its rule). The rule's length closures read those parameters, so
// editing Thickness recomputes every dependent wall through the normal feature engine.
func (d *PartComponentDefinition) EnableSheetMetal() (*sheetmetal.Rule, error) {
	if d.sheetMetal != nil {
		return d.sheetMetal, nil
	}
	thickness, err := d.ensureLengthParam(thicknessParamName, defaultSheetThickness)
	if err != nil {
		return nil, err
	}
	bendRadius, err := d.ensureLengthParam(bendRadiusParamName, defaultSheetBendRadius)
	if err != nil {
		return nil, err
	}
	// The seeded rule mirrors Inventor's own Default style (#1960): a STRAIGHT bend relief at half
	// thickness, the corner trimmed to the bend at four times thickness, and a three-bend corner
	// rounded at the bend radius. The sizes are closures over the backing parameters, so a gauge
	// edit moves the reliefs with every wall.
	relief := sheetmetal.Relief{
		Shape: types.ReliefStraight,
		Width: func() float64 { return 0.5 * thickness.ModelValue() },
		Depth: func() float64 { return 0.5 * thickness.ModelValue() },
	}
	d.sheetMetal = sheetmetal.NewRule(
		"Default",
		func() float64 { return thickness.ModelValue() },
		func() float64 { return bendRadius.ModelValue() },
		sheetmetal.Constant(0),
		relief,
		sheetmetal.KFactorMethod(0.44),
	)
	d.sheetMetal.SetCornerRelief(defaultCornerRelief(thickness.ModelValue, bendRadius.ModelValue))
	d.flatOrientations = sheetmetal.NewOrientations()
	return d.sheetMetal, nil
}

// defaultCornerRelief is Inventor's Default-style corner relief (#1960): the corner trimmed to the
// bend at four times thickness, on the bend tangents, and a three-bend corner rounded at the bend
// radius. The sizes read the backing parameters, so a gauge edit moves them with every wall.
func defaultCornerRelief(thickness, bendRadius func() float64) sheetmetal.CornerRelief {
	return sheetmetal.CornerRelief{
		Shape:          types.CornerTrimToBend,
		Size:           func() float64 { return 4 * thickness() },
		Placement:      types.CornerReliefAtBendTangent,
		ThreeBendShape: types.CornerRoundWithRadius,
		ThreeBendSize:  bendRadius,
	}
}

// FlatOrientations returns the part's flat-pattern orientations (M13-F05), or nil when the
// part is not in the sheet-metal environment.
func (d *PartComponentDefinition) FlatOrientations() *sheetmetal.Orientations {
	return d.flatOrientations
}

// SetSheetMetalLengthParam re-authors one of the rule's backing length parameters
// (Thickness or BendRadius) from a unit expression, keeping the rule parameter-backed. The
// caller recomputes afterwards.
func (d *PartComponentDefinition) SetSheetMetalLengthParam(name, expr string) error {
	p, ok := d.params.ByName(name)
	if !ok {
		return fmt.Errorf("sheet-metal: no %q parameter on this part", name)
	}
	if err := d.params.SetExpression(p.ID(), expr); err != nil {
		return fmt.Errorf("sheet-metal %s = %q: %w", name, expr, err)
	}
	return nil
}

// ThicknessParamName and BendRadiusParamName expose the well-known parameter names so the
// router can address them in setStyle.
func ThicknessParamName() string  { return thicknessParamName }
func BendRadiusParamName() string { return bendRadiusParamName }

// sheetMetalRecipe is the persisted form of the rule: the parameter-backed lengths
// (thickness/bend-radius) round-trip as ordinary parameters, so the recipe carries only the
// rule's own state — its name, relief geometry, gap, and the full unfold method (K-factor,
// equation source, or bend-table rows).
type sheetMetalRecipe struct {
	Name        string  `yaml:"name,omitempty"`
	ReliefShape string  `yaml:"reliefShape,omitempty"`
	ReliefWidth float64 `yaml:"reliefWidth,omitempty"`
	ReliefDepth float64 `yaml:"reliefDepth,omitempty"`
	Gap         float64 `yaml:"gap,omitempty"`
	// The corner-relief block (#1960) — separate from the bend relief above.
	CornerReliefShape     string               `yaml:"cornerReliefShape,omitempty"`
	CornerReliefSize      float64              `yaml:"cornerReliefSize,omitempty"`
	CornerReliefPlacement string               `yaml:"cornerReliefPlacement,omitempty"`
	ThreeBendReliefShape  string               `yaml:"threeBendReliefShape,omitempty"`
	ThreeBendReliefSize   float64              `yaml:"threeBendReliefSize,omitempty"`
	UnfoldMethod          string               `yaml:"unfoldMethod,omitempty"`
	KFactor               float64              `yaml:"kFactor,omitempty"`
	Equation              string               `yaml:"equation,omitempty"`
	BendTable             []bendTableRowRecipe `yaml:"bendTable,omitempty"`
	Orientations          []orientationRecipe  `yaml:"orientations,omitempty"` // M13-F05
	ActiveOrient          string               `yaml:"activeOrientation,omitempty"`
	DeferUpdate           bool                 `yaml:"deferFlatUpdate,omitempty"` // M13-F05
	BendOrder             []string             `yaml:"bendOrder,omitempty"`       // M13-F06
	Centerlines           []centerlineRecipe   `yaml:"centerlines,omitempty"`     // M13-F06
}

// centerlineRecipe is one persisted cosmetic centerline (flat 2D coordinates, cm).
type centerlineRecipe struct {
	X1 float64 `yaml:"x1"`
	Y1 float64 `yaml:"y1"`
	X2 float64 `yaml:"x2"`
	Y2 float64 `yaml:"y2"`
}

// bendTableRowRecipe is one persisted bend-table sample (database units, cm; angle radians).
type bendTableRowRecipe struct {
	Angle     float64 `yaml:"angle"`
	Radius    float64 `yaml:"radius"`
	Thickness float64 `yaml:"thickness"`
	Allowance float64 `yaml:"allowance"`
}

// orientationRecipe is one persisted flat-pattern orientation (M13-F05; rotation in radians,
// alignment axis as a hex reference key).
type orientationRecipe struct {
	Name              string  `yaml:"name"`
	AlignmentType     string  `yaml:"alignmentType,omitempty"`
	AlignmentRotation float64 `yaml:"alignmentRotation,omitempty"`
	AlignmentAxis     string  `yaml:"alignmentAxis,omitempty"`
	FlipAlignmentAxis bool    `yaml:"flipAlignmentAxis,omitempty"`
	FlipBaseFace      bool    `yaml:"flipBaseFace,omitempty"`
}

// sheetMetalRecipeOf captures the active rule, or nil when the part is not sheet metal.
func (d *PartComponentDefinition) sheetMetalRecipeOf() *sheetMetalRecipe {
	if d.sheetMetal == nil {
		return nil
	}
	r := d.sheetMetal
	rec := &sheetMetalRecipe{
		Name:                  r.Name(),
		ReliefShape:           r.Relief().Shape.String(),
		ReliefWidth:           r.ReliefWidth(),
		ReliefDepth:           r.ReliefDepth(),
		Gap:                   r.Gap(),
		CornerReliefShape:     r.CornerRelief().Shape.String(),
		CornerReliefSize:      r.CornerReliefSize(),
		CornerReliefPlacement: r.CornerRelief().Placement.String(),
		ThreeBendReliefShape:  r.CornerRelief().ThreeBendShape.String(),
		ThreeBendReliefSize:   r.ThreeBendReliefSize(),
		UnfoldMethod:          r.Unfold().Type.String(),
		KFactor:               r.Unfold().KFactor,
		Equation:              r.Unfold().EquationSource(),
	}
	if t := r.Unfold().Table; t != nil {
		for _, row := range t.Rows() {
			rec.BendTable = append(rec.BendTable, bendTableRowRecipe{row.Angle, row.Radius, row.Thickness, row.Allowance})
		}
	}
	d.captureOrientations(rec)
	return rec
}

// captureOrientations records the flat-pattern orientations (and which is active) into rec.
func (d *PartComponentDefinition) captureOrientations(rec *sheetMetalRecipe) {
	if d.flatOrientations == nil {
		return
	}
	for _, o := range d.flatOrientations.List() {
		rec.Orientations = append(rec.Orientations, orientationRecipe{
			Name: o.Name, AlignmentType: o.AlignmentType.String(), AlignmentRotation: o.AlignmentRotation,
			AlignmentAxis: hex.EncodeToString(o.AlignmentAxisKey), FlipAlignmentAxis: o.FlipAlignmentAxis, FlipBaseFace: o.FlipBaseFace,
		})
	}
	rec.ActiveOrient = d.flatOrientations.Active().Name
	rec.DeferUpdate = d.flatSettings.DeferUpdate
	rec.BendOrder = d.bendOrder
	for _, c := range d.centerlines {
		rec.Centerlines = append(rec.Centerlines, centerlineRecipe{c.Start.X, c.Start.Y, c.End.X, c.End.Y})
	}
}

// applySheetMetalRecipe rebuilds the rule from rec, binding its thickness/bend-radius to the
// already-restored Thickness/BendRadius parameters. Called after parameters are applied.
func (d *PartComponentDefinition) applySheetMetalRecipe(rec *sheetMetalRecipe) error {
	if rec == nil {
		return nil
	}
	rule, err := d.EnableSheetMetal() // creates/reuses the backing parameters, seeds defaults
	if err != nil {
		return err
	}
	rule.SetName(rec.Name)
	if err := d.restoreReliefAndGap(rule, rec); err != nil {
		return err
	}
	unfold, err := unfoldFromRecipe(rec)
	if err != nil {
		return err
	}
	rule.SetUnfold(unfold)
	return d.restoreOrientations(rec)
}

// restoreOrientations rebuilds the flat-pattern orientations from rec onto the seeded set
// (the default orientation already exists), then re-activates the persisted active one.
func (d *PartComponentDefinition) restoreOrientations(rec *sheetMetalRecipe) error {
	for _, o := range rec.Orientations {
		at, _ := types.ParseAlignmentType(o.AlignmentType)
		key, err := hex.DecodeString(o.AlignmentAxis)
		if err != nil {
			return fmt.Errorf("sheet-metal recipe: bad alignment axis key %q: %w", o.AlignmentAxis, err)
		}
		restored := &sheetmetal.FlatPatternOrientation{
			Name: o.Name, AlignmentType: at, AlignmentRotation: o.AlignmentRotation,
			AlignmentAxisKey: key, FlipAlignmentAxis: o.FlipAlignmentAxis, FlipBaseFace: o.FlipBaseFace,
		}
		if existing, _, ok := d.flatOrientations.ByName(o.Name); ok {
			*existing = *restored // the seeded default already exists — restore its edited fields
			continue
		}
		if err := d.flatOrientations.Add(restored); err != nil {
			return err
		}
	}
	if rec.ActiveOrient != "" {
		if err := d.flatOrientations.Activate(rec.ActiveOrient); err != nil {
			return err
		}
	}
	d.flatSettings.DeferUpdate = rec.DeferUpdate
	d.bendOrder = rec.BendOrder
	d.centerlines = nil
	for _, c := range rec.Centerlines {
		d.centerlines = append(d.centerlines, sheetmetal.CosmeticCenterline{Start: gmath.P2(c.X1, c.Y1), End: gmath.P2(c.X2, c.Y2)})
	}
	return nil
}

// restoreReliefAndGap applies the persisted relief shape/size and gap onto the rule.
func (d *PartComponentDefinition) restoreReliefAndGap(rule *sheetmetal.Rule, rec *sheetMetalRecipe) error {
	shape := types.ReliefStraight
	if rec.ReliefShape != "" {
		s, ok := types.ParseReliefShape(rec.ReliefShape)
		if !ok {
			return fmt.Errorf("sheet-metal recipe: bad relief shape %q", rec.ReliefShape)
		}
		shape = s
	}
	rule.SetRelief(sheetmetal.Relief{
		Shape: shape,
		Width: sheetmetal.Constant(rec.ReliefWidth),
		Depth: sheetmetal.Constant(rec.ReliefDepth),
	})
	rule.SetGap(sheetmetal.Constant(rec.Gap))
	return d.restoreCornerRelief(rule, rec)
}

// restoreCornerRelief applies the persisted corner-relief block (#1960). An absent block keeps the
// rule's defaults, which are Inventor's; an unrecognised name is an error rather than a silent
// fallback to a different cut.
func (d *PartComponentDefinition) restoreCornerRelief(rule *sheetmetal.Rule, rec *sheetMetalRecipe) error {
	corner := rule.CornerRelief()
	for _, e := range []struct {
		name  string
		field *types.CornerReliefShape
	}{{rec.CornerReliefShape, &corner.Shape}, {rec.ThreeBendReliefShape, &corner.ThreeBendShape}} {
		if e.name == "" {
			continue
		}
		shape, ok := types.ParseCornerReliefShape(e.name)
		if !ok {
			return fmt.Errorf("sheet-metal recipe: bad corner relief shape %q", e.name)
		}
		*e.field = shape
	}
	if rec.CornerReliefPlacement != "" {
		placement, ok := types.ParseCornerReliefPlacement(rec.CornerReliefPlacement)
		if !ok {
			return fmt.Errorf("sheet-metal recipe: bad corner relief placement %q", rec.CornerReliefPlacement)
		}
		corner.Placement = placement
	}
	if rec.CornerReliefSize != 0 {
		corner.Size = sheetmetal.Constant(rec.CornerReliefSize)
	}
	if rec.ThreeBendReliefSize != 0 {
		corner.ThreeBendSize = sheetmetal.Constant(rec.ThreeBendReliefSize)
	}
	rule.SetCornerRelief(corner)
	return nil
}

// unfoldFromRecipe reconstructs the unfold method (K-factor / bend-table / equation).
func unfoldFromRecipe(rec *sheetMetalRecipe) (sheetmetal.UnfoldMethod, error) {
	method, ok := types.ParseUnfoldMethodType(rec.UnfoldMethod)
	if !ok && rec.UnfoldMethod != "" {
		return sheetmetal.UnfoldMethod{}, fmt.Errorf("sheet-metal recipe: bad unfold method %q", rec.UnfoldMethod)
	}
	switch method {
	case types.BendTableUnfold:
		rows := make([]sheetmetal.BendTableRow, len(rec.BendTable))
		for i, r := range rec.BendTable {
			rows[i] = sheetmetal.BendTableRow{Angle: r.Angle, Radius: r.Radius, Thickness: r.Thickness, Allowance: r.Allowance}
		}
		return sheetmetal.BendTableMethod(sheetmetal.NewBendTable(rows)), nil
	case types.EquationUnfold:
		return sheetmetal.EquationMethod(rec.Equation)
	default:
		return sheetmetal.KFactorMethod(rec.KFactor), nil
	}
}

// ensureLengthParam returns the named length parameter, creating it from a default value
// (in database units) when absent — so re-enabling or restoring a sheet-metal part reuses
// the existing parameter rather than failing on a duplicate name.
func (d *PartComponentDefinition) ensureLengthParam(name string, defaultValue float64) (*param.Parameter, error) {
	if p, ok := d.params.ByName(name); ok {
		return p, nil
	}
	expr := d.units.Format(param.Q(defaultValue, param.Length))
	p, err := d.params.AddUserParameter(name, expr)
	if err != nil {
		return nil, fmt.Errorf("sheet-metal %s parameter (%q): %w", name, expr, err)
	}
	return p, nil
}
