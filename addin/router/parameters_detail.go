// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
)

// The member-level parameter surface (M02-F08, Oblikovati#607): detail reads,
// presentation/tolerance/value-list mutations, dependency queries and delete.
// Mutations finalize through Session.RecordActiveEdit so they are undoable, and
// the router broadcasts them as edit.committed (mutatingMethods).

// paramDetail marshals the full member-level view of one parameter.
func paramDetail(holder compdef.ParameterHolder, p *param.Parameter) wire.ParameterDetail {
	ps := holder.Parameters()
	d := wire.ParameterDetail{
		ParameterInfo: paramInfo(holder, p),
		Units:         paramUnitName(holder, p),
		Comment:       p.Comment, IsKey: p.IsKey, Visible: p.Visible,
		InUse: ps.InUse(p.ID()), Precision: p.Precision,
		DisplayFormat: p.DisplayFormat.String(), ExposedAsProperty: p.ExposedAsProperty,
		BuiltIn: p.BuiltIn(), Renamed: p.Renamed(),
		DisabledActionTypes:  p.DisabledActionTypes().Names(),
		ModelValueType:       p.ModelValueType().String(),
		CustomPropertyFormat: customPropertyInfo(p.CustomProperty),
		DrivenBy:             paramNames(ps, ps.DrivenBy(p.ID())),
		Dependents:           paramNames(ps, ps.Dependents(p.ID())),
	}
	if p.IsNumeric() {
		tol := p.Tolerance()
		d.ModelValue = p.ModelValue()
		d.Tolerance = &wire.ToleranceInfo{
			Type: tol.Kind().String(), Upper: tol.Upper, Lower: tol.Lower,
			HoleTolerance: tol.HoleTolerance, ShaftTolerance: tol.ShaftTolerance,
		}
	}
	if p.IsMultiValue() {
		d.ExpressionList = &wire.ExpressionListInfo{
			Expressions: p.ExpressionList(), AllowCustomValues: p.AllowsCustomValue(),
			CustomOrder: p.CustomOrder(),
		}
	}
	return d
}

// paramUnitName names the parameter's unit for the wire: the document's display
// unit for dimensioned categories, the category name (text/boolean/unitless)
// otherwise.
func paramUnitName(holder compdef.ParameterHolder, p *param.Parameter) string {
	if name := holder.Units().PreferredName(p.Unit()); name != "" {
		return name
	}
	return p.Unit().String()
}

// customPropertyInfo marshals a custom-property format into its wire DTO.
func customPropertyInfo(cp param.CustomPropertyFormat) *wire.CustomPropertyFormatInfo {
	return &wire.CustomPropertyFormatInfo{
		PropertyType: cp.PropertyType.String(), Units: cp.Units, Precision: cp.Precision.String(),
		ShowLeadingZeros: cp.ShowLeadingZeros, ShowTrailingZeros: cp.ShowTrailingZeros,
		ShowUnitsString: cp.ShowUnitsString,
	}
}

// paramNames maps graph ids onto display names (deleted ids are skipped).
func paramNames(ps *param.Parameters, ids []param.ID) []string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if p, ok := ps.ByID(id); ok {
			names = append(names, p.Name())
		}
	}
	return names
}

// paramByName resolves one parameter on the active part or assembly, naming the method in
// the not-found error.
func paramByName(s *app.Session, method, name string) (compdef.ParameterHolder, *param.Parameter, error) {
	holder, err := modelaccess.ActiveParameterHolder(s)
	if err != nil {
		return nil, nil, err
	}
	p, ok := holder.Parameters().ByName(name)
	if !ok {
		return nil, nil, fmt.Errorf("%s: no parameter named %q", method, name)
	}
	return holder, p, nil
}

// getParameterDetail returns the member-level view of one parameter.
func getParameterDetail(s *app.Session, in wire.ParameterNameArgs) (wire.ParameterDetail, error) {
	holder, p, err := paramByName(s, wire.MethodParametersGetDetail, in.Name)
	if err != nil {
		return wire.ParameterDetail{}, err
	}
	return paramDetail(holder, p), nil
}

// updateParameter applies the non-nil presentation/exposure mutations, records
// one undo step, and returns the updated detail.
func updateParameter(s *app.Session, in wire.ParameterUpdateArgs) (wire.ParameterDetail, error) {
	holder, p, err := paramByName(s, wire.MethodParametersUpdate, in.Name)
	if err != nil {
		return wire.ParameterDetail{}, err
	}
	if err := applyParameterUpdate(p, in); err != nil {
		return wire.ParameterDetail{}, err
	}
	// A model-value selection change moves the value features consume.
	if in.ModelValueType != nil {
		holder.RecomputeAfterChange()
	}
	s.RecordActiveEdit("Edit Parameter")
	return paramDetail(holder, p), nil
}

