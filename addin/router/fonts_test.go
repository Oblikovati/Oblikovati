// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"oblikovati/addin/modelaccess"
	"oblikovati/api/wire"
	"oblikovati/math"
	"oblikovati/model/sketch"
	"oblikovati/model/text"
)

// TestFontsListIncludesEmbedded checks fonts.list always offers the application's bundled face
// (the deterministic part; host fonts vary, so we don't assert on them).
func TestFontsListIncludesEmbedded(t *testing.T) {
	r, s := emptyPartSession(t)
	resp, err := r.Handle(s, wire.MethodFontsList, rawJSON(t, struct{}{}))
	if err != nil {
		t.Fatalf("fonts.list: %v", err)
	}
	var out wire.ListFontsResult
	if err := json.Unmarshal(resp, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !hasEmbeddedFamily(out.Faces, text.DefaultFontFamily) {
		t.Errorf("fonts.list missing the bundled face %q: %+v", text.DefaultFontFamily, out.Faces)
	}
}

// TestSetTextFontEmbedsSystemFont sets a text entity's font from a (temp) system font file and
// checks the document gains a base64 TrueTypeFont resource the entity now cites.
func TestSetTextFontEmbedsSystemFont(t *testing.T) {
	r, s := emptyPartSession(t)
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatalf("active part: %v", err)
	}
	sk := part.Sketches().Add(sketch.XYPlane())
	tb := sk.TextBoxes().Add(math.P2(0, 0), "HELLO", 1, 0, sketch.TextLeft)

	// A "system" font file built from the vendored face (host fonts vary; ADR-0031 test rule).
	data, _ := text.EmbeddedFontBytes(text.DefaultFontFamily)
	path := filepath.Join(t.TempDir(), "MyFont.ttf")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	resp, err := r.Handle(s, wire.MethodSketchSetTextFont, rawJSON(t, wire.SetTextFontArgs{
		SketchIndex: 0, EntityID: uint64(tb.EntityID()), Path: path,
	}))
	if err != nil {
		t.Fatalf("sketch.setTextFont: %v", err)
	}
	var out wire.SetTextFontResult
	if err := json.Unmarshal(resp, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Resource == "" || tb.FontResource != out.Resource {
		t.Fatalf("entity font resource = %q, result = %q; want them set + equal", tb.FontResource, out.Resource)
	}
	r2, ok := part.Resource(out.Resource)
	if !ok || r2.Type != "TrueTypeFont" || r2.Encoding != "base64" || len(r2.Value) == 0 {
		t.Errorf("embedded font resource = %+v, want a base64 TrueTypeFont with bytes", r2)
	}
	if ft, err := part.Resolve(out.Resource); err != nil || ft == nil {
		t.Errorf("Resolve(font resource) = %v, %v", ft, err)
	}
}

// TestSetTextFontUsesEmbeddedFace records a bundled face as a bytes-less embedded resource.
func TestSetTextFontUsesEmbeddedFace(t *testing.T) {
	r, s := emptyPartSession(t)
	part, _ := modelaccess.ActivePart(s)
	sk := part.Sketches().Add(sketch.XYPlane())
	tb := sk.TextBoxes().Add(math.P2(0, 0), "HI", 1, 0, sketch.TextLeft)

	resp, err := r.Handle(s, wire.MethodSketchSetTextFont, rawJSON(t, wire.SetTextFontArgs{
		SketchIndex: 0, EntityID: uint64(tb.EntityID()), Family: text.DefaultFontFamily,
	}))
	if err != nil {
		t.Fatalf("sketch.setTextFont: %v", err)
	}
	var out wire.SetTextFontResult
	_ = json.Unmarshal(resp, &out)
	res, ok := part.Resource(out.Resource)
	if !ok || res.Encoding != "embedded" || len(res.Value) != 0 || res.Origin != text.DefaultFontFamily {
		t.Errorf("embedded-face resource = %+v, want encoding=embedded, no bytes, origin=%q", res, text.DefaultFontFamily)
	}
}

func hasEmbeddedFamily(faces []wire.FontFace, family string) bool {
	for _, f := range faces {
		if f.Source == "embedded" && f.Family == family {
			return true
		}
	}
	return false
}

func rawJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
