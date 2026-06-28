// SPDX-License-Identifier: GPL-2.0-only

package attr

import (
	"math"
	"testing"
)

func TestTypedAccessorsMatchOnlyTheirType(t *testing.T) {
	cases := []struct {
		v    Value
		typ  ValueType
		read func(Value) bool // returns ok from the matching accessor
	}{
		{BoolValue(true), Boolean, func(v Value) bool { b, ok := v.Bool(); return ok && b }},
		{IntValue(42), Integer, func(v Value) bool { i, ok := v.Int(); return ok && i == 42 }},
		{FloatValue(2.5), Double, func(v Value) bool { f, ok := v.Float(); return ok && f == 2.5 }},
		{StringValue("hi"), String, func(v Value) bool { s, ok := v.Str(); return ok && s == "hi" }},
		{BytesValue([]byte{1, 2}), Bytes, func(v Value) bool { b, ok := v.Raw(); return ok && len(b) == 2 }},
	}
	for _, c := range cases {
		if c.v.Type() != c.typ {
			t.Errorf("Type() = %v, want %v", c.v.Type(), c.typ)
		}
		if !c.read(c.v) {
			t.Errorf("matching accessor failed for %v", c.v)
		}
		// A wrong-type read must report ok=false rather than a silent zero.
		if _, ok := c.v.Int(); ok && c.typ != Integer {
			t.Errorf("Int() succeeded on a %v value", c.typ)
		}
	}
}

func TestValueEqualAndString(t *testing.T) {
	if !FloatValue(1.5).Equal(FloatValue(1.5)) {
		t.Error("equal doubles not equal")
	}
	if FloatValue(1.5).Equal(IntValue(1)) {
		t.Error("values of different types reported equal")
	}
	if BytesValue([]byte("ab")).Equal(BytesValue([]byte("ac"))) {
		t.Error("different byte values reported equal")
	}
	if StringValue("x").String() != `string("x")` {
		t.Errorf("String() = %q", StringValue("x").String())
	}
}

func TestValueTypeAndValueStrings(t *testing.T) {
	vtypes := map[ValueType]string{Boolean: "boolean", Integer: "integer", Double: "double", String: "string", Bytes: "bytes", ValueType(9): "valueType(?)"}
	for vt, want := range vtypes {
		if got := vt.String(); got != want {
			t.Errorf("ValueType(%d).String() = %q, want %q", vt, got, want)
		}
	}
	reprs := []struct {
		v    Value
		want string
	}{
		{BoolValue(true), "bool(true)"},
		{IntValue(7), "int(7)"},
		{FloatValue(2.5), "double(2.5)"},
		{BytesValue([]byte{1, 2, 3}), "bytes(3)"},
	}
	for _, c := range reprs {
		if got := c.v.String(); got != c.want {
			t.Errorf("Value.String() = %q, want %q", got, c.want)
		}
	}
	if (Value{typ: ValueType(9)}).String() != "unknown" {
		t.Error("unknown Value.String mismatch")
	}
}

func TestValueEqualAcrossTypes(t *testing.T) {
	if !BoolValue(true).Equal(BoolValue(true)) || BoolValue(true).Equal(BoolValue(false)) {
		t.Error("bool equality wrong")
	}
	if !IntValue(5).Equal(IntValue(5)) || IntValue(5).Equal(IntValue(6)) {
		t.Error("int equality wrong")
	}
	if !StringValue("a").Equal(StringValue("a")) || StringValue("a").Equal(StringValue("b")) {
		t.Error("string equality wrong")
	}
	if !BytesValue([]byte{1}).Equal(BytesValue([]byte{1})) {
		t.Error("bytes equality wrong")
	}
	// Two NaNs compare equal so a round trip is stable.
	nan := FloatValue(math.NaN())
	if !nan.Equal(FloatValue(math.NaN())) {
		t.Error("NaN doubles should compare equal for round-trip stability")
	}
	if _, ok := IntValue(1).Raw(); ok {
		t.Error("Raw() succeeded on a non-bytes value")
	}
}

func TestBytesValueIsCopied(t *testing.T) {
	src := []byte{1, 2, 3}
	v := BytesValue(src)
	src[0] = 9
	got, _ := v.Raw()
	if got[0] != 1 {
		t.Error("BytesValue shares its backing array with the caller")
	}
}
