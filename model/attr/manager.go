// SPDX-License-Identifier: GPL-2.0-only

package attr

import "oblikovati/model/identity"

// AttributeManager holds the attribute sets of every object in a document, anchored
// by reference key. Anchoring by [identity.RefKey] — not by pointer or id — is what
// lets a face's attributes survive recompute: after the B-rep is rebuilt the same
// lineage re-mints an equal key, so the same attributes are found again (core/05).
//
// The map is keyed by the key's serialized bytes, so two equal RefKeys (e.g. one
// minted before a recompute and one after) address the same attributes.
type AttributeManager struct {
	byKey map[string]*AttributeSets
	order []string // insertion order of keys, for stable enumeration/serialization
}

// NewAttributeManager returns an empty manager.
func NewAttributeManager() *AttributeManager {
	return &AttributeManager{byKey: map[string]*AttributeSets{}}
}

// AttributeSets returns the attribute sets anchored to key, creating the anchor if
// none exists yet (get-or-create).
func (m *AttributeManager) AttributeSets(key identity.RefKey) *AttributeSets {
	k := string(key.Encode())
	if ss, ok := m.byKey[k]; ok {
		return ss
	}
	ss := newAttributeSets()
	m.byKey[k] = ss
	m.order = append(m.order, k)
	return ss
}

// Lookup returns the attribute sets anchored to key, or false if none.
func (m *AttributeManager) Lookup(key identity.RefKey) (*AttributeSets, bool) {
	ss, ok := m.byKey[string(key.Encode())]
	return ss, ok
}

// Remove deletes all attributes anchored to key, reporting whether any existed.
func (m *AttributeManager) Remove(key identity.RefKey) bool {
	k := string(key.Encode())
	if _, ok := m.byKey[k]; !ok {
		return false
	}
	delete(m.byKey, k)
	for i, existing := range m.order {
		if existing == k {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}
	return true
}

// Count returns the number of anchored objects.
func (m *AttributeManager) Count() int { return len(m.byKey) }

// AttributeHit is one result of [AttributeManager.FindAttributes]: the anchoring
// key plus the matching set and attribute.
type AttributeHit struct {
	Key       identity.RefKey
	Set       *AttributeSet
	Attribute *Attribute
}

// FindAttributes returns every attribute matching setName and attrName across all
// anchored objects, in key insertion order. It is the query/filter surface for
// e.g. "find every object tagged by add-in X". An empty attrName matches any
// attribute in the named set.
func (m *AttributeManager) FindAttributes(setName, attrName string) []AttributeHit {
	var hits []AttributeHit
	for _, k := range m.order {
		set, ok := m.byKey[k].Lookup(setName)
		if !ok {
			continue
		}
		hits = appendMatches(hits, k, set, attrName)
	}
	return hits
}

// appendMatches adds the matching attributes of one set to hits.
func appendMatches(hits []AttributeHit, encodedKey string, set *AttributeSet, attrName string) []AttributeHit {
	key, err := identity.DecodeKey([]byte(encodedKey))
	if err != nil {
		return hits // skip a corrupt anchor rather than fail the whole query
	}
	for _, a := range set.Attributes() {
		if attrName == "" || a.name == attrName {
			hits = append(hits, AttributeHit{Key: key, Set: set, Attribute: a})
		}
	}
	return hits
}
