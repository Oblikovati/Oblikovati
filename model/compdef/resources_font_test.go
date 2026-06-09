// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/model/doc"
	"oblikovati.org/model/text"
)

// TestEmbedAndResolveFonts covers ADR-0031 font resources: an OS font is embedded by bytes
// (base64) and resolves to a parsed face; an app-provided font is recorded WITHOUT bytes and
// still resolves (the app supplies them); identical embeds dedup; and an unknown ref falls back
// to the family-named default so plain text still renders.
func TestEmbedAndResolveFonts(t *testing.T) {
	data, ok := text.EmbeddedFontBytes(text.DefaultFontFamily)
	if !ok {
		t.Fatal("no embedded font bytes for the fixture")
	}
	d := NewPartComponentDefinition()

	// OS font embedded by bytes.
	id := d.EmbedSystemFont(data, "MyFont.ttf")
	ft, err := d.Resolve(id)
	if err != nil || ft == nil {
		t.Fatalf("Resolve(embedded OS font) = %v, %v", ft, err)
	}
	if ft.Family() == "" {
		t.Error("resolved OS font has no family (parse failed?)")
	}
	if again := d.EmbedSystemFont(data, "MyFont.ttf"); again != id {
		t.Errorf("re-embedding identical bytes minted a new id %q (want dedup to %q)", again, id)
	}

	// App-provided font: a resource with NO bytes, resolved from the bundle.
	eid := d.UseEmbeddedFont(text.DefaultFontFamily)
	r, _ := d.Resource(eid)
	if r.Encoding != doc.EncodingEmbedded || len(r.Value) != 0 {
		t.Errorf("app-provided font resource = %+v, want embedded encoding with no bytes", r)
	}
	if ft2, err := d.Resolve(eid); err != nil || ft2 == nil {
		t.Fatalf("Resolve(app-provided font) = %v, %v", ft2, err)
	}
	if got := len(d.Resources()); got != 2 {
		t.Errorf("resource count = %d, want 2 (one OS + one app-provided)", got)
	}

	// An unknown reference is treated as a family name and falls back to the default face.
	if ft3, err := d.Resolve("Nonexistent Family"); err != nil || ft3 == nil {
		t.Fatalf("Resolve(unknown family) = %v, %v; want the default face", ft3, err)
	}
}
