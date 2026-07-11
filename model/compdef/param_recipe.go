// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/model/param"
)

// Parameter recipe codec, shared by the part and assembly recipes (M39-F01, #1557).
// Both a part and an assembly own a [param.Parameters] table, persist it identically,
// and restore it the same way, so the conversion is keyed on the table itself rather
// than on either definition type. The recipe DTOs live in serialize.go (the part recipe
// declares them); these functions read and rebuild a table from those DTOs.

// parametersRecipeOf captures every parameter in creation order (a valid order to
// re-add: an expression can only reference parameters created before it).
func parametersRecipeOf(params *param.Parameters) []parameterRecipe {
	all := params.All()
	if len(all) == 0 {
		return nil
	}
	out := make([]parameterRecipe, 0, len(all))
	for _, p := range all {
		out = append(out, parameterRecipeOf(p))
	}
	return out
}

// parameterRecipeOf captures one parameter's persisted form: its identity/value (by flavor)
// plus the shared presentation/behavior state (tolerance, multi-value, formatting).
func parameterRecipeOf(p *param.Parameter) parameterRecipe {
	pr := parameterRecipe{
		Name: p.Name(), Kind: p.Kind().String(),
		Comment: p.Comment, Key: p.IsKey, Export: p.ExposedAsProperty, Precision: p.Precision,
	}
	recordParameterValue(&pr, p)
	recordParameterPresentation(&pr, p)
	return pr
}

// recordParameterValue writes the parameter's value in the spelling matching its flavor:
// text/boolean carry a literal, an editable parameter its expression, and a read-only
// (reference/derived) parameter its measured value + unit.
func recordParameterValue(pr *parameterRecipe, p *param.Parameter) {
	switch {
	case p.IsText():
		pr.ValueType, pr.Text = "text", p.Text()
	case p.IsBoolean():
		pr.ValueType, pr.Bool = "boolean", p.Bool()
	case p.Kind().Editable():
		pr.Expression = p.Expression()
	default:
		pr.Value, pr.Unit = p.Value().Value, p.Value().Unit.String()
	}
}

// recordParameterPresentation writes the non-default presentation/behavior state: the
// tolerance band, the model-value selection, the multi-value list, visibility, the display
// format, and the custom-property format. Zero/default values are omitted (the recipe is sparse).
func recordParameterPresentation(pr *parameterRecipe, p *param.Parameter) {
	if t := p.Tolerance(); t != (param.Tolerance{}) {
		pr.Tolerance = &toleranceRecipe{Type: t.Kind().String(), Upper: t.Upper, Lower: t.Lower}
	}
	if m := p.ModelValueType(); m != param.Nominal {
		pr.ModelValueType = m.String()
	}
	if p.IsMultiValue() {
		pr.ExprList, pr.AllowCustom = p.ExpressionList(), p.AllowsCustomValue()
		pr.SortedValueList = !p.CustomOrder()
	}
	pr.Hidden = !p.Visible
	pr.Renamed = p.Renamed()
	pr.DisabledActions = p.DisabledActionTypes().Names()
	if p.DisplayFormat != param.DisplayFormatDecimal {
		pr.DisplayFormat = p.DisplayFormat.String()
	}
	if cp := p.CustomProperty; cp != param.DefaultCustomPropertyFormat() {
		pr.CustomProperty = &customPropertyRecipe{
			PropertyType: cp.PropertyType.String(), Units: cp.Units, Precision: cp.Precision.String(),
			ShowLeadingZeros: cp.ShowLeadingZeros, ShowTrailingZeros: cp.ShowTrailingZeros,
			ShowUnitsString: cp.ShowUnitsString,
		}
	}
}

// applyParametersTo re-creates the parameters (in recipe order) then the custom groups
// (whose member lists name parameters created above). A parse error (bad expression,
// duplicate name, unknown kind/unit) aborts the load rather than dropping silently.
func applyParametersTo(params *param.Parameters, groups []parameterGroupRecipe, recs []parameterRecipe) error {
	for _, pr := range recs {
		p, err := addParameterTo(params, pr)
		if err != nil {
			return fmt.Errorf("compdef: restore parameter %q: %w", pr.Name, err)
		}
		if err := applyParameterState(p, pr); err != nil {
			return fmt.Errorf("compdef: restore parameter %q: %w", pr.Name, err)
		}
	}
	for _, gr := range groups {
		if err := applyParameterGroupTo(params, gr); err != nil {
			return fmt.Errorf("compdef: restore parameter group %q: %w", gr.InternalName, err)
		}
	}
	return nil
}

