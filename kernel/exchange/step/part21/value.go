// SPDX-License-Identifier: GPL-2.0-only

package part21

import (
	"fmt"
	"strconv"
)

// ValueKind discriminates a parsed parameter Value.
type ValueKind uint8

const (
	// ValRef references another entity by id (#123).
	ValRef ValueKind = iota
	// ValInt is an integer literal.
	ValInt
	// ValReal is a floating-point literal.
	ValReal
	// ValString is a decoded string literal.
	ValString
	// ValEnum is a dotted enumeration like .T. (stored without the dots: "T").
	ValEnum
	// ValList is a parenthesized list of values (possibly nested).
	ValList
	// ValNull is the '$' omitted-parameter marker.
	ValNull
	// ValDerived is the '*' inherited/derived marker.
	ValDerived
	// ValTyped is a typed parameter KEYWORD(args) appearing inside a parameter list
	// (e.g. a select type or an inline complex-entity component).
	ValTyped
)

// Value is one parsed Part 21 parameter. Only the field for Kind is meaningful.
// It is a sum type kept explicit (no any) so consumers switch on Kind.
type Value struct {
	Kind     ValueKind
	Ref      int     // ValRef
	Int      int64   // ValInt
	Real     float64 // ValReal
	Str      string  // ValString
	Enum     string  // ValEnum (dots stripped)
	List     []Value // ValList / ValTyped args
	Keyword  string  // ValTyped keyword
	position Token   // source token for error messages
}

// AsRef returns the referenced id, erroring when the value is not a reference.
func (v Value) AsRef() (int, error) {
	if v.Kind != ValRef {
		return 0, v.typeError("reference")
	}
	return v.Ref, nil
}

// AsFloat returns a real or integer value as float64, erroring otherwise.
func (v Value) AsFloat() (float64, error) {
	switch v.Kind {
	case ValReal:
		return v.Real, nil
	case ValInt:
		return float64(v.Int), nil
	default:
		return 0, v.typeError("number")
	}
}

// AsInt returns an integer value, erroring otherwise.
func (v Value) AsInt() (int, error) {
	if v.Kind != ValInt {
		return 0, v.typeError("integer")
	}
	return int(v.Int), nil
}

// AsString returns a string value, erroring otherwise.
func (v Value) AsString() (string, error) {
	if v.Kind != ValString {
		return "", v.typeError("string")
	}
	return v.Str, nil
}

// AsEnum returns an enumeration token (dots stripped), erroring otherwise.
func (v Value) AsEnum() (string, error) {
	if v.Kind != ValEnum {
		return "", v.typeError("enumeration")
	}
	return v.Enum, nil
}

// AsList returns a list's elements, erroring otherwise.
func (v Value) AsList() ([]Value, error) {
	if v.Kind != ValList {
		return nil, v.typeError("list")
	}
	return v.List, nil
}

// AsBool decodes a .T./.F. boolean enumeration.
func (v Value) AsBool() (bool, error) {
	e, err := v.AsEnum()
	if err != nil {
		return false, err
	}
	switch e {
	case "T":
		return true, nil
	case "F":
		return false, nil
	default:
		return false, fmt.Errorf("part21: expected .T./.F. boolean, got .%s. at %d:%d", e, v.position.Line, v.position.Column)
	}
}

// IsNull reports the '$' omitted marker.
func (v Value) IsNull() bool { return v.Kind == ValNull }

// typeError builds a consistent "wanted X, got Y" error citing the source token.
func (v Value) typeError(want string) error {
	return fmt.Errorf("part21: expected %s parameter, got %s at %d:%d",
		want, v.Kind, v.position.Line, v.position.Column)
}

// String renders the kind for diagnostics.
func (k ValueKind) String() string {
	switch k {
	case ValRef:
		return "ref"
	case ValInt:
		return "int"
	case ValReal:
		return "real"
	case ValString:
		return "string"
	case ValEnum:
		return "enum"
	case ValList:
		return "list"
	case ValNull:
		return "null"
	case ValDerived:
		return "derived"
	case ValTyped:
		return "typed"
	default:
		return strconv.Itoa(int(k))
	}
}
