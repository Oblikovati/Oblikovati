// SPDX-License-Identifier: GPL-2.0-only

package router

import (
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

// activeUnitsDocument resolves the active document's units-bearing content. It is the resolver
// the typedCtx/ctxQuery adapters run before the units handlers (the same custom-context seam
// ActivePart/ActiveAssembly fill for part/assembly handlers, #1649).
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
func getDocumentUnits(_ *app.Session, doc unitsDocument) (wire.DocumentUnitsInfo, error) {
	return unitsInfoOf(doc.Units()), nil
}

// setDocumentUnits applies the non-nil unit/precision/format preferences and
// returns the updated units. It validates each change first; a bad value
// rejects the whole update naming the offending value.
func setDocumentUnits(_ *app.Session, doc unitsDocument, in wire.SetDocumentUnitsArgs) (wire.DocumentUnitsInfo, error) {
	u := doc.Units().Clone()
	if err := applyUnitPreferences(&u, in); err != nil {
		return wire.DocumentUnitsInfo{}, err
	}
	doc.SetUnits(u)
	return unitsInfoOf(doc.Units()), nil
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
func unitsConvert(_ *app.Session, in wire.ConvertUnitsArgs) (wire.ConvertUnitsResult, error) {
	v, err := param.ConvertValue(in.Value, in.From, in.To)
	if err != nil {
		return wire.ConvertUnitsResult{}, err
	}
	return wire.ConvertUnitsResult{Value: v}, nil
}

// unitsGetStringFromValue formats a database-unit value in the document's
// display unit, honoring the display precision and format (decimal / fractional
// / architectural lengths, decimal-degree / DMS angles).
func unitsGetStringFromValue(_ *app.Session, doc unitsDocument, in wire.StringFromValueArgs) (wire.StringResult, error) {
	cat, err := categoryOf(in.UnitsType)
	if err != nil {
		return wire.StringResult{}, err
	}
	return wire.StringResult{Value: doc.Units().FormatDisplay(param.Q(in.Value, cat))}, nil
}

// unitsGetPreciseStringFromValue formats a database-unit value at full
// precision (no display rounding).
func unitsGetPreciseStringFromValue(_ *app.Session, doc unitsDocument, in wire.StringFromValueArgs) (wire.StringResult, error) {
	cat, err := categoryOf(in.UnitsType)
	if err != nil {
		return wire.StringResult{}, err
	}
	return wire.StringResult{Value: doc.Units().Format(param.Q(in.Value, cat))}, nil
}

// unitsGetValueFromExpression evaluates an expression to a database-unit value
// of the target category. A bare number is interpreted in that category's
// display unit (the GetValueFromExpression convention).
func unitsGetValueFromExpression(_ *app.Session, doc unitsDocument, in wire.ExpressionWithTypeArgs) (wire.ValueResult, error) {
	cat, err := categoryOf(in.UnitsType)
	if err != nil {
		return wire.ValueResult{}, err
	}
	q, err := doc.Parameters().EvaluateExpression(in.Expression)
	if err != nil {
		return wire.ValueResult{}, err
	}
	if q.Unit == param.Unitless && cat != param.Unitless {
		q = doc.Units().FromPreferred(q.Value, cat)
	}
	return wire.ValueResult{Value: q.Value}, nil
}

// unitsGetDatabaseUnitsFromExpression evaluates an expression to a database-unit
// value, auto-detecting its category.
func unitsGetDatabaseUnitsFromExpression(_ *app.Session, doc unitsDocument, in wire.ExpressionArgs) (wire.DatabaseUnitsResult, error) {
	q, err := doc.Parameters().EvaluateExpression(in.Expression)
	if err != nil {
		return wire.DatabaseUnitsResult{}, err
	}
	return wire.DatabaseUnitsResult{Value: q.Value, UnitsType: q.Unit.String()}, nil
}

// unitsIsExpressionValid reports whether an expression parses, evaluates, and is
// dimensionally compatible with the target category.
func unitsIsExpressionValid(_ *app.Session, doc unitsDocument, in wire.ExpressionWithTypeArgs) (wire.ExpressionValidResult, error) {
	cat, err := categoryOf(in.UnitsType)
	if err != nil {
		return wire.ExpressionValidResult{}, err
	}
	q, evErr := doc.Parameters().EvaluateExpression(in.Expression)
	if evErr != nil {
		return wire.ExpressionValidResult{Valid: false, Error: evErr.Error()}, nil
	}
	if !unitCompatible(q.Unit, cat) {
		return wire.ExpressionValidResult{
			Valid: false,
			Error: fmt.Sprintf("expression unit %s is not compatible with %s", q.Unit, cat),
		}, nil
	}
	return wire.ExpressionValidResult{Valid: true}, nil
}

// unitsCompatibleUnits reports whether an expression's resolved unit is
// dimensionally compatible with the target category.
func unitsCompatibleUnits(_ *app.Session, doc unitsDocument, in wire.ExpressionWithTypeArgs) (wire.CompatibleUnitsResult, error) {
	cat, err := categoryOf(in.UnitsType)
	if err != nil {
		return wire.CompatibleUnitsResult{}, err
	}
	q, evErr := doc.Parameters().EvaluateExpression(in.Expression)
	return wire.CompatibleUnitsResult{Compatible: evErr == nil && unitCompatible(q.Unit, cat)}, nil
}

// unitCompatible reports whether a resolved unit can stand in for the target
// category: an exact category match, or a bare (unitless) number coerced to it.
func unitCompatible(got, want param.Unit) bool {
	return got == want || got == param.Unitless
}

// unitsGetTypeFromString returns the category a unit name belongs to.
func unitsGetTypeFromString(_ *app.Session, in wire.UnitStringArgs) (wire.UnitsTypeResult, error) {
	cat, ok := param.UnitCategoryOf(in.UnitString)
	if !ok {
		return wire.UnitsTypeResult{}, fmt.Errorf("units: unknown unit %q", in.UnitString)
	}
	return wire.UnitsTypeResult{UnitsType: cat.String()}, nil
}

// unitsGetStringFromType returns the document-preferred unit name for a category.
func unitsGetStringFromType(_ *app.Session, doc unitsDocument, in wire.UnitsTypeArgs) (wire.StringResult, error) {
	cat, err := categoryOf(in.UnitsType)
	if err != nil {
		return wire.StringResult{}, err
	}
	return wire.StringResult{Value: doc.Units().PreferredName(cat)}, nil
}

// unitsGetLocaleCorrectedExpression normalizes an expression's number formatting
// to the canonical evaluator form. The evaluator currently accepts the
// canonical (en) form directly, so this is a validated passthrough.
func unitsGetLocaleCorrectedExpression(_ *app.Session, in wire.ExpressionArgs) (wire.StringResult, error) {
	if _, err := param.Parse(in.Expression); err != nil {
		return wire.StringResult{}, err
	}
	return wire.StringResult{Value: in.Expression}, nil
}

// unitsGetDrivingParameters returns the parameter names an expression references.
func unitsGetDrivingParameters(_ *app.Session, in wire.ExpressionArgs) (wire.DrivingParametersResult, error) {
	names, err := param.ExpressionReferences(in.Expression)
	if err != nil {
		return wire.DrivingParametersResult{}, err
	}
	if names == nil {
		names = []string{}
	}
	return wire.DrivingParametersResult{Names: names}, nil
}
