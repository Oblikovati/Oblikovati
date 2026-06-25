// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"strings"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
)

// The member-level parameter surface (M02-F08, Oblikovati#607): detail reads,
// presentation/tolerance/value-list mutations, dependency queries and delete.
// Mutations finalize through Session.RecordAddInEdit so they are undoable, and
// the router broadcasts them as edit.committed (mutatingMethods).

// paramDetail marshals the full member-level view of one parameter.
func paramDetail(part *compdef.PartComponentDefinition, p *param.Parameter) wire.ParameterDetail {
	ps := part.Parameters()
	d := wire.ParameterDetail{
		ParameterInfo: paramInfo(part, p),
		Units:         paramUnitName(part, p),
		Comment:       p.Comment, IsKey: p.IsKey, Visible: p.Visible,
		InUse: ps.InUse(p.ID()), Precision: p.Precision,
		DisplayFormat: p.DisplayFormat.String(), ExposedAsProperty: p.ExposedAsProperty,
		ModelValueType:       p.ModelValueType().String(),
		CustomPropertyFormat: customPropertyInfo(p.CustomProperty),
		DrivenBy:             paramNames(ps, ps.DrivenBy(p.ID())),
		Dependents:           paramNames(ps, ps.Dependents(p.ID())),
	}
	if p.IsNumeric() {
		tol := p.Tolerance()
		d.ModelValue = p.ModelValue()
		d.Tolerance = &wire.ToleranceInfo{Type: tol.Kind().String(), Upper: tol.Upper, Lower: tol.Lower}
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
func paramUnitName(part *compdef.PartComponentDefinition, p *param.Parameter) string {
	if name := part.Units().PreferredName(p.Unit()); name != "" {
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

// paramByName resolves one parameter on the active part, naming the method in
// the not-found error.
func paramByName(s *app.Session, method, name string) (*compdef.PartComponentDefinition, *param.Parameter, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, nil, err
	}
	p, ok := part.Parameters().ByName(name)
	if !ok {
		return nil, nil, fmt.Errorf("%s: no parameter named %q", method, name)
	}
	return part, p, nil
}

// getParameterDetail returns the member-level view of one parameter.
func getParameterDetail(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.ParameterNameArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	part, p, err := paramByName(s, "parameters.getDetail", in.Name)
	if err != nil {
		return nil, err
	}
	return json.Marshal(paramDetail(part, p))
}

// updateParameter applies the non-nil presentation/exposure mutations, records
// one undo step, and returns the updated detail.
func updateParameter(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.ParameterUpdateArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	part, p, err := paramByName(s, "parameters.update", in.Name)
	if err != nil {
		return nil, err
	}
	if err := applyParameterUpdate(p, in); err != nil {
		return nil, err
	}
	// A model-value selection change moves the value features consume.
	if in.ModelValueType != nil {
		part.RecomputeAfterParameterEdit()
	}
	s.RecordAddInEdit(part, "Edit Parameter")
	return json.Marshal(paramDetail(part, p))
}

// applyParameterUpdate copies the non-nil update fields onto the parameter.
func applyParameterUpdate(p *param.Parameter, in wire.ParameterUpdateArgs) error {
	setIfPresent(in.Comment, &p.Comment)
	setIfPresent(in.IsKey, &p.IsKey)
	setIfPresent(in.Visible, &p.Visible)
	setIfPresent(in.Precision, &p.Precision)
	setIfPresent(in.ExposedAsProperty, &p.ExposedAsProperty)
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
	return applyCustomPropertyUpdate(p, in.CustomPropertyFormat)
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
func setParameterTolerance(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.ParameterToleranceArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	part, p, err := paramByName(s, "parameters.setTolerance", in.Name)
	if err != nil {
		return nil, err
	}
	if err := applyToleranceMode(part, p, in); err != nil {
		return nil, err
	}
	part.RecomputeAfterParameterEdit()
	s.RecordAddInEdit(part, "Edit Tolerance")
	return json.Marshal(paramDetail(part, p))
}

// applyToleranceMode dispatches one wire tolerance mode onto the model setters.
func applyToleranceMode(part *compdef.PartComponentDefinition, p *param.Parameter, in wire.ParameterToleranceArgs) error {
	switch in.Mode {
	case "default":
		return p.SetToleranceDefault()
	case "min":
		return p.SetToleranceMinMax(types.ToleranceMin)
	case "max":
		return p.SetToleranceMinMax(types.ToleranceMax)
	case "symmetric":
		band, err := toleranceOperand(part, p, in.Upper, "upper")
		if err != nil {
			return err
		}
		return p.SetToleranceSymmetric(band)
	case "deviation", "limits":
		return applyToleranceBand(part, p, in)
	default:
		return fmt.Errorf("parameters.setTolerance: unknown mode %q (want default|deviation|symmetric|limits|min|max)", in.Mode)
	}
}

// applyToleranceBand parses both operands and applies a deviation or limits band.
func applyToleranceBand(part *compdef.PartComponentDefinition, p *param.Parameter, in wire.ParameterToleranceArgs) error {
	upper, err := toleranceOperand(part, p, in.Upper, "upper")
	if err != nil {
		return err
	}
	lower, err := toleranceOperand(part, p, in.Lower, "lower")
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
func toleranceOperand(part *compdef.PartComponentDefinition, p *param.Parameter, expr, field string) (float64, error) {
	if expr == "" {
		return 0, fmt.Errorf("parameters.setTolerance: %s value is required for this mode", field)
	}
	q, err := resolveQuantity(part, expr, p.Unit())
	if err != nil {
		return 0, fmt.Errorf("parameters.setTolerance: %s %q: %w", field, expr, err)
	}
	return q.Value, nil
}

// setParameterExpressionList replaces the multi-value choices (empty clears),
// records one undo step, and returns the updated detail.
func setParameterExpressionList(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.ParameterExpressionListArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	part, p, err := paramByName(s, "parameters.setExpressionList", in.Name)
	if err != nil {
		return nil, err
	}
	if len(in.Expressions) == 0 {
		p.ClearExpressionList()
	} else if err := replaceExpressionList(p, in); err != nil {
		return nil, err
	}
	s.RecordAddInEdit(part, "Edit Value List")
	return json.Marshal(paramDetail(part, p))
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

// deleteParameter removes an unused parameter; an in-use one is rejected with
// the offending dependents so the caller can resolve them first.
func deleteParameter(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.ParameterNameArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	part, p, err := paramByName(s, "parameters.delete", in.Name)
	if err != nil {
		return nil, err
	}
	ps := part.Parameters()
	if ps.InUse(p.ID()) {
		return nil, fmt.Errorf("parameters.delete: %q is in use by [%s]; remove those references first",
			in.Name, strings.Join(deleteBlockers(ps, p), ", "))
	}
	if err := ps.Delete(p.ID()); err != nil {
		return nil, err
	}
	part.RecomputeAfterParameterEdit()
	s.RecordAddInEdit(part, "Delete Parameter")
	return json.Marshal(struct{}{})
}

// deleteBlockers names what keeps a parameter alive: its dependents, or its
// owning feature dimension for model parameters.
func deleteBlockers(ps *param.Parameters, p *param.Parameter) []string {
	if names := paramNames(ps, ps.Dependents(p.ID())); len(names) > 0 {
		return names
	}
	return []string{"its feature dimension"}
}

// parameterDrivenBy / parameterDependents answer the dependency queries.
func parameterDrivenBy(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	return parameterNeighbors(s, "parameters.drivenBy", args, (*param.Parameters).DrivenBy)
}

func parameterDependents(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	return parameterNeighbors(s, "parameters.dependents", args, (*param.Parameters).Dependents)
}

// parameterNeighbors resolves one side of the dependency graph to names.
func parameterNeighbors(s *app.Session, method string, args json.RawMessage, side func(*param.Parameters, param.ID) []param.ID) (json.RawMessage, error) {
	var in wire.ParameterNameArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	part, p, err := paramByName(s, method, in.Name)
	if err != nil {
		return nil, err
	}
	ps := part.Parameters()
	return json.Marshal(wire.ParameterNamesResult{Names: paramNames(ps, side(ps, p.ID()))})
}
