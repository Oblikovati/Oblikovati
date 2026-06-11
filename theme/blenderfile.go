// SPDX-License-Identifier: GPL-2.0-only

package theme

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/theme/blenderxml"
)

// A theme file is a Blender theme XML document (`<bpy><Theme>…`), optionally extended
// with one `<oblikovati_tokens>` element as a sibling of <Theme> (ADR-0032). That
// element carries the theme's display name and a full token→hex snapshot:
//   - the snapshot keeps customs self-contained (ADR-0021) — derived tokens (hover,
//     disabled, grid emphasis) have no Blender slot, so their edits live only here;
//   - a plain Blender export (no extension) still loads: the mapping resolves every
//     token and the name falls back to the caller's (the file's base name);
//   - stripping the one element yields a stock Blender theme again.
const tokenSnapshotElem = "oblikovati_tokens"

// nameAttr holds the display name inside the snapshot element ("name" cannot collide
// with a token key: tokens are dotted, e.g. "chrome.text").
const nameAttr = "name"

// decodeThemeXML reads a theme file into a Theme: mapping first, snapshot overrides on
// top. fallbackName names a plain Blender export that has no snapshot element.
func decodeThemeXML(data []byte, fallbackName string, kind Kind) (*Theme, error) {
	doc, err := blenderxml.Parse(data)
	if err != nil {
		return nil, err
	}
	palette, err := resolvePalette(doc)
	if err != nil {
		return nil, err
	}
	name := fallbackName
	if snap := doc.Find(tokenSnapshotElem); snap != nil {
		name = applySnapshot(snap, palette, name)
	}
	if name == "" {
		return nil, fmt.Errorf("theme: file has no name and no fallback was given")
	}
	return &Theme{name: name, kind: kind, palette: palette, doc: doc}, nil
}

// applySnapshot overlays the snapshot's colors onto the mapped palette and returns the
// stored display name (or the fallback when the attribute is absent). A malformed hex
// in the snapshot is skipped — the mapped color underneath already covers the token.
func applySnapshot(snap *blenderxml.Node, palette Palette, fallbackName string) string {
	for _, tok := range types.AllThemeTokens() {
		hex, ok := snap.Attr(string(tok))
		if !ok {
			continue
		}
		if c, err := types.ParseHex(hex); err == nil {
			palette[tok] = c
		}
	}
	if name, ok := snap.Attr(nameAttr); ok && name != "" {
		return name
	}
	return fallbackName
}

// encodeThemeXML renders a custom theme to its file bytes: the (write-back-maintained)
// Blender body with the snapshot element rebuilt from the current palette. The element
// is replaced wholesale so stale colors from an earlier save never linger.
func encodeThemeXML(t *Theme) ([]byte, error) {
	if t.doc == nil {
		return nil, fmt.Errorf("theme: %q has no blender document to serialize", t.name)
	}
	t.doc.RemoveChild(tokenSnapshotElem)
	snap := blenderxml.NewElement(tokenSnapshotElem)
	snap.SetAttr(nameAttr, t.name)
	for _, tok := range types.AllThemeTokens() {
		snap.SetAttr(string(tok), t.palette.Color(tok).Hex())
	}
	t.doc.AppendChild(snap)
	return t.doc.Marshal()
}
