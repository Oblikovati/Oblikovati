// SPDX-License-Identifier: GPL-2.0-only

package theme

import "oblikovati/api/contract"

// Theme is one named UI color theme: a display name, a kind (light/dark/custom), and a
// full-snapshot [Palette]. It is the GPL implementation of [contract.Theme].
type Theme struct {
	name    string
	kind    Kind
	palette Palette
}

// compile-time proof that the implementation satisfies the public contract (ADR-0018).
var _ contract.Theme = (*Theme)(nil)

// New builds a theme from a name, kind, and palette. The palette is cloned so later
// edits to one theme never leak into another that shared the map.
func New(name string, kind Kind, palette Palette) *Theme {
	return &Theme{name: name, kind: kind, palette: palette.Clone()}
}

// Name is the display label, unique within a [Library].
func (t *Theme) Name() string { return t.name }

// Kind classifies the theme as a built-in (light/dark) or a user copy (custom).
func (t *Theme) Kind() Kind { return t.kind }

// Color returns the color bound to token, or the palette fallback if absent.
func (t *Theme) Color(token Token) Rgba { return t.palette.Color(token) }

// Palette returns the theme's color map (the editor reads it to populate color rows;
// callers must not mutate it — use [Theme.SetColor]).
func (t *Theme) Palette() Palette { return t.palette }

// SetColor recolors one token. It is a no-op on a built-in (light/dark), which is
// read-only by contract; only custom themes are editable (see [types.ThemeKind.Editable]).
func (t *Theme) SetColor(token Token, c Rgba) {
	if !t.kind.Editable() {
		return
	}
	t.palette[token] = c
}

// Duplicate returns an independent custom copy under newName — a full snapshot of this
// theme's colors (the user chose self-contained customs, ADR-0021), so the copy keeps
// its look even if a built-in's defaults later change.
func (t *Theme) Duplicate(newName string) *Theme {
	return &Theme{name: newName, kind: KindCustom, palette: t.palette.Clone()}
}
