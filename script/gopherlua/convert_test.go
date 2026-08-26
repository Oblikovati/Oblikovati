// SPDX-License-Identifier: GPL-2.0-only

package gopherlua

import (
	"encoding/json"
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

// roundTrip decodes json into a Lua value and back, so a test asserts the structural
// table↔JSON↔table conversion is lossless (the property the generic bridge relies on).
func roundTrip(t *testing.T, jsonIn string) string {
	t.Helper()
	l := lua.NewState(lua.Options{SkipOpenLibs: true})
	defer l.Close()
	v, err := jsonToTable(l, []byte(jsonIn))
	if err != nil {
		t.Fatalf("jsonToTable(%q): %v", jsonIn, err)
	}
	out, err := tableToJSON(v)
	if err != nil {
		t.Fatalf("tableToJSON: %v", err)
	}
	return string(out)
}

func TestConvertRoundTripsNestedShapes(t *testing.T) {
	// NOTE: JSON null is intentionally NOT round-tripped — Lua cannot store nil in a
	// table (assigning nil deletes the key), so a null field becomes an absent field.
	// That is a documented property of any Lua bridge, asserted separately below.
	cases := []string{
		`{"a":1,"b":"two","c":true}`,
		`{"nested":{"x":[1,2,3],"y":{"z":false}}}`,
		`[1,2,3]`,
		`{"arr":[{"k":1},{"k":2}],"flag":true}`,
	}
	for _, in := range cases {
		got := roundTrip(t, in)
		if !jsonEqual(t, in, got) {
			t.Errorf("round-trip changed shape: in=%s out=%s", in, got)
		}
	}
}

func TestConvertNullBecomesAbsentField(t *testing.T) {
	// Lua tables cannot hold nil, so a JSON null key drops out on round-trip. Pin the
	// behaviour so a future change to convert.go is a conscious decision.
	got := roundTrip(t, `{"present":1,"gone":null}`)
	if !jsonEqual(t, `{"present":1}`, got) {
		t.Errorf("null field should drop, got %s", got)
	}
}

func TestConvertEmptyInputYieldsTable(t *testing.T) {
	l := lua.NewState(lua.Options{SkipOpenLibs: true})
	defer l.Close()
	v, err := jsonToTable(l, nil)
	if err != nil {
		t.Fatalf("jsonToTable(nil): %v", err)
	}
	if v.Type() != lua.LTTable {
		t.Fatalf("empty input: want table, got %s", v.Type())
	}
}

func TestConvertRejectsCyclicTable(t *testing.T) {
	l := lua.NewState(lua.Options{SkipOpenLibs: true})
	defer l.Close()
	tb := l.NewTable()
	tb.RawSetString("self", tb) // cycle
	_, err := tableToJSON(tb)
	if err == nil {
		t.Fatal("expected an error for a cyclic table")
	}
	if !strings.Contains(err.Error(), "nesting") {
		t.Errorf("error should name the nesting limit, got %q", err.Error())
	}
}

func TestConvertRejectsFunctionValue(t *testing.T) {
	l := lua.NewState(lua.Options{SkipOpenLibs: true})
	defer l.Close()
	tb := l.NewTable()
	tb.RawSetString("fn", l.NewFunction(func(*lua.LState) int { return 0 }))
	_, err := tableToJSON(tb)
	if err == nil {
		t.Fatal("expected an error converting a function value")
	}
	if !strings.Contains(err.Error(), "function") {
		t.Errorf("error should name the offending Lua type, got %q", err.Error())
	}
}

// jsonEqual compares two JSON documents structurally (key order / whitespace agnostic).
func jsonEqual(t *testing.T, a, b string) bool {
	t.Helper()
	var va, vb any
	if err := json.Unmarshal([]byte(a), &va); err != nil {
		t.Fatalf("unmarshal %q: %v", a, err)
	}
	if err := json.Unmarshal([]byte(b), &vb); err != nil {
		t.Fatalf("unmarshal %q: %v", b, err)
	}
	ba, _ := json.Marshal(va)
	bb, _ := json.Marshal(vb)
	return string(ba) == string(bb)
}
