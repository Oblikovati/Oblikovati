// SPDX-License-Identifier: GPL-2.0-only

package material

import "fmt"

// OpenPBRAppearances returns the OpenPBR assets in insertion order. Callers must not
// mutate the slice.
func (l *Library) OpenPBRAppearances() []*OpenPBRAppearance {
	out := make([]*OpenPBRAppearance, 0, len(l.openpbrApprOrder))
	for _, id := range l.openpbrApprOrder {
		out = append(out, l.openpbrAppearances[id])
	}
	return out
}

// OpenPBRAppearance looks up an OpenPBR asset by id.
func (l *Library) OpenPBRAppearance(id string) (*OpenPBRAppearance, bool) {
	a, ok := l.openpbrAppearances[id]
	return a, ok
}

// DefaultOpenPBRAppearance returns the neutral built-in OpenPBR appearance, the OpenPBR
// resolver's last resort. It is always present (seeded by the built-in catalog).
func (l *Library) DefaultOpenPBRAppearance() *OpenPBRAppearance {
	return l.openpbrAppearances[DefaultOpenPBRAppearanceID]
}

// AddOpenPBRAppearance inserts an asset (built-in seeding, library/document load),
// replacing any existing asset with the same id without disturbing display order. It does
// not bump the revision — loading is not a user edit.
func (l *Library) AddOpenPBRAppearance(a *OpenPBRAppearance) {
	if _, exists := l.openpbrAppearances[a.id]; !exists {
		l.openpbrApprOrder = append(l.openpbrApprOrder, a.id)
	}
	l.openpbrAppearances[a.id] = a
}

// DuplicateOpenPBRAppearance copies base into a new editable asset of the given source
// under name, adds it, and bumps the revision. It errors on an unknown base or empty name.
func (l *Library) DuplicateOpenPBRAppearance(baseID, name string, source Source) (*OpenPBRAppearance, error) {
	base, ok := l.openpbrAppearances[baseID]
	if !ok {
		return nil, fmt.Errorf("material: openpbr appearance base %q not found", baseID)
	}
	if name == "" {
		return nil, fmt.Errorf("material: openpbr appearance name is empty")
	}
	exists := func(id string) bool { _, ok := l.openpbrAppearances[id]; return ok }
	dup := base.duplicate(l.freshID(name, exists), name, source)
	l.AddOpenPBRAppearance(dup)
	l.revision++
	return dup, nil
}

// EditOpenPBRAppearance replaces an editable asset's spec and bumps the revision. A no-op
// for an unknown id or a read-only built-in.
func (l *Library) EditOpenPBRAppearance(id string, spec OpenPBRAppearanceSpec) {
	if a, ok := l.openpbrAppearances[id]; ok && a.Source().Editable() {
		a.SetSpec(spec)
		l.revision++
	}
}

// RemoveOpenPBRAppearance deletes a non-built-in asset and bumps the revision.
func (l *Library) RemoveOpenPBRAppearance(id string) error {
	a, ok := l.openpbrAppearances[id]
	if !ok {
		return fmt.Errorf("material: openpbr appearance %q not found", id)
	}
	if !a.Source().Editable() {
		return fmt.Errorf("material: built-in openpbr appearance %q cannot be removed", id)
	}
	delete(l.openpbrAppearances, id)
	l.openpbrApprOrder = removeID(l.openpbrApprOrder, id)
	l.revision++
	return nil
}
