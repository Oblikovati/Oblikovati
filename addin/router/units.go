// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"errors"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/param"
)

// Document units of measure + the unit conversion / expression / formatting
// service (Oblikovati/Oblikovati#146). Backed by the active document's
// param.UnitsOfMeasure and parameter set.

// unitsDocument is the document content that owns display units and a parameter
// scope — satisfied by both part and assembly definitions.
type unitsDocument interface {
	Units() param.UnitsOfMeasure
	SetUnits(param.UnitsOfMeasure)
	Parameters() *param.Parameters
}

// activeUnitsDocument resolves the active document's units-bearing content.
func activeUnitsDocument(s *app.Session) (unitsDocument, error) {
	d := s.ActiveDocument()
	if d == nil {
		return nil, errors.New("units: no active document")
	}
	doc, ok := d.Content().(unitsDocument)
	if !ok {
		return nil, fmt.Errorf("units: active document %q has no units of measure", d.DisplayName())
	}
	return doc, nil
}

// unitsInfoOf marshals a document's display-unit preferences into the wire DTO.
func unitsInfoOf(u param.UnitsOfMeasure) wire.DocumentUnitsInfo {
	return wire.DocumentUnitsInfo{
		LengthUnit:             u.PreferredName(param.Length),
		AngleUnit:              u.PreferredName(param.Angle),
		MassUnit:               u.PreferredName(param.Mass),
		TimeUnit:               u.PreferredName(param.Time),
		LengthDisplayPrecision: u.LengthPrecision(),
		AngleDisplayPrecision:  u.AnglePrecision(),
		LengthDisplayFormat:    u.LengthFormat().String(),
		WorkingScaleCm:         u.WorkingScale(),
	}
}

// categoryOf resolves a types.UnitsType spelling ("length") to a param.Unit.
func categoryOf(spelling string) (param.Unit, error) {
	cat, ok := param.ParseUnitCategory(spelling)
	if !ok {
		return 0, fmt.Errorf("units: unknown unit category %q", spelling)
	}
	return cat, nil
}

// getDocumentUnits returns the active document's display-unit preferences.
func getDocumentUnits(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	doc, err := activeUnitsDocument(s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(unitsInfoOf(doc.Units()))
}

// setDocumentUnits applies the non-nil unit/precision/format preferences and
// returns the updated units. It validates each change first; a bad value
// rejects the whole update naming the offending value.
func setDocumentUnits(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	doc, err := activeUnitsDocument(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetDocumentUnitsArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	u := doc.Units().Clone()
	if err := applyUnitPreferences(&u, in); err != nil {
		return nil, err
	}
	doc.SetUnits(u)
	return json.Marshal(unitsInfoOf(doc.Units()))
}

// applyUnitPreferences mutates u with each supplied (non-nil) preference.
func applyUnitPreferences(u *param.UnitsOfMeasure, in wire.SetDocumentUnitsArgs) error {
	for cat, name := range map[param.Unit]*string{
		param.Length: in.LengthUnit, param.Angle: in.AngleUnit,
		param.Mass: in.MassUnit, param.Time: in.TimeUnit,
	} {
		if name != nil {
			if err := u.SetPreferred(cat, *name); err != nil {
				return err
			}
		}
	}
	if in.LengthDisplayPrecision != nil {
		if err := u.SetLengthPrecision(*in.LengthDisplayPrecision); err != nil {
			return err
		}
	}
	if in.AngleDisplayPrecision != nil {
		if err := u.SetAnglePrecision(*in.AngleDisplayPrecision); err != nil {
			return err
		}
	}
	if in.LengthDisplayFormat != nil {
		f, ok := types.ParseParameterDisplayFormat(*in.LengthDisplayFormat)
		if !ok {
			return fmt.Errorf("units: unknown length display format %q", *in.LengthDisplayFormat)
		}
		u.SetLengthFormat(f)
	}
	return nil
}

// unitsConvert converts a value between two unit names of the same category.
func unitsConvert(_ *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.ConvertUnitsArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	v, err := param.ConvertValue(in.Value, in.From, in.To)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.ConvertUnitsResult{Value: v})
}

// unitsGetStringFromValue formats a database-unit value in the document's
// display unit, honoring the display precision and format (decimal / fractional
// / architectural lengths, decimal-degree / DMS angles).
func unitsGetStringFromValue(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	doc, cat, value, err := stringFromValueInputs(s, args)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.StringResult{Value: doc.Units().FormatDisplay(param.Q(value, cat))})
}

// unitsGetPreciseStringFromValue formats a database-unit value at full
// precision (no display rounding).
func unitsGetPreciseStringFromValue(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	doc, cat, value, err := stringFromValueInputs(s, args)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.StringResult{Value: doc.Units().Format(param.Q(value, cat))})
}

// stringFromValueInputs decodes and resolves the shared inputs of the two
// value-formatting methods.
func stringFromValueInputs(s *app.Session, args json.RawMessage) (unitsDocument, param.Unit, float64, error) {
	doc, err := activeUnitsDocument(s)
	if err != nil {
		return nil, 0, 0, err
	}
	var in wire.StringFromValueArgs
	if err := decode(args, &in); err != nil {
		return nil, 0, 0, err
	}
	cat, err := categoryOf(in.UnitsType)
	if err != nil {
		return nil, 0, 0, err
	}
	return doc, cat, in.Value, nil
}

