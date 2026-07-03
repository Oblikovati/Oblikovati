// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/text"
	"oblikovati.org/osfont"
)

// registerFontHandlers wires the font-picker methods: list the selectable faces, and set a
// sketch text entity's font (embedding the chosen face into the document) — ADR-0031.
func (r *Router) registerFontHandlers() {
	r.readOnly(wire.MethodFontsList, listFonts)
	r.mutating(wire.MethodSketchSetTextFont, "Edit Text", typedPart(setTextFont))
}

// listFonts returns every face the picker can offer: the application's bundled faces (source
// "embedded") plus the host's installed fonts (source "system", carrying the file path whose
// bytes get embedded on select). Needs no active document — it is a host capability.
func listFonts(_ *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	var faces []wire.FontFace
	for _, family := range text.EmbeddedFamilies() {
		faces = append(faces, wire.FontFace{Family: family, Source: "embedded"})
	}
	for _, f := range osfont.System() {
		faces = append(faces, wire.FontFace{Family: f.Family, Style: f.Style, Source: "system", Path: f.Path})
	}
	return json.Marshal(wire.ListFontsResult{Faces: faces})
}

// setTextFont embeds the chosen font into the active document and points the sketch text entity
// at it (by resource UUID), so the text/emboss is self-contained on reopen (ADR-0031).
func setTextFont(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SetTextFontArgs) (wire.SetTextFontResult, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.SetTextFontResult{}, err
	}
	tb, err := textBoxRef(sk, in.EntityID)
	if err != nil {
		return wire.SetTextFontResult{}, err
	}
	resource, family, err := embedTextFont(part, in)
	if err != nil {
		return wire.SetTextFontResult{}, err
	}
	tb.FontResource = resource
	if family != "" {
		tb.Family = family
	}
	return wire.SetTextFontResult{Resource: resource, Family: tb.Family}, nil
}

// embedTextFont turns the request into a document font resource: a host file's bytes (Path) or a
// bundled face recorded without bytes (Family), returning the resource UUID and family label.
func embedTextFont(part *compdef.PartComponentDefinition, in wire.SetTextFontArgs) (resource, family string, err error) {
	if in.Path != "" {
		data, err := osfont.ReadFont(in.Path)
		if err != nil {
			return "", "", fmt.Errorf("sketch.setTextFont: read %q: %w", in.Path, err)
		}
		return part.EmbedSystemFont(data, filepath.Base(in.Path)), fontFamilyOf(data), nil
	}
	if in.Family == "" {
		return "", "", fmt.Errorf("sketch.setTextFont: need a font path (system) or family (embedded)")
	}
	return part.UseEmbeddedFont(in.Family), in.Family, nil
}

// fontFamilyOf parses a font file's bytes for its family name (the picker label), or "" if it
// will not parse — the embed still succeeds, the label just falls back to the existing one.
func fontFamilyOf(data []byte) string {
	ft, err := text.Parse(data)
	if err != nil {
		return ""
	}
	return ft.Family()
}
