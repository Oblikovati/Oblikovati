// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"slices"
	"strings"
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/event"
)

// The member-level parameter surface over the wire (M02-F08, Oblikovati#607):
// parameters.getDetail/update/setTolerance/setExpressionList/delete/drivenBy/
// dependents against a live session.

// getDetail fetches one parameter's member-level view, failing the test on error.
func getDetail(t *testing.T, r *Router, s *app.Session, name string) wire.ParameterDetail {
	t.Helper()
	var d wire.ParameterDetail
	call(t, r, s, "parameters.getDetail", `{"name":"`+name+`"}`, &d)
	return d
}

func TestParameterGetDetailDefaults(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	d := getDetail(t, r, s, "width")
	if d.Name != "width" || d.Expression != "4 cm" {
		t.Fatalf("detail identity = %+v, want width / 4 cm", d.ParameterInfo)
	}
	if !d.Visible || d.DisplayFormat != "decimal" || d.ModelValueType != "nominal" {
		t.Errorf("presentation defaults = visible=%v format=%q mvt=%q, want true/decimal/nominal",
			d.Visible, d.DisplayFormat, d.ModelValueType)
	}
	if d.Tolerance == nil || d.Tolerance.Type != "default" || d.Tolerance.Upper != 0 {
		t.Errorf("tolerance = %+v, want bandless default", d.Tolerance)
	}
	if d.ModelValue != 4 {
		t.Errorf("model value = %v, want the 4 cm nominal", d.ModelValue)
	}
	if d.InUse || d.ExpressionList != nil || len(d.DrivenBy)+len(d.Dependents) != 0 {
		t.Errorf("fresh parameter not standalone: %+v", d)
	}
	if d.CustomPropertyFormat == nil || d.CustomPropertyFormat.PropertyType != "text" {
		t.Errorf("custom property format = %+v, want the text default", d.CustomPropertyFormat)
	}
}

func TestParameterDetailDependencyNeighborhood(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	call(t, r, s, "parameters.add", `{"name":"half","expression":"width / 2"}`, nil)

	if d := getDetail(t, r, s, "width"); !d.InUse || !slices.Equal(d.Dependents, []string{"half"}) {
		t.Errorf("width detail = inUse=%v dependents=%v, want in use by [half]", d.InUse, d.Dependents)
	}
	if d := getDetail(t, r, s, "half"); !slices.Equal(d.DrivenBy, []string{"width"}) {
		t.Errorf("half drivenBy = %v, want [width]", d.DrivenBy)
	}

	var names wire.ParameterNamesResult
	call(t, r, s, "parameters.dependents", `{"name":"width"}`, &names)
	if !slices.Equal(names.Names, []string{"half"}) {
		t.Errorf("parameters.dependents(width) = %v, want [half]", names.Names)
	}
	call(t, r, s, "parameters.drivenBy", `{"name":"half"}`, &names)
	if !slices.Equal(names.Names, []string{"width"}) {
		t.Errorf("parameters.drivenBy(half) = %v, want [width]", names.Names)
	}
}

func TestParameterUpdatePresentation(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	var d wire.ParameterDetail
	call(t, r, s, "parameters.update", `{
		"name":"width","comment":"outer width","isKey":true,"visible":false,
		"precision":2,"displayFormat":"fractional","exposedAsProperty":true,
		"customPropertyFormat":{"propertyType":"number","precision":"twoDecimalPlaces","showTrailingZeros":true}
	}`, &d)
	if d.Comment != "outer width" || !d.IsKey || d.Visible || d.Precision != 2 {
		t.Errorf("updated detail = %+v, want comment/key/hidden/precision applied", d)
	}
	if d.DisplayFormat != "fractional" || !d.ExposedAsProperty {
		t.Errorf("format/exposure = %q/%v, want fractional/true", d.DisplayFormat, d.ExposedAsProperty)
	}
	cp := d.CustomPropertyFormat
	if cp == nil || cp.PropertyType != "number" || cp.Precision != "twoDecimalPlaces" || !cp.ShowTrailingZeros {
		t.Errorf("custom property format = %+v, want number/twoDecimalPlaces/trailing zeros", cp)
	}
	// Unsent fields stay untouched: a second update changing only the comment
	// must not reset the rest.
	call(t, r, s, "parameters.update", `{"name":"width","comment":"narrower"}`, &d)
	if d.Comment != "narrower" || !d.IsKey || d.DisplayFormat != "fractional" {
		t.Errorf("partial update clobbered other fields: %+v", d)
	}
}

