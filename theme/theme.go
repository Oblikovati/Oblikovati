// SPDX-License-Identifier: GPL-2.0-only

package theme

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/theme/blenderxml"
)

// Theme is one named UI color theme: a display name, a kind (light/dark/custom), a
// full-snapshot [Palette], and the Blender XML document it was decoded from (ADR-0032).
// The document is what persists — the palette is its resolved, render-ready view — and
// it keeps every Blender attribute the mapping does not consume, so a theme survives
// load → edit → save without losing fidelity. It is the GPL implementation of
// [contract.Theme].
type Theme struct {
	name    string
	kind    Kind
	palette Palette
	doc     *blenderxml.Node // nil only for palette-only themes built via New (tests)
}

// compile-time proof that the implementation satisfies the public contract (ADR-0018).
var _ contract.Theme = (*Theme)(nil)

// New builds a document-less theme from a name, kind, and palette (a test seam;
// file-backed themes come from the store's XML decoder). The palette is cloned so
// later edits to one theme never leak into another that shared the map.
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
// A direct token is also written through to its Blender attribute, so the saved file
// stays a faithful Blender theme (derived tokens persist via the snapshot section).
func (t *Theme) SetColor(token Token, c Rgba) {
	if !t.kind.Editable() {
		return
	}
	t.palette[token] = c
	if t.doc != nil {
		writeBackColor(t.doc, token, c)
	}
}

// Duplicate returns an independent custom copy under newName — a full snapshot of this
// theme's colors and Blender document (the user chose self-contained customs,
// ADR-0021), so the copy keeps its look even if a built-in's defaults later change.
func (t *Theme) Duplicate(newName string) *Theme {
	d := t.clone()
	d.name, d.kind = newName, KindCustom
	return d
}

// clone returns a deep copy (palette and document), same name and kind.
func (t *Theme) clone() *Theme {
	out := &Theme{name: t.name, kind: t.kind, palette: t.palette.Clone()}
	if t.doc != nil {
		out.doc = t.doc.Clone()
	}
	return out
}
