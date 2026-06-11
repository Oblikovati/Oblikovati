// SPDX-License-Identifier: GPL-2.0-only

package theme

import (
	"strings"
	"testing"

	"oblikovati.org/api/types"
)

// A custom theme must survive encode → decode bit-for-bit: name, every direct edit,
// and every derived-token edit (which only the snapshot section can carry).
func TestThemeFileRoundTrip(t *testing.T) {
	custom := DefaultDark().Duplicate("My Dark")
	accent := Rgba{R: 1, G: 0.25, B: 0, A: 1}
	hover := Rgba{R: 0.1, G: 0.9, B: 0.3, A: 0.8} // derived token: no Blender slot
	custom.SetColor(types.TokenChromeAccent, accent)
	custom.SetColor(types.TokenChromeControlHover, hover)

	data, err := encodeThemeXML(custom)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	got, err := decodeThemeXML(data, "fallback-ignored", KindCustom)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name() != "My Dark" {
		t.Errorf("name = %q, want \"My Dark\"", got.Name())
	}
	// Compare as hex: the file stores 8-bit channels, so float values quantize.
	for _, tok := range types.AllThemeTokens() {
		if got.Color(tok).Hex() != custom.Color(tok).Hex() {
			t.Errorf("token %q not round-tripped: %s != %s",
				tok, got.Color(tok).Hex(), custom.Color(tok).Hex())
		}
	}
}

// A plain Blender export (no oblikovati_tokens section) must load via the mapping
// alone, named after the caller's fallback (the file's base name) — this is what lets
// a user drop any Blender theme into the themes directory.
func TestForeignBlenderThemeLoads(t *testing.T) {
	got, err := decodeThemeXML(lightXML, "Downloaded Theme", KindCustom)
	if err != nil {
		t.Fatalf("decode plain blender file: %v", err)
	}
	if got.Name() != "Downloaded Theme" {
		t.Errorf("name = %q, want the fallback", got.Name())
	}
	if got.Kind() != KindCustom {
		t.Errorf("kind = %q, want custom", got.Kind())
	}
}

// Unmapped Blender attributes must survive a full edit/save cycle — fidelity is the
// point of keeping the document (ADR-0032).
func TestEncodePreservesUnmappedBlenderAttributes(t *testing.T) {
	custom := DefaultDark().Duplicate("Keeper")
	custom.SetColor(types.TokenChromeAccent, Rgba{R: 1, A: 1})
	data, err := encodeThemeXML(custom)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	// menu_shadow_fac is real Blender data no token maps; it must still be there.
	if !strings.Contains(string(data), "menu_shadow_fac=") {
		t.Error("unmapped attribute menu_shadow_fac was lost on save")
	}
}

// Saving twice must not stack snapshot sections.
func TestEncodeReplacesStaleSnapshot(t *testing.T) {
	custom := DefaultDark().Duplicate("Twice")
	if _, err := encodeThemeXML(custom); err != nil {
		t.Fatalf("first encode: %v", err)
	}
	data, err := encodeThemeXML(custom)
	if err != nil {
		t.Fatalf("second encode: %v", err)
	}
	if n := strings.Count(string(data), "<"+tokenSnapshotElem); n != 1 {
		t.Errorf("found %d %s sections, want exactly 1", n, tokenSnapshotElem)
	}
}

// A document-less theme (built via New, the test seam) has no Blender body to
// serialize; encode must refuse rather than emit garbage.
func TestEncodeRejectsDocumentlessTheme(t *testing.T) {
	bare := New("Bare", KindCustom, Palette{})
	if _, err := encodeThemeXML(bare); err == nil {
		t.Error("encode of a document-less theme should fail")
	}
}
