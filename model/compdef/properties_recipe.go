// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"encoding/base64"
	"fmt"
	"strconv"

	"oblikovati.org/model/attr"
)

// Persistence of document iProperties in the part/assembly recipe (#156): each set property is
// stored as a small YAML record (set, name, typed value) so the .obk stays human-readable. The
// four standard sets are recreated empty by attr.NewPropertySets on load; only authored
// properties are persisted, then re-applied into their sets (a custom set name is recreated).

// propertyRecipe is the persisted form of one document property: its set and name, the value's
// type tag, and the value rendered to a string (base64 for bytes).
type propertyRecipe struct {
	Set   string `yaml:"set"`
	Name  string `yaml:"name"`
	Type  string `yaml:"type"`
	Value string `yaml:"value"`
}

// propertiesRecipeOf captures every authored property across all the document's sets, in set then
// insertion order. An empty document yields no records (the standard sets are implicit).
func propertiesRecipeOf(ps *attr.PropertySets) []propertyRecipe {
	var out []propertyRecipe
	for _, set := range ps.Sets() {
		for _, p := range set.Properties() {
			typ, val := valueToRecipe(p.Value())
			out = append(out, propertyRecipe{Set: set.Name(), Name: p.Name(), Type: typ, Value: val})
		}
	}
	return out
}

// applyPropertiesRecipe re-creates the persisted properties on ps (Put into the named set,
// creating a custom set if needed). An unparsable record is skipped rather than failing the load,
// so one bad value never blocks reopening the document.
func applyPropertiesRecipe(ps *attr.PropertySets, recs []propertyRecipe) {
	for _, r := range recs {
		v, err := valueFromRecipe(r.Type, r.Value)
		if err != nil {
			continue
		}
		ps.Set(r.Set).Put(r.Name, v)
	}
}

// valueToRecipe renders a typed value to its (type tag, string) recipe form.
func valueToRecipe(v attr.Value) (typ, val string) {
	switch v.Type() {
	case attr.Boolean:
		b, _ := v.Bool()
		return "boolean", strconv.FormatBool(b)
	case attr.Integer:
		i, _ := v.Int()
		return "integer", strconv.FormatInt(i, 10)
	case attr.Double:
		f, _ := v.Float()
		return "double", strconv.FormatFloat(f, 'g', -1, 64)
	case attr.String:
		s, _ := v.Str()
		return "string", s
	case attr.Bytes:
		raw, _ := v.Raw()
		return "bytes", base64.StdEncoding.EncodeToString(raw)
	default:
		return "string", ""
	}
}

// valueFromRecipe reconstructs a typed value from its (type tag, string) recipe form, erroring on
// an unknown tag or an unparsable value.
func valueFromRecipe(typ, val string) (attr.Value, error) {
	switch typ {
	case "boolean":
		b, err := strconv.ParseBool(val)
		return attr.BoolValue(b), err
	case "integer":
		i, err := strconv.ParseInt(val, 10, 64)
		return attr.IntValue(i), err
	case "double":
		f, err := strconv.ParseFloat(val, 64)
		return attr.FloatValue(f), err
	case "string":
		return attr.StringValue(val), nil
	case "bytes":
		raw, err := base64.StdEncoding.DecodeString(val)
		return attr.BytesValue(raw), err
	default:
		return attr.Value{}, fmt.Errorf("compdef: unknown property type %q", typ)
	}
}
