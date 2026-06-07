// SPDX-License-Identifier: GPL-2.0-only

package gopherlua

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"

	lua "github.com/yuin/gopher-lua"
)

// maxConvertDepth bounds nesting so a deeply recursive or cyclic table raises a clear
// error instead of overflowing the Go stack. 64 is well past any real DTO.
const maxConvertDepth = 64

// tableToJSON marshals a Lua value (typically the args table) to its JSON encoding,
// purely structurally: a Lua table becomes a JSON array when it is a 1..n integer
// sequence, otherwise a JSON object; numbers/strings/bools/nil map directly. This is
// what makes the bridge method-agnostic — any DTO that marshals as JSON round-trips
// with zero per-method code (ADR-0028 §3).
func tableToJSON(v lua.LValue) ([]byte, error) {
	g, err := luaToGo(v, 0)
	if err != nil {
		return nil, err
	}
	return json.Marshal(g)
}

// luaToGo converts one Lua value to a JSON-marshalable Go value, recursing into tables.
// It rejects nesting past maxConvertDepth (the cyclic/too-deep guard) and unsupported
// types (functions, userdata, threads) with a message naming the offending Lua type.
func luaToGo(v lua.LValue, depth int) (interface{}, error) {
	if depth > maxConvertDepth {
		return nil, fmt.Errorf("script: table nesting exceeds %d (cyclic or too deep)", maxConvertDepth)
	}
	switch t := v.(type) {
	case *lua.LNilType:
		return nil, nil
	case lua.LBool:
		return bool(t), nil
	case lua.LNumber:
		return float64(t), nil
	case lua.LString:
		return string(t), nil
	case *lua.LTable:
		return tableToGo(t, depth)
	default:
		return nil, fmt.Errorf("script: cannot convert Lua %s to JSON", v.Type().String())
	}
}

// tableToGo converts a Lua table to either a []interface{} (when it is a dense
// 1..n array) or a map[string]interface{} (otherwise), recursing through luaToGo.
func tableToGo(tb *lua.LTable, depth int) (interface{}, error) {
	if n := tb.Len(); n > 0 && isSequence(tb, n) {
		return sequenceToGo(tb, n, depth)
	}
	return mapToGo(tb, depth)
}

// isSequence reports whether tb is a dense 1..n integer-keyed array (so it should
// marshal as a JSON array, not an object).
func isSequence(tb *lua.LTable, n int) bool {
	count := 0
	tb.ForEach(func(k, _ lua.LValue) {
		if num, ok := k.(lua.LNumber); ok && num == lua.LNumber(math.Trunc(float64(num))) {
			count++
		}
	})
	return count == n
}

// sequenceToGo converts a 1..n Lua array to a Go slice.
func sequenceToGo(tb *lua.LTable, n, depth int) ([]interface{}, error) {
	out := make([]interface{}, 0, n)
	for i := 1; i <= n; i++ {
		g, err := luaToGo(tb.RawGetInt(i), depth+1)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

// mapToGo converts a Lua table to a Go map keyed by the string form of each key, so
// the result marshals to a JSON object. Non-string keys are stringified.
func mapToGo(tb *lua.LTable, depth int) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	var convErr error
	tb.ForEach(func(k, val lua.LValue) {
		if convErr != nil {
			return
		}
		g, err := luaToGo(val, depth+1)
		if err != nil {
			convErr = err
			return
		}
		out[k.String()] = g
	})
	return out, convErr
}

// jsonToTable decodes a JSON document into a fresh Lua value on l: objects become
// tables keyed by field, arrays become 1..n tables, scalars map directly. An empty or
// null input yields an empty table so a handler that returns no body still gives the
// script a table to index.
func jsonToTable(l *lua.LState, data []byte) (lua.LValue, error) {
	if len(data) == 0 {
		return l.NewTable(), nil
	}
	var g interface{}
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("script: decode result JSON: %w", err)
	}
	return goToLua(l, g), nil
}

// goToLua converts a decoded JSON value (from encoding/json) into a Lua value on l.
func goToLua(l *lua.LState, g interface{}) lua.LValue {
	switch t := g.(type) {
	case nil:
		return lua.LNil
	case bool:
		return lua.LBool(t)
	case float64:
		return lua.LNumber(t)
	case string:
		return lua.LString(t)
	case []interface{}:
		return sliceToTable(l, t)
	case map[string]interface{}:
		return objectToTable(l, t)
	default:
		return lua.LString(fmt.Sprintf("%v", t))
	}
}

// sliceToTable builds a 1..n Lua array table from a Go slice.
func sliceToTable(l *lua.LState, s []interface{}) *lua.LTable {
	tb := l.NewTable()
	for _, e := range s {
		tb.Append(goToLua(l, e))
	}
	return tb
}

// objectToTable builds a Lua table from a Go map, inserting keys in sorted order so
// the conversion is deterministic (repeatable tests, stable iteration).
func objectToTable(l *lua.LState, m map[string]interface{}) *lua.LTable {
	tb := l.NewTable()
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		tb.RawSetString(k, goToLua(l, m[k]))
	}
	return tb
}