// applyParameterUpdate copies the non-nil update fields onto the parameter.
func applyParameterUpdate(p *param.Parameter, in wire.ParameterUpdateArgs) error {
	setIfPresent(in.Comment, &p.Comment)
	setIfPresent(in.IsKey, &p.IsKey)
	setIfPresent(in.Visible, &p.Visible)
	setIfPresent(in.Precision, &p.Precision)
	setIfPresent(in.ExposedAsProperty, &p.ExposedAsProperty)
	if err := applyParameterEnums(p, in); err != nil {
		return err
	}
	return applyCustomPropertyUpdate(p, in.CustomPropertyFormat)
}

// applyParameterEnums applies the update's enum-valued fields (display format,
// model-value selection, disabled-action mask), validating each spelling.
func applyParameterEnums(p *param.Parameter, in wire.ParameterUpdateArgs) error {
	if in.DisplayFormat != nil {
		f, ok := types.ParseParameterDisplayFormat(*in.DisplayFormat)
		if !ok {
			return fmt.Errorf("parameters.update: unknown display format %q (want decimal|fractional|architectural)", *in.DisplayFormat)
		}
		p.DisplayFormat = f
	}
	if in.ModelValueType != nil {
		m, ok := types.ParseModelValueType(*in.ModelValueType)
		if !ok {
			return fmt.Errorf("parameters.update: unknown model value type %q (want nominal|lower|upper|median)", *in.ModelValueType)
		}
		if err := p.SetModelValueType(m); err != nil {
			return err
		}
	}
	if in.DisabledActionTypes != nil {
		mask, ok := types.ActionTypeMask(*in.DisabledActionTypes)
		if !ok {
			return fmt.Errorf("parameters.update: unknown disabled action type in %v (want edit|rename|delete)", *in.DisabledActionTypes)
		}
		p.SetDisabledActionTypes(mask)
	}
	return nil
}

// setIfPresent copies a pointer-optional update field when it was sent.
func setIfPresent[V any](src *V, dst *V) {
	if src != nil {
		*dst = *src
	}
}

// applyCustomPropertyUpdate replaces the parameter's custom-property format
// from its wire DTO, validating the enum spellings.
func applyCustomPropertyUpdate(p *param.Parameter, in *wire.CustomPropertyFormatInfo) error {
	if in == nil {
		return nil
	}
	t, ok := types.ParseCustomPropertyType(in.PropertyType)
	if !ok {
		return fmt.Errorf("parameters.update: unknown custom property type %q (want text|number)", in.PropertyType)
	}
	prec, ok := types.ParseCustomPropertyPrecision(in.Precision)
	if !ok {
		return fmt.Errorf("parameters.update: unknown custom property precision %q", in.Precision)
	}
	p.CustomProperty = param.CustomPropertyFormat{
		PropertyType: t, Units: in.Units, Precision: prec,
		ShowLeadingZeros: in.ShowLeadingZeros, ShowTrailingZeros: in.ShowTrailingZeros,
		ShowUnitsString: in.ShowUnitsString,
	}
	return nil
}

// setParameterTolerance applies one tolerance mode, recomputes (the model value
// may move), records one undo step, and returns the updated detail.
func setParameterTolerance(s *app.Session, in wire.ParameterToleranceArgs) (wire.ParameterDetail, error) {
	holder, p, err := paramByName(s, wire.MethodParametersSetTolerance, in.Name)
	if err != nil {
		return wire.ParameterDetail{}, err
	}
	if err := applyToleranceMode(holder, p, in); err != nil {
		return wire.ParameterDetail{}, err
	}
	holder.RecomputeAfterChange()
	s.RecordActiveEdit("Edit Tolerance")
	return paramDetail(holder, p), nil
}

// applyToleranceMode dispatches one wire tolerance mode onto the model setters.
// The operand-bearing modes (symmetric/deviation/limits) parse their unit-bearing
// values here; the rest go through applyOperandlessTolerance.
func applyToleranceMode(holder compdef.ParameterHolder, p *param.Parameter, in wire.ParameterToleranceArgs) error {
	switch in.Mode {
	case "symmetric":
		band, err := toleranceOperand(holder, p, in.Upper, "upper")
		if err != nil {
			return err
		}
		return p.SetToleranceSymmetric(band)
	case "deviation", "limits":
		return applyToleranceBand(holder, p, in)
	}
	return applyOperandlessTolerance(p, in)
}

