// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
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
	r.handlers[wire.MethodDocumentsListProperties] = listDocumentProperties
	r.handlers[wire.MethodDocumentsGetProperty] = getDocumentProperty
	r.handlers[wire.MethodDocumentsSetProperty] = setDocumentProperty
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
func listDocumentProperties(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.ListPropertiesArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	ps, err := documentProperties(s, in.Document, wire.MethodDocumentsListProperties)
	if err != nil {
		return nil, err
	}
	var infos []wire.PropertyInfo
	for _, set := range ps.Sets() {
		for _, p := range set.Properties() {
			infos = append(infos, propertyInfo(set.Name(), p))
		}
	}
	return json.Marshal(wire.ListPropertiesResult{Properties: infos})
}

// getDocumentProperty returns one property addressed by its set and name.
func getDocumentProperty(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.GetPropertyArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	ps, err := documentProperties(s, in.Document, wire.MethodDocumentsGetProperty)
	if err != nil {
		return nil, err
	}
	set, ok := ps.Lookup(in.Set)
	if !ok {
		return nil, fmt.Errorf("%s: no property set %q", wire.MethodDocumentsGetProperty, in.Set)
	}
	p, ok := set.Property(in.Name)
	if !ok {
		return nil, fmt.Errorf("%s: no property %q in set %q", wire.MethodDocumentsGetProperty, in.Name, in.Set)
	}
	return json.Marshal(wire.PropertyResult{Property: propertyInfo(in.Set, p)})
}

// setDocumentProperty creates or replaces a property's value (creating a custom set on demand).
func setDocumentProperty(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.SetPropertyArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	ps, err := documentProperties(s, in.Document, wire.MethodDocumentsSetProperty)
	if err != nil {
		return nil, err
	}
	if in.Name == "" {
		return nil, fmt.Errorf("%s: a property name is required", wire.MethodDocumentsSetProperty)
	}
	p := ps.Set(in.Set).Put(in.Name, valueFromVariant(in.Value))
	return json.Marshal(wire.PropertyResult{Property: propertyInfo(in.Set, p)})
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