// addParameterTo re-creates one parameter's value from its recipe entry, returning it so the
// shared state can be applied. Read-only parameters return nil (no editable state to set).
func addParameterTo(params *param.Parameters, pr parameterRecipe) (*param.Parameter, error) {
	switch pr.ValueType {
	case "text":
		return params.AddTextUserParameter(pr.Name, pr.Text)
	case "boolean":
		return params.AddBooleanUserParameter(pr.Name, pr.Bool)
	}
	switch pr.Kind {
	case param.UserParam.String():
		return params.AddUserParameter(pr.Name, pr.Expression)
	case param.ModelParam.String():
		return params.AddModelParameter(pr.Name, pr.Expression)
	case param.TableParam.String():
		return params.AddTableParameter(pr.Name, pr.Expression)
	case param.ReferenceParam.String():
		return addReadOnlyParameterTo(pr, params.AddReferenceParameter)
	case param.DerivedParam.String():
		return addReadOnlyParameterTo(pr, params.AddDerivedParameter)
	default:
		return nil, fmt.Errorf("unknown parameter kind %q (want user|model|table|reference|derived)", pr.Kind)
	}
}

// addReadOnlyParameterTo rebuilds a read-only parameter's measured quantity from its
// value + unit and adds it through the given collection method.
func addReadOnlyParameterTo(pr parameterRecipe, add func(string, param.Quantity) (*param.Parameter, error)) (*param.Parameter, error) {
	unit, ok := unitCategoryByName(pr.Unit)
	if !ok {
		return nil, fmt.Errorf("unknown unit %q", pr.Unit)
	}
	return add(pr.Name, param.Q(pr.Value, unit))
}

// applyParameterState restores the shared presentation/behavior fields onto a freshly
// re-added parameter: comment, key, export, precision, visibility, display format,
// tolerance, multi-value list, custom-property format.
func applyParameterState(p *param.Parameter, pr parameterRecipe) error {
	p.Comment, p.IsKey, p.ExposedAsProperty, p.Precision = pr.Comment, pr.Key, pr.Export, pr.Precision
	p.Visible = !pr.Hidden
	p.SetRenamed(pr.Renamed)
	if err := applyDisabledActions(p, pr); err != nil {
		return err
	}
	if err := applyParameterFormats(p, pr); err != nil {
		return err
	}
	if err := applyParameterTolerance(p, pr); err != nil {
		return err
	}
	return applyParameterValueList(p, pr)
}

// applyDisabledActions restores the restricted-action mask from its persisted
// spellings (#1853).
func applyDisabledActions(p *param.Parameter, pr parameterRecipe) error {
	if len(pr.DisabledActions) == 0 {
		return nil
	}
	mask, ok := types.ActionTypeMask(pr.DisabledActions)
	if !ok {
		return fmt.Errorf("unknown disabled action type in %v (want edit|rename|delete)", pr.DisabledActions)
	}
	p.SetDisabledActionTypes(mask)
	return nil
}

// applyParameterFormats restores the display format and custom-property format
// from their persisted wire spellings.
func applyParameterFormats(p *param.Parameter, pr parameterRecipe) error {
	if pr.DisplayFormat != "" {
		f, ok := types.ParseParameterDisplayFormat(pr.DisplayFormat)
		if !ok {
			return fmt.Errorf("unknown display format %q (want decimal|fractional|architectural)", pr.DisplayFormat)
		}
		p.DisplayFormat = f
	}
	if pr.CustomProperty == nil {
		return nil
	}
	cp, err := customPropertyFromRecipe(*pr.CustomProperty)
	if err != nil {
		return err
	}
	p.CustomProperty = cp
	return nil
}

// customPropertyFromRecipe parses one persisted custom-property format,
// defaulting absent enum spellings to the new-parameter defaults.
func customPropertyFromRecipe(cr customPropertyRecipe) (param.CustomPropertyFormat, error) {
	cp := param.DefaultCustomPropertyFormat()
	cp.Units, cp.ShowLeadingZeros, cp.ShowTrailingZeros, cp.ShowUnitsString =
		cr.Units, cr.ShowLeadingZeros, cr.ShowTrailingZeros, cr.ShowUnitsString
	if cr.PropertyType != "" {
		t, ok := types.ParseCustomPropertyType(cr.PropertyType)
		if !ok {
			return cp, fmt.Errorf("unknown custom property type %q (want text|number)", cr.PropertyType)
		}
		cp.PropertyType = t
	}
	if cr.Precision != "" {
		prec, ok := types.ParseCustomPropertyPrecision(cr.Precision)
		if !ok {
			return cp, fmt.Errorf("unknown custom property precision %q", cr.Precision)
		}
		cp.Precision = prec
	}
	return cp, nil
}

// applyParameterTolerance restores the tolerance band and the model-value
// selection from their persisted wire spellings.
func applyParameterTolerance(p *param.Parameter, pr parameterRecipe) error {
	if pr.Tolerance != nil {
		t, ok := types.ParseToleranceType(pr.Tolerance.Type)
		if !ok {
			return fmt.Errorf("unknown tolerance type %q", pr.Tolerance.Type)
		}
		p.SetTolerance(param.Tolerance{Type: t, Upper: pr.Tolerance.Upper, Lower: pr.Tolerance.Lower})
	}
	if pr.ModelValueType == "" {
		return nil
	}
	m, ok := types.ParseModelValueType(pr.ModelValueType)
	if !ok {
		return fmt.Errorf("unknown model value type %q (want nominal|lower|upper|median)", pr.ModelValueType)
	}
	return p.SetModelValueType(m)
}

