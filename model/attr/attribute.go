// SPDX-License-Identifier: GPL-2.0-only

package attr

// Attribute is a single named, typed value within an [AttributeSet].
type Attribute struct {
	name  string
	value Value
}

// Name returns the attribute name (unique within its set).
func (a *Attribute) Name() string { return a.name }

// Value returns the attribute's value.
func (a *Attribute) Value() Value { return a.value }

// SetValue replaces the attribute's value.
func (a *Attribute) SetValue(v Value) { a.value = v }

// ValueType returns the type tag of the attribute's value.
func (a *Attribute) ValueType() ValueType { return a.value.typ }

// AttributeSet is a named group of attributes — the unit of namespacing, so an
// add-in's private data lives under its own set name without colliding with others.
type AttributeSet struct {
	name  string
	attrs map[string]*Attribute
	order []string
}

func newAttributeSet(name string) *AttributeSet {
	return &AttributeSet{name: name, attrs: map[string]*Attribute{}}
}

// Name returns the set name.
func (s *AttributeSet) Name() string { return s.name }

// Put creates or replaces the attribute named name with value v and returns it.
func (s *AttributeSet) Put(name string, v Value) *Attribute {
	if existing, ok := s.attrs[name]; ok {
		existing.value = v
		return existing
	}
	a := &Attribute{name: name, value: v}
	s.attrs[name] = a
	s.order = append(s.order, name)
	return a
}

// Attribute returns the attribute named name, or false if absent.
func (s *AttributeSet) Attribute(name string) (*Attribute, bool) {
	a, ok := s.attrs[name]
	return a, ok
}

// Remove deletes the named attribute, reporting whether it existed.
func (s *AttributeSet) Remove(name string) bool {
	if _, ok := s.attrs[name]; !ok {
		return false
	}
	delete(s.attrs, name)
	for i, n := range s.order {
		if n == name {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	return true
}

// Attributes returns the set's attributes in insertion order.
func (s *AttributeSet) Attributes() []*Attribute {
	out := make([]*Attribute, 0, len(s.order))
	for _, n := range s.order {
		out = append(out, s.attrs[n])
	}
	return out
}

// Count returns the number of attributes in the set.
func (s *AttributeSet) Count() int { return len(s.attrs) }

// AttributeSets is the collection of attribute sets attached to one object.
type AttributeSets struct {
	sets  map[string]*AttributeSet
	order []string
}

func newAttributeSets() *AttributeSets {
	return &AttributeSets{sets: map[string]*AttributeSet{}}
}

// Set returns the named set, creating it if absent (get-or-create, the common case
// when an add-in stashes data).
func (ss *AttributeSets) Set(name string) *AttributeSet {
	if s, ok := ss.sets[name]; ok {
		return s
	}
	s := newAttributeSet(name)
	ss.sets[name] = s
	ss.order = append(ss.order, name)
	return s
}

// Lookup returns the named set, or false if it does not exist (no creation).
func (ss *AttributeSets) Lookup(name string) (*AttributeSet, bool) {
	s, ok := ss.sets[name]
	return s, ok
}

// Remove deletes a set, reporting whether it existed.
func (ss *AttributeSets) Remove(name string) bool {
	if _, ok := ss.sets[name]; !ok {
		return false
	}
	delete(ss.sets, name)
	for i, n := range ss.order {
		if n == name {
			ss.order = append(ss.order[:i], ss.order[i+1:]...)
			break
		}
	}
	return true
}

// Names returns the set names in insertion order.
func (ss *AttributeSets) Names() []string {
	out := make([]string, len(ss.order))
	copy(out, ss.order)
	return out
}

// Sets returns the sets in insertion order.
func (ss *AttributeSets) Sets() []*AttributeSet {
	out := make([]*AttributeSet, 0, len(ss.order))
	for _, n := range ss.order {
		out = append(out, ss.sets[n])
	}
	return out
}

// Count returns the number of sets.
func (ss *AttributeSets) Count() int { return len(ss.sets) }