func TestParameterUpdateRejectsUnknownSpellings(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	for args, want := range map[string]string{
		`{"name":"width","displayFormat":"roman"}`:                                                       "display format",
		`{"name":"width","modelValueType":"midpoint"}`:                                                   "model value type",
		`{"name":"width","customPropertyFormat":{"propertyType":"blob","precision":"twoDecimalPlaces"}}`: "custom property type",
		`{"name":"nope","comment":"x"}`:                                                                  "no parameter named",
	} {
		if _, err := r.Handle(s, "parameters.update", []byte(args)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("update(%s) err = %v, want it to mention %q", args, err, want)
		}
	}
}

// TestParameterIntrospectionOverWire: getDetail reports builtIn/renamed, update
// round-trips disabledActionTypes, and a rename of a model parameter flips
// renamed to true (#1853).
func TestParameterIntrospectionOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	// A user parameter is neither built-in nor renamed, with no disabled actions.
	if d := getDetail(t, r, s, "width"); d.BuiltIn || d.Renamed || len(d.DisabledActionTypes) != 0 {
		t.Errorf("user param introspection = builtIn=%v renamed=%v disabled=%v, want all clear", d.BuiltIn, d.Renamed, d.DisabledActionTypes)
	}
	// disabledActionTypes replaces the whole mask and reads back in emission order.
	var d wire.ParameterDetail
	call(t, r, s, "parameters.update", `{"name":"width","disabledActionTypes":["rename","delete"]}`, &d)
	if !slices.Equal(d.DisabledActionTypes, []string{"rename", "delete"}) {
		t.Errorf("disabledActionTypes = %v, want [rename delete]", d.DisabledActionTypes)
	}
	// An empty list clears it. Read back through a fresh getDetail: the omitempty
	// response field would otherwise leave the reused struct's stale value.
	call(t, r, s, "parameters.update", `{"name":"width","disabledActionTypes":[]}`, nil)
	if cleared := getDetail(t, r, s, "width"); len(cleared.DisabledActionTypes) != 0 {
		t.Errorf("cleared disabledActionTypes = %v, want empty", cleared.DisabledActionTypes)
	}
	// Renaming a model parameter raises renamed.
	call(t, r, s, "parameters.add", `{"name":"d0","kind":"model","expression":"5 mm"}`, nil)
	call(t, r, s, "parameters.rename", `{"name":"d0","newName":"thickness"}`, nil)
	if d := getDetail(t, r, s, "thickness"); !d.Renamed {
		t.Errorf("renamed model param reports renamed=%v, want true", d.Renamed)
	}
}

// TestParameterUpdateRejectsUnknownAction: an unrecognised disabled-action
// spelling is refused, naming the offender (#1853).
func TestParameterUpdateRejectsUnknownAction(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	_, err := r.Handle(s, "parameters.update", []byte(`{"name":"width","disabledActionTypes":["suppress"]}`))
	if err == nil || !strings.Contains(err.Error(), "disabled action type") {
		t.Errorf("update with unknown action err = %v, want it to mention the disabled action type", err)
	}
}

func TestParameterToleranceModesOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	// Each mutation's response is checked through a fresh getDetail: omitempty
	// fields would otherwise merge stale values when reusing one struct.
	call(t, r, s, "parameters.setTolerance", `{"name":"width","mode":"symmetric","upper":"0.1 cm"}`, nil)
	if d := getDetail(t, r, s, "width"); d.Tolerance.Type != "symmetric" || d.Tolerance.Upper != 0.1 || d.Tolerance.Lower != -0.1 {
		t.Errorf("symmetric tolerance = %+v, want ±0.1", d.Tolerance)
	}

	// Limits are absolute values, stored as deviations from the 4 cm nominal.
	call(t, r, s, "parameters.setTolerance", `{"name":"width","mode":"limits","upper":"4.3 cm","lower":"3.8 cm"}`, nil)
	if d := getDetail(t, r, s, "width"); d.Tolerance.Type != "limitsStacked" || !approx(d.Tolerance.Upper, 0.3) || !approx(d.Tolerance.Lower, -0.2) {
		t.Errorf("limits tolerance = %+v, want +0.3/-0.2", d.Tolerance)
	}

	// The model-value selection picks a value inside the band; features consume it.
	call(t, r, s, "parameters.update", `{"name":"width","modelValueType":"upper"}`, nil)
	if d := getDetail(t, r, s, "width"); d.ModelValueType != "upper" || !approx(d.ModelValue, 4.3) {
		t.Errorf("model value after upper selection = %v (%s), want 4.3", d.ModelValue, d.ModelValueType)
	}

	call(t, r, s, "parameters.setTolerance", `{"name":"width","mode":"max"}`, nil)
	if d := getDetail(t, r, s, "width"); d.Tolerance.Type != "max" || d.Tolerance.Upper != 0 || d.Tolerance.Lower != 0 {
		t.Errorf("max tolerance = %+v, want bandless max", d.Tolerance)
	}

	call(t, r, s, "parameters.setTolerance", `{"name":"width","mode":"default"}`, nil)
	if d := getDetail(t, r, s, "width"); d.Tolerance.Type != "default" || !approx(d.ModelValue, 4) {
		t.Errorf("default tolerance = %+v model=%v, want default band and the nominal back", d.Tolerance, d.ModelValue)
	}
}

// TestParameterFitsToleranceOverWire: the fits/basic/reference modes resolve and round-trip
// through the wire (#1848). width is 4 cm (40 mm); H7 at 40 mm is +25/0 µm = +0.0025/0 cm.
func TestParameterFitsToleranceOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	call(t, r, s, "parameters.setTolerance", `{"name":"width","mode":"fits","hole":"H7","shaft":"g6"}`, nil)
	d := getDetail(t, r, s, "width")
	if d.Tolerance.Type != "limitsFitsStacked" || d.Tolerance.HoleTolerance != "H7" || d.Tolerance.ShaftTolerance != "g6" {
		t.Errorf("fits tolerance = %+v, want limitsFitsStacked H7/g6", d.Tolerance)
	}
	if !approx(d.Tolerance.Upper, 0.0025) || d.Tolerance.Lower != 0 {
		t.Errorf("fits band = +%v/%v cm, want +0.0025/0 (40H7)", d.Tolerance.Upper, d.Tolerance.Lower)
	}
	// basic and reference carry no band.
	call(t, r, s, "parameters.setTolerance", `{"name":"width","mode":"basic"}`, nil)
	if d := getDetail(t, r, s, "width"); d.Tolerance.Type != "basic" || d.Tolerance.Upper != 0 {
		t.Errorf("basic tolerance = %+v, want bandless basic", d.Tolerance)
	}
	call(t, r, s, "parameters.setTolerance", `{"name":"width","mode":"reference"}`, nil)
	if d := getDetail(t, r, s, "width"); d.Tolerance.Type != "reference" {
		t.Errorf("reference tolerance = %+v, want reference", d.Tolerance)
	}
}

func TestParameterToleranceRejectsBadInput(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	for args, want := range map[string]string{
		`{"name":"width","mode":"wobbly"}`:                                       "unknown mode",
		`{"name":"width","mode":"symmetric"}`:                                    "upper value is required",
		`{"name":"width","mode":"deviation","upper":"0.1 cm"}`:                   "lower value is required",
		`{"name":"width","mode":"symmetric","upper":"banana"}`:                   "banana",
		`{"name":"width","mode":"deviation","upper":"-0.1 cm","lower":"0.1 cm"}`: "upper",
		`{"name":"width","mode":"fits"}`:                                         "hole or shaft class",
		`{"name":"width","mode":"fits","hole":"Z9"}`:                             "unsupported ISO fit letter",
	} {
		if _, err := r.Handle(s, "parameters.setTolerance", []byte(args)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("setTolerance(%s) err = %v, want it to mention %q", args, err, want)
		}
	}
}

func TestParameterExpressionListOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)

	call(t, r, s, "parameters.setExpressionList",
		`{"name":"width","expressions":["6 cm","4 cm"],"allowCustomValues":true,"customOrder":true}`, nil)
	if el := getDetail(t, r, s, "width").ExpressionList; el == nil || !slices.Equal(el.Expressions, []string{"6 cm", "4 cm"}) || !el.AllowCustomValues || !el.CustomOrder {
		t.Fatalf("expression list = %+v, want authored order [6 cm, 4 cm] with custom values", el)
	}

	// customOrder=false sorts the choices (the reference CustomOrder=False).
	call(t, r, s, "parameters.setExpressionList",
		`{"name":"width","expressions":["6 cm","4 cm"],"customOrder":false}`, nil)
	if el := getDetail(t, r, s, "width").ExpressionList; el == nil || !slices.Equal(el.Expressions, []string{"4 cm", "6 cm"}) || el.CustomOrder {
		t.Errorf("sorted expression list = %+v, want [4 cm, 6 cm]", el)
	}

	// An empty list returns the parameter to single-valued.
	call(t, r, s, "parameters.setExpressionList", `{"name":"width"}`, nil)
	if el := getDetail(t, r, s, "width").ExpressionList; el != nil {
		t.Errorf("cleared expression list = %+v, want single-valued", el)
	}
}

func TestParameterDeleteRejectsInUse(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	call(t, r, s, "parameters.add", `{"name":"half","expression":"width / 2"}`, nil)

	// width drives half, so deleting it must name the blocker.
	if _, err := r.Handle(s, "parameters.delete", []byte(`{"name":"width"}`)); err == nil || !strings.Contains(err.Error(), "half") {
		t.Fatalf("delete(width) err = %v, want in-use rejection naming half", err)
	}

	call(t, r, s, "parameters.delete", `{"name":"half"}`, nil)
	call(t, r, s, "parameters.delete", `{"name":"width"}`, nil)
	if _, err := r.Handle(s, "parameters.getDetail", []byte(`{"name":"width"}`)); err == nil {
		t.Fatal("width still resolvable after delete")
	}
}

// TestParameterDetailMutationsAreUndoable drives a wire mutation against a
// recording session and checks RecordAddInEdit lands it as one undo step.
func TestParameterDetailMutationsAreUndoable(t *testing.T) {
	t.Parallel()
	r := New(opregistry.Default())
	s := app.NewSession()
	if _, err := s.NewPart(); err != nil {
		t.Fatalf("NewPart: %v", err)
	}
	// Add through the recording Session seam (wire parameters.add does not record
	// its own undo step yet) so the undo baseline holds the parameter.
	if err := s.AddNumericUserParameter("h", "3 cm"); err != nil {
		t.Fatalf("add param: %v", err)
	}
	call(t, r, s, "parameters.update", `{"name":"h","comment":"the height"}`, nil)

	var st wire.UndoState
	call(t, r, s, "transaction.state", "{}", &st)
	if !st.CanUndo || st.NextUndo != "Edit Parameter" {
		t.Fatalf("state after update = %+v, want one Edit Parameter undo step", st)
	}
	call(t, r, s, "transaction.undo", "{}", &st)
	if d := getDetail(t, r, s, "h"); d.Comment != "" {
		t.Errorf("comment after undo = %q, want it reverted", d.Comment)
	}
}

// TestParameterDetailMutationsBroadcast checks the new mutation methods emit
// edit.committed (the replication seam) and the detail read does not.
func TestParameterDetailMutationsBroadcast(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	var methods []string
	sub := event.Subscribe(s.Events(), event.After, func(_ event.Context, e app.EditCommitted) event.Outcome {
		methods = append(methods, e.Method)
		return event.Continue()
	})
	defer sub.Cancel()

	call(t, r, s, "parameters.getDetail", `{"name":"width"}`, nil)
	call(t, r, s, "parameters.update", `{"name":"width","comment":"c"}`, nil)
	call(t, r, s, "parameters.setTolerance", `{"name":"width","mode":"symmetric","upper":"0.1 cm"}`, nil)
	call(t, r, s, "parameters.setExpressionList", `{"name":"width","expressions":["4 cm","6 cm"]}`, nil)
	call(t, r, s, "parameters.delete", `{"name":"width"}`, nil)

	want := []string{
		wire.MethodParametersUpdate, wire.MethodParametersSetTolerance,
		wire.MethodParametersSetExpressionList, wire.MethodParametersDelete,
	}
	if !slices.Equal(methods, want) {
		t.Errorf("edit.committed methods = %v, want %v (and none for getDetail)", methods, want)
	}
}

// approx absorbs float noise from unit conversion round-trips.
func approx(got, want float64) bool {
	const eps = 1e-9
	return got-want < eps && want-got < eps
}