// applyParameterValueList restores the multi-value choices, re-sorting when the
// list was saved as auto-sorted.
func applyParameterValueList(p *param.Parameter, pr parameterRecipe) error {
	if len(pr.ExprList) == 0 {
		return nil
	}
	if err := p.SetExpressionList(pr.ExprList, pr.AllowCustom); err != nil {
		return err
	}
	if pr.SortedValueList {
		p.SetCustomOrder(false)
	}
	return nil
}

// parameterGroupsRecipeOf renders the custom groups with their member names, in
// creation order.
func parameterGroupsRecipeOf(params *param.Parameters) []parameterGroupRecipe {
	var out []parameterGroupRecipe
	for _, g := range params.Groups() {
		gr := parameterGroupRecipe{InternalName: g.InternalName(), ClientID: g.ClientID}
		if g.DisplayName() != g.InternalName() {
			gr.DisplayName = g.DisplayName()
		}
		for _, id := range params.GroupMembers(g.InternalName()) {
			if p, ok := params.ByID(id); ok {
				gr.Members = append(gr.Members, p.Name())
			}
		}
		out = append(out, gr)
	}
	return out
}

// applyParameterGroupTo re-creates one custom group and re-attaches its members
// by name.
func applyParameterGroupTo(params *param.Parameters, gr parameterGroupRecipe) error {
	g, err := params.AddGroup(gr.InternalName, gr.DisplayName, gr.ClientID)
	if err != nil {
		return err
	}
	for _, name := range gr.Members {
		p, ok := params.ByName(name)
		if !ok {
			return fmt.Errorf("member %q is not a parameter", name)
		}
		if err := params.AddToGroup(p.ID(), g.InternalName()); err != nil {
			return err
		}
	}
	return nil
}

// parameterSettingsRecipeOf persists the parameter settings, nil when they are
// the new-document defaults (the recipe omits zero state).
func parameterSettingsRecipeOf(params *param.Parameters) *parameterSettingsRecipe {
	s := *params.Settings()
	if s == param.DefaultCollectionSettings() {
		return nil
	}
	return &parameterSettingsRecipe{
		LinearStandardTolerance:      s.LinearStandardTolerance,
		AngularStandardTolerance:     s.AngularStandardTolerance,
		UseStandardTolerances:        s.UseStandardTolerances,
		ExportStandardTolerances:     s.ExportStandardTolerances,
		LinearDimensionPrecision:     s.LinearDimensionPrecision,
		AngularDimensionPrecision:    s.AngularDimensionPrecision,
		DimensionDisplayType:         s.DimensionDisplayType.String(),
		DisplayParameterAsExpression: s.DisplayParameterAsExpression,
	}
}

// applyParameterSettingsTo restores the persisted parameter settings (nil keeps
// the defaults).
func applyParameterSettingsTo(params *param.Parameters, sr *parameterSettingsRecipe) error {
	if sr == nil {
		return nil
	}
	display, ok := types.ParseDimensionDisplayType(sr.DimensionDisplayType)
	if !ok {
		return fmt.Errorf("compdef: unknown dimension display type %q (want value|name|expression|tolerance|preciseValue)", sr.DimensionDisplayType)
	}
	*params.Settings() = param.CollectionSettings{
		LinearStandardTolerance:      sr.LinearStandardTolerance,
		AngularStandardTolerance:     sr.AngularStandardTolerance,
		UseStandardTolerances:        sr.UseStandardTolerances,
		ExportStandardTolerances:     sr.ExportStandardTolerances,
		LinearDimensionPrecision:     sr.LinearDimensionPrecision,
		AngularDimensionPrecision:    sr.AngularDimensionPrecision,
		DimensionDisplayType:         display,
		DisplayParameterAsExpression: sr.DisplayParameterAsExpression,
	}
	return nil
}

// derivedTablesRecipeOf renders the derived parameter tables in creation order.
func derivedTablesRecipeOf(params *param.Parameters) []derivedTableRecipe {
	var out []derivedTableRecipe
	for _, t := range params.DerivedTables() {
		out = append(out, derivedTableRecipe{
			ID: t.ID(), SourceDocument: t.SourceDocument(),
			Linked: t.Linked(), OwnedByFeature: t.OwnedByFeature(),
		})
	}
	return out
}

// applyDerivedTablesTo re-creates the persisted tables after the parameters they
// reconnect to.
func applyDerivedTablesTo(params *param.Parameters, tables []derivedTableRecipe) error {
	for _, tr := range tables {
		if err := params.RestoreDerivedTable(tr.ID, tr.SourceDocument, tr.Linked, tr.OwnedByFeature); err != nil {
			return fmt.Errorf("compdef: restore derived table %d: %w", tr.ID, err)
		}
	}
	return nil
}
