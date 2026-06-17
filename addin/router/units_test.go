// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"strings"
	"testing"

	"oblikovati.org/api/wire"
)

// Document units of measure + unit/expression service over the wire (#146).

func TestDocumentUnitsRoundTripOverWire(t *testing.T) {
	r, s := seededSession(t)

	var got wire.DocumentUnitsInfo
	call(t, r, s, "documents.getUnits", "{}", &got)
	if got.LengthUnit != "mm" || got.AngleUnit != "deg" || got.MassUnit != "kg" || got.TimeUnit != "s" {
		t.Fatalf("default units = %+v, want metric mm/deg/kg/s", got)
	}
	if got.LengthDisplayPrecision != 3 || got.AngleDisplayPrecision != 2 || got.LengthDisplayFormat != "decimal" {
		t.Fatalf("default precision/format = %+v, want 3/2/decimal", got)
	}

	call(t, r, s, "documents.setUnits", `{"lengthUnit":"in","lengthDisplayPrecision":4,"lengthDisplayFormat":"fractional"}`, &got)
	if got.LengthUnit != "in" || got.LengthDisplayPrecision != 4 || got.LengthDisplayFormat != "fractional" {
		t.Errorf("after set = %+v, want in/4/fractional", got)
	}
	// Unsent fields stay untouched.
	if got.AngleUnit != "deg" || got.AngleDisplayPrecision != 2 {
		t.Errorf("angle prefs changed to %s/%d, want the untouched deg/2", got.AngleUnit, got.AngleDisplayPrecision)
	}

	for args, want := range map[string]string{
		`{"lengthUnit":"furlong"}`:        "registered length unit",
		`{"lengthDisplayFormat":"bogus"}`: "display format",
		`{"lengthDisplayPrecision":-1}`:   "negative",
	} {
		if _, err := r.Handle(s, "documents.setUnits", []byte(args)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("setUnits(%s) err = %v, want it to mention %q", args, err, want)
		}
	}
}

func TestUnitsServiceOverWire(t *testing.T) {
	r, s := seededSession(t)

	var conv wire.ConvertUnitsResult
	call(t, r, s, "units.convert", `{"value":1,"from":"in","to":"cm"}`, &conv)
	if conv.Value != 2.54 {
		t.Errorf("convert 1 in→cm = %g, want 2.54", conv.Value)
	}
	if _, err := r.Handle(s, "units.convert", []byte(`{"value":1,"from":"mm","to":"deg"}`)); err == nil {
		t.Error("converting mm→deg (different categories) should error")
	}

	var str wire.StringResult
	call(t, r, s, "units.getStringFromValue", `{"value":4,"unitsType":"length"}`, &str)
	if str.Value != "40 mm" {
		t.Errorf("getStringFromValue(4 cm) = %q, want \"40 mm\"", str.Value)
	}

	var val wire.ValueResult
	call(t, r, s, "units.getValueFromExpression", `{"expression":"width * 2","unitsType":"length"}`, &val)
	if val.Value != 8 {
		t.Errorf("width*2 = %g cm, want 8", val.Value)
	}
	// A bare number is interpreted in the category's display unit (mm → cm).
	call(t, r, s, "units.getValueFromExpression", `{"expression":"5","unitsType":"length"}`, &val)
	if val.Value != 0.5 {
		t.Errorf("bare 5 (mm) = %g cm, want 0.5", val.Value)
	}

	var dbu wire.DatabaseUnitsResult
	call(t, r, s, "units.getDatabaseUnitsFromExpression", `{"expression":"25 mm"}`, &dbu)
	if dbu.Value != 2.5 || dbu.UnitsType != "length" {
		t.Errorf("25 mm = %+v, want 2.5 cm / length", dbu)
	}

	var typ wire.UnitsTypeResult
	call(t, r, s, "units.getTypeFromString", `{"unitString":"in"}`, &typ)
	if typ.UnitsType != "length" {
		t.Errorf("typeFromString(in) = %q, want length", typ.UnitsType)
	}

	var name wire.StringResult
	call(t, r, s, "units.getStringFromType", `{"unitsType":"length"}`, &name)
	if name.Value != "mm" {
		t.Errorf("stringFromType(length) = %q, want mm", name.Value)
	}

	var v wire.ExpressionValidResult
	call(t, r, s, "units.isExpressionValid", `{"expression":"3 mm + 1 cm","unitsType":"length"}`, &v)
	if !v.Valid {
		t.Errorf("3 mm + 1 cm should be valid length, got %+v", v)
	}
	call(t, r, s, "units.isExpressionValid", `{"expression":"width","unitsType":"angle"}`, &v)
	if v.Valid {
		t.Error("a length expression should be invalid for the angle category")
	}

	var dp wire.DrivingParametersResult
	call(t, r, s, "units.getDrivingParameters", `{"expression":"width + height"}`, &dp)
	if len(dp.Names) != 2 || dp.Names[0] != "width" || dp.Names[1] != "height" {
		t.Errorf("drivingParameters = %v, want [width height]", dp.Names)
	}
}
