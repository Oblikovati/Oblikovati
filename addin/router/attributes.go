// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"sort"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/attr"
	"oblikovati.org/model/identity"
)

// Add-in attribute sets (#155): named, typed values an add-in attaches to a document and that
// persist with it. This surface targets the document itself — its sets are anchored under
// [identity.DocumentKey] in the document's [attr.AttributeManager] — and values cross the wire as
// the shared [types.Variant], converted to/from the model [attr.Value] by the document-properties
// helpers (variantFromValue / valueFromVariant).

// registerAttributeHandlers wires the attributes.* methods.
func (r *Router) registerAttributeHandlers() {
	r.mutating(wire.MethodAttributesSet, "Set Attribute", setAttribute)
	r.readOnly(wire.MethodAttributesGet, getAttribute)
	r.readOnly(wire.MethodAttributesList, listAttributes)
	r.readOnly(wire.MethodAttributesListSets, listAttributeSets)
	r.mutating(wire.MethodAttributesDelete, "Delete Attribute", deleteAttribute)
	r.readOnly(wire.MethodAttributesFind, findByAttribute)
}

// targetKey resolves a wire target string to the reference key that anchors its attributes: the
// document itself when empty, otherwise the entity addressed by its (opaque, external) reference
// key — the body/face/edge key an add-in received from body.list / model.referenceKeys.
func targetKey(target string) identity.RefKey {
	if target == "" {
		return identity.DocumentKey()
	}
	return identity.ExternalKey([]byte(target))
}

// targetString renders an anchor key back to the wire target an add-in addressed it by: empty for
// the document, the external reference key for an entity anchor.
func targetString(key identity.RefKey) string {
	if ref, ok := key.ExternalRef(); ok {
		return string(ref)
	}
	return ""
}

// targetAttributeSets resolves the attribute sets anchored to target (empty = the document) on the
// open document.
func targetAttributeSets(s *app.Session, id uint64, target, method string) (*attr.AttributeSets, error) {
	d, err := documentByID(s, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	return d.Attributes().AttributeSets(targetKey(target)), nil
}

// setAttribute creates or replaces the named attribute in the set with the typed value.
func setAttribute(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.SetAttributeArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if in.Set == "" || in.Name == "" {
		return nil, fmt.Errorf("%s: a set and a name are required", wire.MethodAttributesSet)
	}
	ss, err := targetAttributeSets(s, in.Document, in.Target, wire.MethodAttributesSet)
	if err != nil {
		return nil, err
	}
	a := ss.Set(in.Set).Put(in.Name, valueFromVariant(in.Value))
	return json.Marshal(wire.AttributeResult{Attribute: attributeInfo(in.Set, in.Target, a), Found: true})
}

// getAttribute reads one attribute by set and name; Found is false when it is absent.
func getAttribute(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.GetAttributeArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	ss, err := targetAttributeSets(s, in.Document, in.Target, wire.MethodAttributesGet)
	if err != nil {
		return nil, err
	}
	set, ok := ss.Lookup(in.Set)
	if !ok {
		return json.Marshal(notFoundAttribute(in.Set, in.Name, in.Target))
	}
	a, ok := set.Attribute(in.Name)
	if !ok {
		return json.Marshal(notFoundAttribute(in.Set, in.Name, in.Target))
	}
	return json.Marshal(wire.AttributeResult{Attribute: attributeInfo(in.Set, in.Target, a), Found: true})
}

// listAttributes returns every attribute on the document, or only those in Set when it is set.
func listAttributes(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.ListAttributesArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if in.AllTargets {
		d, err := documentByID(s, in.Document)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", wire.MethodAttributesList, err)
		}
		var infos []wire.AttributeInfo
		for _, anchor := range d.Attributes().Anchors() {
			infos = appendSetInfos(infos, anchor.Sets, in.Set, targetString(anchor.Key))
		}
		return json.Marshal(wire.ListAttributesResult{Attributes: infos})
	}
	ss, err := targetAttributeSets(s, in.Document, in.Target, wire.MethodAttributesList)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.ListAttributesResult{Attributes: appendSetInfos(nil, ss, in.Set, in.Target)})
}

// listAttributeSets returns the document's attribute set names, sorted.
func listAttributeSets(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.ListAttributeSetsArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	ss, err := targetAttributeSets(s, in.Document, "", wire.MethodAttributesListSets)
	if err != nil {
		return nil, err
	}
	names := ss.Names()
	sort.Strings(names)
	return json.Marshal(wire.ListAttributeSetsResult{Sets: names})
}

// deleteAttribute removes the named attribute, or the whole set when Name is empty, reporting how
// many attributes were removed.
func deleteAttribute(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.DeleteAttributeArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	ss, err := targetAttributeSets(s, in.Document, in.Target, wire.MethodAttributesDelete)
	if err != nil {
		return nil, err
	}
	set, ok := ss.Lookup(in.Set)
	if !ok {
		return json.Marshal(wire.DeleteAttributeResult{Removed: 0})
	}
	if in.Name == "" {
		removed := set.Count()
		ss.Remove(in.Set)
		return json.Marshal(wire.DeleteAttributeResult{Removed: removed})
	}
	removed := 0
	if set.Remove(in.Name) {
		removed = 1
	}
	return json.Marshal(wire.DeleteAttributeResult{Removed: removed})
}

// findByAttribute locates the open documents carrying an attribute in Set (optionally by Name).
func findByAttribute(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.FindByAttributeArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if in.Set == "" {
		return nil, fmt.Errorf("%s: a set is required", wire.MethodAttributesFind)
	}
	var matches []wire.AttributeMatch
	for _, d := range s.Workspace().Documents() {
		for _, h := range d.Attributes().FindAttributes(in.Set, in.Name) {
			matches = append(matches, wire.AttributeMatch{
				Document:  uint64(d.ID()),
				Attribute: attributeInfo(h.Set.Name(), targetString(h.Key), h.Attribute),
			})
		}
	}
	return json.Marshal(wire.FindByAttributeResult{Matches: matches})
}

// appendSetInfos appends one anchor's attributes (filtered to setFilter when non-empty), each
// stamped with target, to infos.
func appendSetInfos(infos []wire.AttributeInfo, ss *attr.AttributeSets, setFilter, target string) []wire.AttributeInfo {
	for _, set := range ss.Sets() {
		if setFilter != "" && set.Name() != setFilter {
			continue
		}
		for _, a := range set.Attributes() {
			infos = append(infos, attributeInfo(set.Name(), target, a))
		}
	}
	return infos
}

// attributeInfo renders a model attribute in set, anchored to target, as a wire DTO.
func attributeInfo(set, target string, a *attr.Attribute) wire.AttributeInfo {
	return wire.AttributeInfo{Set: set, Name: a.Name(), Value: variantFromValue(a.Value()), Target: target}
}

// notFoundAttribute is the get reply for an absent attribute: Found=false echoing the requested
// set/name/target with an empty (but well-typed, so it serializes) value the caller ignores.
func notFoundAttribute(set, name, target string) wire.AttributeResult {
	return wire.AttributeResult{Attribute: wire.AttributeInfo{Set: set, Name: name, Value: types.StringVariant(""), Target: target}, Found: false}
}
