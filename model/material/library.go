// SPDX-License-Identifier: GPL-2.0-only

package material

import (
	"fmt"
	"strings"
)

// Library is the set of appearances and materials available to a session: the shipped
// built-ins, the active project's shared assets, and the assets embedded in open
// documents, keyed by id and kept in insertion order for stable display. A monotonic
// revision counter is bumped on any change that affects the rendered look (add, edit,
// duplicate, remove) so the head can refresh the viewport on the next frame — the same
// live-update mechanism as the theme library.
type Library struct {
	appearances map[string]*Appearance
	materials   map[string]*Material
	apprOrder   []string
	matOrder    []string
	revision    uint64
}

// NewLibrary returns a library seeded with the shipped built-in catalog.
func NewLibrary() *Library {
	lib := &Library{appearances: map[string]*Appearance{}, materials: map[string]*Material{}, revision: 1}
	seedBuiltins(lib)
	return lib
}

// Revision returns the change counter; compare it across frames to detect edits.
func (l *Library) Revision() uint64 { return l.revision }

// Appearances / Materials return the assets in insertion order. Callers must not mutate
// the slices.
func (l *Library) Appearances() []*Appearance {
	out := make([]*Appearance, 0, len(l.apprOrder))
	for _, id := range l.apprOrder {
		out = append(out, l.appearances[id])
	}
	return out
}

func (l *Library) Materials() []*Material {
	out := make([]*Material, 0, len(l.matOrder))
	for _, id := range l.matOrder {
		out = append(out, l.materials[id])
	}
	return out
}

// Appearance / Material look up an asset by id.
func (l *Library) Appearance(id string) (*Appearance, bool) { a, ok := l.appearances[id]; return a, ok }
func (l *Library) Material(id string) (*Material, bool)     { m, ok := l.materials[id]; return m, ok }

// appearanceOrNil returns the appearance for id, or nil for an empty/unknown id — the
// form the assignment resolver wants so it can fall through the precedence chain.
func (l *Library) appearanceOrNil(id string) *Appearance {
	if id == "" {
		return nil
	}
	return l.appearances[id]
}

// defaultAppearance returns the neutral built-in appearance, the resolver's last resort.
// It is always present (seeded by the built-in catalog).
func (l *Library) defaultAppearance() *Appearance { return l.appearances[DefaultAppearanceID] }

// AddAppearance / AddMaterial insert an asset (built-in seeding, library/document load),
// replacing any existing asset with the same id without disturbing display order. They do
// not bump the revision — loading is not a user edit.
func (l *Library) AddAppearance(a *Appearance) {
	if _, exists := l.appearances[a.id]; !exists {
		l.apprOrder = append(l.apprOrder, a.id)
	}
	l.appearances[a.id] = a
}

func (l *Library) AddMaterial(m *Material) {
	if _, exists := l.materials[m.id]; !exists {
		l.matOrder = append(l.matOrder, m.id)
	}
	l.materials[m.id] = m
}

// DuplicateAppearance copies base into a new editable asset of the given source under
// name, adds it, and bumps the revision. It errors on an unknown base or empty name.
func (l *Library) DuplicateAppearance(baseID, name string, source Source) (*Appearance, error) {
	base, ok := l.appearances[baseID]
	if !ok {
		return nil, fmt.Errorf("material: appearance base %q not found", baseID)
	}
	if name == "" {
		return nil, fmt.Errorf("material: appearance name is empty")
	}
	dup := base.duplicate(l.freshID(name, func(id string) bool { _, ok := l.appearances[id]; return ok }), name, source)
	l.AddAppearance(dup)
	l.revision++
	return dup, nil
}

// DuplicateMaterial copies base into a new editable asset under name.
func (l *Library) DuplicateMaterial(baseID, name string, source Source) (*Material, error) {
	base, ok := l.materials[baseID]
	if !ok {
		return nil, fmt.Errorf("material: material base %q not found", baseID)
	}
	if name == "" {
		return nil, fmt.Errorf("material: material name is empty")
	}
	dup := base.duplicate(l.freshID(name, func(id string) bool { _, ok := l.materials[id]; return ok }), name, source)
	l.AddMaterial(dup)
	l.revision++
	return dup, nil
}

// EditAppearance / EditMaterial replace an editable asset's spec and bump the revision.
// A no-op for an unknown id or a read-only built-in.
func (l *Library) EditAppearance(id string, spec AppearanceSpec) {
	if a, ok := l.appearances[id]; ok && a.Source().Editable() {
		a.SetSpec(spec)
		l.revision++
	}
}

func (l *Library) EditMaterial(id string, spec MaterialSpec) {
	if m, ok := l.materials[id]; ok && m.Source().Editable() {
		m.SetSpec(spec)
		l.revision++
	}
}

// RemoveAppearance / RemoveMaterial delete a non-built-in asset and bump the revision.
func (l *Library) RemoveAppearance(id string) error {
	a, ok := l.appearances[id]
	if !ok {
		return fmt.Errorf("material: appearance %q not found", id)
	}
	if !a.Source().Editable() {
		return fmt.Errorf("material: built-in appearance %q cannot be removed", id)
	}
	delete(l.appearances, id)
	l.apprOrder = removeID(l.apprOrder, id)
	l.revision++
	return nil
}

func (l *Library) RemoveMaterial(id string) error {
	m, ok := l.materials[id]
	if !ok {
		return fmt.Errorf("material: material %q not found", id)
	}
	if !m.Source().Editable() {
		return fmt.Errorf("material: built-in material %q cannot be removed", id)
	}
	delete(l.materials, id)
	l.matOrder = removeID(l.matOrder, id)
	l.revision++
	return nil
}

// freshID returns a slug of name that exists reports as free (suffixing -2, -3… on
// collision), so two "Brushed Aluminum" copies get distinct stable ids.
func (l *Library) freshID(name string, exists func(id string) bool) string {
	base := slug(name)
	id := base
	for n := 2; exists(id); n++ {
		id = fmt.Sprintf("%s-%d", base, n)
	}
	return id
}

// slug turns a display name into a lower-case, alphanumeric-with-dashes id.
func slug(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "asset"
	}
	return out
}

// removeID returns ids with the named one removed (order preserved).
func removeID(ids []string, id string) []string {
	out := ids[:0:0]
	for _, x := range ids {
		if x != id {
			out = append(out, x)
		}
	}
	return out
}
