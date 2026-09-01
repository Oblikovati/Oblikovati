// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"strings"
	"testing"

	"oblikovati.org/api/wire"
)

// Parameter settings, tolerance sweeps & XML exchange over the wire
// (M02-F07, Oblikovati#606).

func TestParameterSettingsRoundTripOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)

	var got wire.ParameterSettingsInfo
	call(t, r, s, "parameters.getSettings", "{}", &got)
	if got.LinearDimensionPrecision != 3 || got.AngularDimensionPrecision != 2 || got.DimensionDisplayType != "value" {
		t.Fatalf("default settings = %+v, want 3/2 precision and value display", got)
	}

	call(t, r, s, "parameters.setSettings", `{
		"linearStandardTolerance":"0.1 mm","useStandardTolerances":true,
		"linearDimensionPrecision":4,"dimensionDisplayType":"expression"
	}`, &got)
	if got.LinearStandardTolerance != "0.1 mm" || !got.UseStandardTolerances {
		t.Errorf("standard tolerance = %+v, want 0.1 mm enabled", got)
	}
	if got.LinearDimensionPrecision != 4 || got.DimensionDisplayType != "expression" {
		t.Errorf("precision/display = %d/%q, want 4/expression", got.LinearDimensionPrecision, got.DimensionDisplayType)
	}
	// Unsent fields stay untouched.
	if got.AngularDimensionPrecision != 2 {
		t.Errorf("angular precision = %d, want the untouched 2", got.AngularDimensionPrecision)
	}

	for args, want := range map[string]string{
		`{"dimensionDisplayType":"hex"}`:   "dimension display type",
		`{"linearDimensionPrecision":12}`:  "out of range",
		`{"angularDimensionPrecision":-1}`: "out of range",
	} {
		if _, err := r.Handle(s, "parameters.setSettings", []byte(args)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("setSettings(%s) err = %v, want it to mention %q", args, err, want)
		}
	}
}

func TestParameterSweepOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	call(t, r, s, "parameters.add", `{"name":"od","expression":"4 cm"}`, nil)
	call(t, r, s, "parameters.setTolerance", `{"name":"od","mode":"deviation","upper":"0.2 cm","lower":"-0.1 cm"}`, nil)
	call(t, r, s, "parameters.setTolerance", `{"name":"width","mode":"symmetric","upper":"0.05 cm"}`, nil)

	var res wire.ParameterSweepResult
	call(t, r, s, "parameters.setAllModelValueType", `{"modelValueType":"upper"}`, &res)
	if res.Affected != 2 {
		t.Fatalf("sweep affected = %d, want both toleranced parameters", res.Affected)
	}
	if d := getDetail(t, r, s, "od"); d.ModelValueType != "upper" || !approx(d.ModelValue, 4.2) {
		t.Errorf("od after sweep = %v (%s), want 4.2 upper", d.ModelValue, d.ModelValueType)
	}

	call(t, r, s, "parameters.setAllModelValueType", `{"modelValueType":"nominal"}`, &res)
	if d := getDetail(t, r, s, "width"); d.ModelValueType != "nominal" || !approx(d.ModelValue, 4) {
		t.Errorf("width after nominal sweep = %v (%s), want the nominal back", d.ModelValue, d.ModelValueType)
	}

	if _, err := r.Handle(s, "parameters.setAllModelValueType", []byte(`{"modelValueType":"sideways"}`)); err == nil {
		t.Error("unknown sweep selection must be rejected")
	}
}

func TestParameterXMLExchangeOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	call(t, r, s, "parameters.update", `{"name":"width","comment":"outer width"}`, nil)

	var exp wire.ParameterExportResult
	call(t, r, s, "parameters.export", "{}", &exp)
	if !strings.Contains(exp.XML, `name="width"`) || !strings.Contains(exp.XML, `comment="outer width"`) {
		t.Fatalf("export = %s, want width with its comment", exp.XML)
	}

	// Re-import an edited set: width updates, a new parameter appears.
	edited := strings.Replace(exp.XML, `expression="4 cm"`, `expression="5 cm"`, 1)
	edited = strings.Replace(edited, "</parameters>",
		`<parameter name="depth" expression="2 cm"/></parameters>`, 1)
	var imp wire.ParameterImportResult
	call(t, r, s, "parameters.import", `{"xml":`+jsonString(edited)+`}`, &imp)
	if imp.Added != 1 || imp.Updated != 1 {
		t.Fatalf("import counts = %+v, want 1 added / 1 updated", imp)
	}
	if d := getDetail(t, r, s, "width"); d.Expression != "5 cm" {
		t.Errorf("width expression = %q, want the imported 5 cm", d.Expression)
	}
	if d := getDetail(t, r, s, "depth"); d.Expression != "2 cm" {
		t.Errorf("depth = %+v, want the imported parameter", d)
	}
}

// TestParameterImportRollsBackOnBadSet checks atomicity: a set whose late
// entry fails must leave no trace of its earlier entries.
func TestParameterImportRollsBackOnBadSet(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	bad := `<parameters>
		<parameter name="ok" expression="1 cm"/>
		<parameter name="broken" expression="1 nonsenseunit"/>
	</parameters>`
	if _, err := r.Handle(s, "parameters.import", []byte(`{"xml":`+jsonString(bad)+`}`)); err == nil || !strings.Contains(err.Error(), "broken") {
		t.Fatalf("import err = %v, want a rejection naming the broken entry", err)
	}
	if _, err := r.Handle(s, "parameters.getDetail", []byte(`{"name":"ok"}`)); err == nil {
		t.Error("rolled-back import must not leave earlier entries behind")
	}
}