// applyOperandlessTolerance dispatches the tolerance modes that take no
// unit-bearing operands: the bandless flavors and the ISO fits/basic/reference
// modes (#1848).
func applyOperandlessTolerance(p *param.Parameter, in wire.ParameterToleranceArgs) error {
	switch in.Mode {
	case "default":
		return p.SetToleranceDefault()
	case "min":
		return p.SetToleranceMinMax(types.ToleranceMin)
	case "max":
		return p.SetToleranceMinMax(types.ToleranceMax)
	case "fits":
		return p.SetToleranceFits(in.Hole, in.Shaft)
	case "basic":
		return p.SetToleranceBasic()
	case "reference":
		return p.SetToleranceReference()
	default:
		return fmt.Errorf("parameters.setTolerance: unknown mode %q (want default|deviation|symmetric|limits|min|max|fits|basic|reference)", in.Mode)
	}
}

// applyToleranceBand parses both operands and applies a deviation or limits band.
func applyToleranceBand(holder compdef.ParameterHolder, p *param.Parameter, in wire.ParameterToleranceArgs) error {
	upper, err := toleranceOperand(holder, p, in.Upper, "upper")
	if err != nil {
		return err
	}
	lower, err := toleranceOperand(holder, p, in.Lower, "lower")
	if err != nil {
		return err
	}
	if in.Mode == "limits" {
		return p.SetToleranceLimits(upper, lower)
	}
	return p.SetToleranceDeviation(upper, lower)
}

// toleranceOperand parses one unit-bearing tolerance expression in the
// parameter's unit category, returning the database-unit value.
func toleranceOperand(holder compdef.ParameterHolder, p *param.Parameter, expr, field string) (float64, error) {
	if expr == "" {
		return 0, fmt.Errorf("parameters.setTolerance: %s value is required for this mode", field)
	}
	q, err := resolveQuantity(holder, expr, p.Unit())
	if err != nil {
		return 0, fmt.Errorf("parameters.setTolerance: %s %q: %w", field, expr, err)
	}
	return q.Value, nil
}

// setParameterExpressionList replaces the multi-value choices (empty clears),
// records one undo step, and returns the updated detail.
func setParameterExpressionList(s *app.Session, in wire.ParameterExpressionListArgs) (wire.ParameterDetail, error) {
	holder, p, err := paramByName(s, wire.MethodParametersSetExpressionList, in.Name)
	if err != nil {
		return wire.ParameterDetail{}, err
	}
	if len(in.Expressions) == 0 {
		p.ClearExpressionList()
	} else if err := replaceExpressionList(p, in); err != nil {
		return wire.ParameterDetail{}, err
	}
	s.RecordActiveEdit("Edit Value List")
	return paramDetail(holder, p), nil
}

// replaceExpressionList sets the choices and the ordering flag.
func replaceExpressionList(p *param.Parameter, in wire.ParameterExpressionListArgs) error {
	if err := p.SetExpressionList(in.Expressions, in.AllowCustomValues); err != nil {
		return err
	}
	if !in.CustomOrder {
		p.SetCustomOrder(false)
	}
	return nil
}

// deleteParameter removes an unused parameter through the shared Session verb;
// the in-use refusal (with the blockers named) comes from the aggregate, so
// this wire path and the head UI enforce one invariant (#1612, audit B1).
func deleteParameter(s *app.Session, in wire.ParameterNameArgs) (struct{}, error) {
	_, p, err := paramByName(s, wire.MethodParametersDelete, in.Name)
	if err != nil {
		return struct{}{}, err
	}
	if err := s.DeleteParameter(p.ID()); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

// parameterDrivenBy / parameterDependents answer the dependency queries.
func parameterDrivenBy(s *app.Session, in wire.ParameterNameArgs) (wire.ParameterNamesResult, error) {
	return parameterNeighbors(s, wire.MethodParametersDrivenBy, in, (*param.Parameters).DrivenBy)
}

func parameterDependents(s *app.Session, in wire.ParameterNameArgs) (wire.ParameterNamesResult, error) {
	return parameterNeighbors(s, wire.MethodParametersDependents, in, (*param.Parameters).Dependents)
}

// parameterNeighbors resolves one side of the dependency graph to names.
func parameterNeighbors(s *app.Session, method string, in wire.ParameterNameArgs, side func(*param.Parameters, param.ID) []param.ID) (wire.ParameterNamesResult, error) {
	holder, p, err := paramByName(s, method, in.Name)
	if err != nil {
		return wire.ParameterNamesResult{}, err
	}
	ps := holder.Parameters()
	return wire.ParameterNamesResult{Names: paramNames(ps, side(ps, p.ID()))}, nil
}