// unitsGetValueFromExpression evaluates an expression to a database-unit value
// of the target category. A bare number is interpreted in that category's
// display unit (the GetValueFromExpression convention).
func unitsGetValueFromExpression(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	doc, in, err := expressionWithType(s, args)
	if err != nil {
		return nil, err
	}
	cat, err := categoryOf(in.UnitsType)
	if err != nil {
		return nil, err
	}
	q, err := doc.Parameters().EvaluateExpression(in.Expression)
	if err != nil {
		return nil, err
	}
	if q.Unit == param.Unitless && cat != param.Unitless {
		q = doc.Units().FromPreferred(q.Value, cat)
	}
	return json.Marshal(wire.ValueResult{Value: q.Value})
}

// unitsGetDatabaseUnitsFromExpression evaluates an expression to a database-unit
// value, auto-detecting its category.
func unitsGetDatabaseUnitsFromExpression(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	doc, err := activeUnitsDocument(s)
	if err != nil {
		return nil, err
	}
	var in wire.ExpressionArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	q, err := doc.Parameters().EvaluateExpression(in.Expression)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.DatabaseUnitsResult{Value: q.Value, UnitsType: q.Unit.String()})
}

// unitsIsExpressionValid reports whether an expression parses, evaluates, and is
// dimensionally compatible with the target category.
func unitsIsExpressionValid(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	doc, in, err := expressionWithType(s, args)
	if err != nil {
		return nil, err
	}
	cat, err := categoryOf(in.UnitsType)
	if err != nil {
		return nil, err
	}
	if q, evErr := doc.Parameters().EvaluateExpression(in.Expression); evErr != nil {
		return json.Marshal(wire.ExpressionValidResult{Valid: false, Error: evErr.Error()})
	} else if !unitCompatible(q.Unit, cat) {
		return json.Marshal(wire.ExpressionValidResult{
			Valid: false,
			Error: fmt.Sprintf("expression unit %s is not compatible with %s", q.Unit, cat),
		})
	}
	return json.Marshal(wire.ExpressionValidResult{Valid: true})
}

// unitsCompatibleUnits reports whether an expression's resolved unit is
// dimensionally compatible with the target category.
func unitsCompatibleUnits(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	doc, in, err := expressionWithType(s, args)
	if err != nil {
		return nil, err
	}
	cat, err := categoryOf(in.UnitsType)
	if err != nil {
		return nil, err
	}
	q, evErr := doc.Parameters().EvaluateExpression(in.Expression)
	return json.Marshal(wire.CompatibleUnitsResult{Compatible: evErr == nil && unitCompatible(q.Unit, cat)})
}

// unitCompatible reports whether a resolved unit can stand in for the target
// category: an exact category match, or a bare (unitless) number coerced to it.
func unitCompatible(got, want param.Unit) bool {
	return got == want || got == param.Unitless
}

// expressionWithType decodes the shared {expression, unitsType} request and
// resolves the active units document.
func expressionWithType(s *app.Session, args json.RawMessage) (unitsDocument, wire.ExpressionWithTypeArgs, error) {
	doc, err := activeUnitsDocument(s)
	if err != nil {
		return nil, wire.ExpressionWithTypeArgs{}, err
	}
	var in wire.ExpressionWithTypeArgs
	if err := decode(args, &in); err != nil {
		return nil, wire.ExpressionWithTypeArgs{}, err
	}
	return doc, in, nil
}

// unitsGetTypeFromString returns the category a unit name belongs to.
func unitsGetTypeFromString(_ *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.UnitStringArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	cat, ok := param.UnitCategoryOf(in.UnitString)
	if !ok {
		return nil, fmt.Errorf("units: unknown unit %q", in.UnitString)
	}
	return json.Marshal(wire.UnitsTypeResult{UnitsType: cat.String()})
}

// unitsGetStringFromType returns the document-preferred unit name for a category.
func unitsGetStringFromType(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	doc, err := activeUnitsDocument(s)
	if err != nil {
		return nil, err
	}
	var in wire.UnitsTypeArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	cat, err := categoryOf(in.UnitsType)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.StringResult{Value: doc.Units().PreferredName(cat)})
}

// unitsGetLocaleCorrectedExpression normalizes an expression's number formatting
// to the canonical evaluator form. The evaluator currently accepts the
// canonical (en) form directly, so this is a validated passthrough.
func unitsGetLocaleCorrectedExpression(_ *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.ExpressionArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	if _, err := param.Parse(in.Expression); err != nil {
		return nil, err
	}
	return json.Marshal(wire.StringResult{Value: in.Expression})
}

// unitsGetDrivingParameters returns the parameter names an expression references.
func unitsGetDrivingParameters(_ *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.ExpressionArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	names, err := param.ExpressionReferences(in.Expression)
	if err != nil {
		return nil, err
	}
	if names == nil {
		names = []string{}
	}
	return json.Marshal(wire.DrivingParametersResult{Names: names})
}
