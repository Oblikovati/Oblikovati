// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"strings"
	"testing"

	"oblikovati.org/api/types"
)

// Audit I8 (#1631): format routing is one registry, not two hand-maintained switches. These tests
// lock the anti-drift guarantee — a format recognized by extension is also dispatchable, and the
// two can never diverge (the #1416 class: a menu-offered format the dispatch then refuses).

// TestEverySketchFormatHasADecoder is the parity that made the two-switch design fragile: every
// format that reports IsSketch must carry a registered DrawingDecoder, or a file the menu accepts
// falls through to "no drawing decoder" at import time.
func TestEverySketchFormatHasADecoder(t *testing.T) {
	t.Parallel()
	for _, f := range []types.ExchangeFormat{types.FormatDWG, types.FormatDXF, types.FormatPDF} {
		if !f.IsSketch() {
			t.Errorf("%q should report IsSketch (it is a drawing format)", f)
		}
		if _, ok := drawingDecoderFor(f); !ok {
			t.Errorf("sketch format %q has no registered DrawingDecoder — the menu would offer it and import would refuse it", f)
		}
	}
}

// TestDecodersAgreeWithTheirFormat: a decoder registered under a format must claim that format and a
// sketch format, and its extensions must resolve back to it — the round-trip that keeps FormatFromPath
// and dispatch in lockstep.
func TestDecodersAgreeWithTheirFormat(t *testing.T) {
	t.Parallel()
	for _, d := range []DrawingDecoder{dwgDrawingDecoder{}, dxfDrawingDecoder{}, pdfDrawingDecoder{}} {
		if !d.Format().IsSketch() {
			t.Errorf("decoder %T serves %q, which is not a sketch format", d, d.Format())
		}
		for _, ext := range d.Extensions() {
			if ext != strings.ToLower(ext) || !strings.HasPrefix(ext, ".") {
				t.Errorf("decoder %T extension %q must be lowercase and dot-prefixed", d, ext)
			}
			if f, ok := formatForExtension(ext); !ok || f != d.Format() {
				t.Errorf("extension %q resolves to (%q, %v), want %q", ext, f, ok, d.Format())
			}
		}
	}
}

// TestFormatFromPathRoutesEveryRegisteredExtension: FormatFromPath is now a registry lookup, so every
// extension the registry knows resolves (case-insensitively) and an unknown extension is rejected.
func TestFormatFromPathRoutesEveryRegisteredExtension(t *testing.T) {
	t.Parallel()
	for ext, want := range formatRoutes.byExtension {
		if got, ok := FormatFromPath("drawing" + strings.ToUpper(ext)); !ok || got != want {
			t.Errorf("FormatFromPath(%q) = (%q, %v), want %q", ext, got, ok, want)
		}
	}
	if f, ok := FormatFromPath("mystery.zzz"); ok {
		t.Errorf("FormatFromPath(unknown) = (%q, true), want not-found", f)
	}
}

// TestGLTFRouteHasExportCapability: the glTF route registers .glb with an export
// capability and no decoder — the export-only v1 contract (R1-2).
func TestGLTFRouteHasExportCapability(t *testing.T) {
	t.Parallel()
	r, ok := formatRoutes.byFormat[types.FormatGLTF]
	if !ok {
		t.Fatal("FormatGLTF has no registered route")
	}
	if r.exportBodies == nil {
		t.Error("glTF route has no export capability")
	}
	if r.decoder != nil {
		t.Error("glTF route carries a decoder; v1 is export-only")
	}
	if len(r.extensions) != 1 || r.extensions[0] != ".glb" {
		t.Errorf("glTF extensions = %v, want [.glb] (R1-2)", r.extensions)
	}
	if f, ok := formatForExtension(".glb"); !ok || f != types.FormatGLTF {
		t.Errorf(".glb resolves to (%q, %v), want FormatGLTF", f, ok)
	}
	if _, ok := formatForExtension(".gltf"); ok {
		t.Error(".gltf resolves to a format; v1 registers only .glb (R1-2)")
	}
}

// TestRegisterRejectsDuplicateFormatAndExtension is the decisive guard: a second route claiming an
// already-registered format or extension panics at construction, so a copy-paste registration cannot
// silently shadow another format.
func TestRegisterRejectsDuplicateFormatAndExtension(t *testing.T) {
	t.Parallel()
	assertPanics(t, "duplicate format", func() {
		s := &formatRouteSet{byFormat: map[types.ExchangeFormat]formatRoute{}, byExtension: map[string]types.ExchangeFormat{}}
		s.registerDrawing(dwgDrawingDecoder{})
		s.registerDrawing(dwgDrawingDecoder{}) // same format again
	})
	assertPanics(t, "duplicate extension", func() {
		s := &formatRouteSet{byFormat: map[types.ExchangeFormat]formatRoute{}, byExtension: map[string]types.ExchangeFormat{}}
		s.register(formatRoute{format: types.FormatDWG, extensions: []string{".shared"}, decoder: dwgDrawingDecoder{}})
		s.register(formatRoute{format: types.FormatDXF, extensions: []string{".shared"}, decoder: dxfDrawingDecoder{}})
	})
	assertPanics(t, "sketch format without decoder", func() {
		s := &formatRouteSet{byFormat: map[types.ExchangeFormat]formatRoute{}, byExtension: map[string]types.ExchangeFormat{}}
		s.register(formatRoute{format: types.FormatDWG, extensions: []string{".dwg"}}) // sketch format, no decoder
	})
}

// assertPanics fails the named case unless fn panics.
func assertPanics(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s: expected a panic, got none", name)
		}
	}()
	fn()
}
