// SPDX-License-Identifier: GPL-2.0-only

package theme

import "fmt"

// Library is the set of themes available in a session: the two built-ins followed by the
// user's customs, with one active selection. A monotonic revision counter is bumped on
// every change that affects the rendered look (select, edit, add, remove) so the head can
// cheaply detect "theme changed since I last applied it" and restyle on the next frame —
// the hook behind live preview (ADR-0021).
type Library struct {
	themes   []*Theme // built-ins first (Dark, Light), then customs in load order
	active   string   // name of the active theme
	revision uint64
}

// NewLibrary builds a library from the loaded customs and the persisted active-theme
// name. The built-ins are always present and first; an unknown or empty active name
// falls back to Dark, so a corrupt preference never leaves the UI unstyled.
func NewLibrary(customs []*Theme, active string) *Library {
	lib := &Library{themes: Builtins(), active: active, revision: 1}
	lib.themes = append(lib.themes, customs...)
	if _, ok := lib.find(active); !ok {
		lib.active = "Dark"
	}
	return lib
}

// Themes returns every theme in display order (built-ins then customs). Callers must not
// mutate the slice.
func (l *Library) Themes() []*Theme { return l.themes }

// Customs returns just the user themes — what [Store.Save] persists.
func (l *Library) Customs() []*Theme {
	var out []*Theme
	for _, t := range l.themes {
		if t.Kind() == KindCustom {
			out = append(out, t)
		}
	}
	return out
}

// Active returns the active theme (never nil; the constructor guarantees a valid active).
func (l *Library) Active() *Theme {
	t, _ := l.find(l.active)
	return t
}

// ActiveName returns the active theme's name (the value [Store] persists).
func (l *Library) ActiveName() string { return l.active }

// Revision returns the change counter; compare it to a previously-seen value to decide
// whether to re-apply the theme.
func (l *Library) Revision() uint64 { return l.revision }

// SetActive switches the active theme by name, bumping the revision. It is a no-op
// (no bump) for an unknown name or the already-active one.
func (l *Library) SetActive(name string) {
	if name == l.active {
		return
	}
	if _, ok := l.find(name); !ok {
		return
	}
	l.active = name
	l.revision++
}

// Duplicate creates a custom full-snapshot copy of base under newName, adds it, makes it
// active, and bumps the revision. It errors on an unknown base or a name already in use.
func (l *Library) Duplicate(base, newName string) (*Theme, error) {
	src, ok := l.find(base)
	if !ok {
		return nil, fmt.Errorf("theme: duplicate base %q not found", base)
	}
	if _, taken := l.find(newName); taken || newName == "" {
		return nil, fmt.Errorf("theme: name %q is empty or already in use", newName)
	}
	dup := src.Duplicate(newName)
	l.themes = append(l.themes, dup)
	l.active = newName
	l.revision++
	return dup, nil
}

// Remove deletes a custom theme by name, bumping the revision. It errors when the name is
// unknown or names a built-in (built-ins cannot be removed). Removing the active theme
// reselects Dark.
func (l *Library) Remove(name string) error {
	t, ok := l.find(name)
	if !ok {
		return fmt.Errorf("theme: remove %q not found", name)
	}
	if t.Kind() != KindCustom {
		return fmt.Errorf("theme: built-in %q cannot be removed", name)
	}
	l.themes = without(l.themes, name)
	if l.active == name {
		l.active = "Dark"
	}
	l.revision++
	return nil
}

// EditActiveColor recolors one token of the active theme and bumps the revision (the
// live-preview path). It is a no-op when the active theme is a built-in (read-only).
func (l *Library) EditActiveColor(token Token, c Rgba) {
	a := l.Active()
	if !a.Kind().Editable() {
		return
	}
	a.SetColor(token, c)
	l.revision++
}

// find returns the named theme and whether it exists.
func (l *Library) find(name string) (*Theme, bool) {
	for _, t := range l.themes {
		if t.Name() == name {
			return t, true
		}
	}
	return nil, false
}

// without returns themes with the named one removed (order preserved).
func without(themes []*Theme, name string) []*Theme {
	out := themes[:0:0]
	for _, t := range themes {
		if t.Name() != name {
			out = append(out, t)
		}
	}
	return out
}
