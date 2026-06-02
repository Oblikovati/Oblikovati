// SPDX-License-Identifier: GPL-2.0-only

// Package attr is the extensible metadata side-channel: typed named attributes
// any model object can carry (grouped in sets, anchored by reference key so they
// survive recompute) and document-level property sets / iProperties that flow to
// BOMs and drawings (parametric-cad §11, architecture core/05).
//
// Values are a small closed set of typed variants rather than an untyped map — the
// COM "NameValueMap as everything" habit is replaced by [Value], a tagged union
// with typed constructors and accessors (no any, no map[string]interface{}).
package attr

import (
	"fmt"
	"math"
)

// ValueType is the type tag of an attribute/property [Value] — the modernized
// ValueTypeEnum. Values are stable: they are written into the persisted metadata.
type ValueType uint8

const (
	// Boolean is the zero value, but a zero Value still needs an explicit type to
	// be meaningful; use the constructors.
	Boolean ValueType = 0
	Integer ValueType = 1
	Double  ValueType = 2
	String  ValueType = 3
	Bytes   ValueType = 4
)

// String returns a stable name for diagnostics.
func (t ValueType) String() string {
	switch t {
	case Boolean:
		return "boolean"
	case Integer:
		return "integer"
	case Double:
		return "double"
	case String:
		return "string"
	case Bytes:
		return "bytes"
	default:
		return "unknown"
	}
}

// Value is a typed attribute/property value. Exactly one field is meaningful,
// selected by typ; the typed accessors enforce that, returning ok=false on a type
// mismatch so a wrong read is never silent.
type Value struct {
	typ ValueType
	b   bool
	i   int64
	f   float64
	s   string
	raw []byte
}

// BoolValue, IntValue, FloatValue, StringValue and BytesValue construct a Value of
// the corresponding type.
func BoolValue(v bool) Value     { return Value{typ: Boolean, b: v} }
func IntValue(v int64) Value     { return Value{typ: Integer, i: v} }
func FloatValue(v float64) Value { return Value{typ: Double, f: v} }
func StringValue(v string) Value { return Value{typ: String, s: v} }
func BytesValue(v []byte) Value  { return Value{typ: Bytes, raw: append([]byte(nil), v...)} }

// Type returns the value's type tag.
func (v Value) Type() ValueType { return v.typ }

// Bool/Int/Float/Str/Raw return the typed value and ok=true only when the type tag
// matches; otherwise the zero value and ok=false.
func (v Value) Bool() (bool, bool)     { return v.b, v.typ == Boolean }
func (v Value) Int() (int64, bool)     { return v.i, v.typ == Integer }
func (v Value) Float() (float64, bool) { return v.f, v.typ == Double }
func (v Value) Str() (string, bool)    { return v.s, v.typ == String }
func (v Value) Raw() ([]byte, bool) {
	if v.typ != Bytes {
		return nil, false
	}
	return append([]byte(nil), v.raw...), true
}

// Equal reports whether two values have the same type and contents.
func (v Value) Equal(o Value) bool {
	if v.typ != o.typ {
		return false
	}
	switch v.typ {
	case Boolean:
		return v.b == o.b
	case Integer:
		return v.i == o.i
	case Double:
		return v.f == o.f || (math.IsNaN(v.f) && math.IsNaN(o.f))
	case String:
		return v.s == o.s
	case Bytes:
		return string(v.raw) == string(o.raw)
	default:
		return false
	}
}

// String renders the value for diagnostics, tagged with its type.
func (v Value) String() string {
	switch v.typ {
	case Boolean:
		return fmt.Sprintf("bool(%t)", v.b)
	case Integer:
		return fmt.Sprintf("int(%d)", v.i)
	case Double:
		return fmt.Sprintf("double(%g)", v.f)
	case String:
		return fmt.Sprintf("string(%q)", v.s)
	case Bytes:
		return fmt.Sprintf("bytes(%d)", len(v.raw))
	default:
		return "unknown"
	}
}
