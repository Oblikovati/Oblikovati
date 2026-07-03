// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/attr"
)

// The document iProperties surface (#156): list, read, and write a document's metadata sets —
// the standard Summary / Document Summary / Design Tracking sets plus user-defined custom
// properties. A document's content (part or assembly) owns the sets through propertyHolder;
// values cross the wire as the shared [types.Variant], converted to/from the model [attr.Value]
// here.

// propertyHolder is the content interface a document with iProperties satisfies — both the part
// and assembly definitions do.
type propertyHolder interface {
	Properties() *attr.PropertySets
}

// registerDocumentPropertyHandlers wires the documents.*Property* methods.
func (r *Router) registerDocumentPropertyHandlers() {
	r.readOnly(wire.MethodDocumentsListProperties, typed(listDocumentProperties))
	r.readOnly(wire.MethodDocumentsGetProperty, typed(getDocumentProperty))
	r.mutating(wire.MethodDocumentsSetProperty, "Set Property", typed(setDocumentProperty))
}

// documentProperties resolves the open document addressed by id to its property sets, erroring
// when the document is not open or its content carries no properties (a bare reference stub).
func documentProperties(s *app.Session, id uint64, method string) (*attr.PropertySets, error) {
	d, err := documentByID(s, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	holder, ok := d.Content().(propertyHolder)
	if !ok {
		return nil, fmt.Errorf("%s: document %d (%s) has no properties", method, id, d.DisplayName())
	}
	return holder.Properties(), nil
}

// listDocumentProperties returns every property across the document's sets.
func listDocumentProperties(s *app.Session, in wire.ListPropertiesArgs) (wire.ListPropertiesResult, error) {
	ps, err := documentProperties(s, in.Document, wire.MethodDocumentsListProperties)
	if err != nil {
		return wire.ListPropertiesResult{}, err
	}
	var infos []wire.PropertyInfo
	for _, set := range ps.Sets() {
		for _, p := range set.Properties() {
			infos = append(infos, propertyInfo(set.Name(), p))
		}
	}
	return wire.ListPropertiesResult{Properties: infos}, nil
}

// getDocumentProperty returns one property addressed by its set and name.
func getDocumentProperty(s *app.Session, in wire.GetPropertyArgs) (wire.PropertyResult, error) {
	ps, err := documentProperties(s, in.Document, wire.MethodDocumentsGetProperty)
	if err != nil {
		return wire.PropertyResult{}, err
	}
	set, ok := ps.Lookup(in.Set)
	if !ok {
		return wire.PropertyResult{}, fmt.Errorf("%s: no property set %q", wire.MethodDocumentsGetProperty, in.Set)
	}
	p, ok := set.Property(in.Name)
	if !ok {
		return wire.PropertyResult{}, fmt.Errorf("%s: no property %q in set %q", wire.MethodDocumentsGetProperty, in.Name, in.Set)
	}
	return wire.PropertyResult{Property: propertyInfo(in.Set, p)}, nil
}

// setDocumentProperty creates or replaces a property's value (creating a custom set on demand).
func setDocumentProperty(s *app.Session, in wire.SetPropertyArgs) (wire.PropertyResult, error) {
	ps, err := documentProperties(s, in.Document, wire.MethodDocumentsSetProperty)
	if err != nil {
		return wire.PropertyResult{}, err
	}
	if in.Name == "" {
		return wire.PropertyResult{}, fmt.Errorf("%s: a property name is required", wire.MethodDocumentsSetProperty)
	}
	p := ps.Set(in.Set).Put(in.Name, valueFromVariant(in.Value))
	return wire.PropertyResult{Property: propertyInfo(in.Set, p)}, nil
}

// propertyInfo renders a model property in set as a wire DTO.
func propertyInfo(set string, p *attr.Property) wire.PropertyInfo {
	from, _ := p.ExposedFromParameter()
	return wire.PropertyInfo{Set: set, Name: p.Name(), Value: variantFromValue(p.Value()), FromParameter: from}
}

// variantFromValue converts a model [attr.Value] to the wire [types.Variant].
func variantFromValue(v attr.Value) types.Variant {
	switch v.Type() {
	case attr.Boolean:
		b, _ := v.Bool()
		return types.BoolVariant(b)
	case attr.Integer:
		i, _ := v.Int()
		return types.IntegerVariant(i)
	case attr.Double:
		f, _ := v.Float()
		return types.DoubleVariant(f)
	case attr.Bytes:
		raw, _ := v.Raw()
		return types.BytesVariant(raw)
	default:
		s, _ := v.Str()
		return types.StringVariant(s)
	}
}

// valueFromVariant converts a wire [types.Variant] to the model [attr.Value].
func valueFromVariant(v types.Variant) attr.Value {
	switch v.Type() {
	case types.BooleanValue:
		b, _ := v.Bool()
		return attr.BoolValue(b)
	case types.IntegerValue:
		i, _ := v.Integer()
		return attr.IntValue(i)
	case types.DoubleValue:
		f, _ := v.Double()
		return attr.FloatValue(f)
	case types.ByteArrayValue:
		raw, _ := v.Bytes()
		return attr.BytesValue(raw)
	default:
		s, _ := v.Str()
		return attr.StringValue(s)
	}
}
