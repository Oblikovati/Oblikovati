// SPDX-License-Identifier: GPL-2.0-only

package attr

import (
	"bytes"
	"encoding/binary"

	"oblikovati.org/model/param"
)

// The standard document property sets (iProperties). Their names are stable
// because they are persisted and referenced by drawings and BOMs.
const (
	SummaryInformation = "Summary Information"          // title, author, subject, keywords
	DocumentSummary    = "Document Summary Information" // category, company, manager
	DesignTracking     = "Design Tracking Properties"   // part number, material, mass, cost
	UserDefined        = "User Defined Properties"      // custom + parameter-exposed properties
)

// Property is one document property: a named, typed value. When it mirrors a
// parameter (the ExposedAsProperty bridge), source names that parameter.
type Property struct {
	name   string
	value  Value
	source string // parameter name this property was exposed from; "" if authored directly
}

// Name returns the property name.
func (p *Property) Name() string { return p.name }

// Value returns the property's value.
func (p *Property) Value() Value { return p.value }

// SetValue replaces the property's value.
func (p *Property) SetValue(v Value) { p.value = v }

// ExposedFromParameter returns the parameter name this property mirrors, and true,
// or ("", false) if the property was authored directly.
func (p *Property) ExposedFromParameter() (string, bool) {
	return p.source, p.source != ""
}

// PropertySet is a named group of properties (e.g. the summary set).
type PropertySet struct {
	name  string
	props map[string]*Property
	order []string
}

func newPropertySet(name string) *PropertySet {
	return &PropertySet{name: name, props: map[string]*Property{}}
}

// Name returns the set name.
func (s *PropertySet) Name() string { return s.name }

// Put creates or replaces the property named name with value v.
func (s *PropertySet) Put(name string, v Value) *Property {
	if existing, ok := s.props[name]; ok {
		existing.value = v
		return existing
	}
	p := &Property{name: name, value: v}
	s.props[name] = p
	s.order = append(s.order, name)
	return p
}

// Property returns the named property, or false if absent.
func (s *PropertySet) Property(name string) (*Property, bool) {
	p, ok := s.props[name]
	return p, ok
}

// Remove deletes the named property, reporting whether it existed.
func (s *PropertySet) Remove(name string) bool {
	if _, ok := s.props[name]; !ok {
		return false
	}
	delete(s.props, name)
	for i, n := range s.order {
		if n == name {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	return true
}

// Properties returns the set's properties in insertion order.
func (s *PropertySet) Properties() []*Property {
	out := make([]*Property, 0, len(s.order))
	for _, n := range s.order {
		out = append(out, s.props[n])
	}
	return out
}

// Count returns the number of properties.
func (s *PropertySet) Count() int { return len(s.props) }

// PropertySets is a document's collection of property sets (its iProperties).
type PropertySets struct {
	sets  map[string]*PropertySet
	order []string
}

// NewPropertySets returns a property collection pre-populated with the four
// standard sets, ready for a new document.
func NewPropertySets() *PropertySets {
	ps := &PropertySets{sets: map[string]*PropertySet{}}
	for _, name := range []string{SummaryInformation, DocumentSummary, DesignTracking, UserDefined} {
		ps.Set(name)
	}
	return ps
}

// Set returns the named set, creating it if absent.
func (ps *PropertySets) Set(name string) *PropertySet {
	if s, ok := ps.sets[name]; ok {
		return s
	}
	s := newPropertySet(name)
	ps.sets[name] = s
	ps.order = append(ps.order, name)
	return s
}

// Lookup returns the named set, or false if absent.
func (ps *PropertySets) Lookup(name string) (*PropertySet, bool) {
	s, ok := ps.sets[name]
	return s, ok
}

// Sets returns the sets in insertion order.
func (ps *PropertySets) Sets() []*PropertySet {
	out := make([]*PropertySet, 0, len(ps.order))
	for _, n := range ps.order {
		out = append(out, ps.sets[n])
	}
	return out
}

// ExposeParameter promotes a parameter to a custom (user-defined) property,
// mirroring its name and model value so it flows to BOMs and drawings — the
// ExposedAsProperty bridge (PBI-045). Re-exposing the same parameter updates the
// existing property in place.
func (ps *PropertySets) ExposeParameter(p *param.Parameter) *Property {
	prop := ps.Set(UserDefined).Put(p.Name(), FloatValue(p.ModelValue()))
	prop.source = p.Name()
	return prop
}

// EncodeProperties serializes the property sets for persistence. Layout mirrors the
// attribute codec, with each property carrying its source (parameter) name:
//
//	[nSets u32]{ str(name) [nProps u32]{ str(name) str(source) value } }
func (ps *PropertySets) EncodeProperties() []byte {
	buf := binary.LittleEndian.AppendUint32(nil, uint32(len(ps.order)))
	for _, set := range ps.Sets() {
		buf = appendString(buf, set.name)
		buf = binary.LittleEndian.AppendUint32(buf, uint32(len(set.order)))
		for _, p := range set.Properties() {
			buf = appendString(buf, p.name)
			buf = appendString(buf, p.source)
			buf = encodeValue(buf, p.value)
		}
	}
	return buf
}

// DecodeProperties reconstructs property sets from [PropertySets.EncodeProperties].
func DecodeProperties(data []byte) (*PropertySets, error) {
	r := bytes.NewReader(data)
	nSets, err := readU32(r)
	if err != nil {
		return nil, err
	}
	ps := &PropertySets{sets: map[string]*PropertySet{}}
	for range nSets {
		if err := decodePropertySet(r, ps); err != nil {
			return nil, err
		}
	}
	return ps, nil
}

func decodePropertySet(r *bytes.Reader, ps *PropertySets) error {
	name, err := readString(r)
	if err != nil {
		return err
	}
	set := ps.Set(name)
	nProps, err := readU32(r)
	if err != nil {
		return err
	}
	for range nProps {
		propName, err := readString(r)
		if err != nil {
			return err
		}
		source, err := readString(r)
		if err != nil {
			return err
		}
		v, err := decodeValue(r)
		if err != nil {
			return err
		}
		p := set.Put(propName, v)
		p.source = source
	}
	return nil
}
