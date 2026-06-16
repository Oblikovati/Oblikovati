// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"fmt"

	"oblikovati.org/api/types"
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
	relief := sheetmetal.Relief{
		Shape: types.ReliefRound,
		Width: sheetmetal.Constant(0.5 * thickness.ModelValue()),
		Depth: sheetmetal.Constant(0.5 * thickness.ModelValue()),
	}
	d.sheetMetal = sheetmetal.NewRule(
		"Default",
		func() float64 { return thickness.ModelValue() },
		func() float64 { return bendRadius.ModelValue() },
		sheetmetal.Constant(0),
		relief,
		sheetmetal.KFactorMethod(0.44),
	)
	return d.sheetMetal, nil
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
	Name         string               `yaml:"name,omitempty"`
	ReliefShape  string               `yaml:"reliefShape,omitempty"`
	ReliefWidth  float64              `yaml:"reliefWidth,omitempty"`
	ReliefDepth  float64              `yaml:"reliefDepth,omitempty"`
	Gap          float64              `yaml:"gap,omitempty"`
	UnfoldMethod string               `yaml:"unfoldMethod,omitempty"`
	KFactor      float64              `yaml:"kFactor,omitempty"`
	Equation     string               `yaml:"equation,omitempty"`
	BendTable    []bendTableRowRecipe `yaml:"bendTable,omitempty"`
}

// bendTableRowRecipe is one persisted bend-table sample (database units, cm; angle radians).
type bendTableRowRecipe struct {
	Angle     float64 `yaml:"angle"`
	Radius    float64 `yaml:"radius"`
	Thickness float64 `yaml:"thickness"`
	Allowance float64 `yaml:"allowance"`
}

// sheetMetalRecipeOf captures the active rule, or nil when the part is not sheet metal.
func (d *PartComponentDefinition) sheetMetalRecipeOf() *sheetMetalRecipe {
	if d.sheetMetal == nil {
		return nil
	}
	r := d.sheetMetal
	rec := &sheetMetalRecipe{
		Name:         r.Name(),
		ReliefShape:  r.Relief().Shape.String(),
		ReliefWidth:  r.ReliefWidth(),
		ReliefDepth:  r.ReliefDepth(),
		Gap:          r.Gap(),
		UnfoldMethod: r.Unfold().Type.String(),
		KFactor:      r.Unfold().KFactor,
		Equation:     r.Unfold().EquationSource(),
	}
	if t := r.Unfold().Table; t != nil {
		for _, row := range t.Rows() {
			rec.BendTable = append(rec.BendTable, bendTableRowRecipe{row.Angle, row.Radius, row.Thickness, row.Allowance})
		}
	}
	return rec
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
	return nil
}

// restoreReliefAndGap applies the persisted relief shape/size and gap onto the rule.
func (d *PartComponentDefinition) restoreReliefAndGap(rule *sheetmetal.Rule, rec *sheetMetalRecipe) error {
	shape := types.ReliefRound
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
