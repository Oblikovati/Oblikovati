// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"path/filepath"
	"testing"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// TestEvcovExportDXFWritesActiveSketch drives export.dxf: the seeded sketch (a closed 4-line
// rectangle) is entered so the session has an active 2D sketch, then exported to a temp .dxf —
// the reply must report the four written curves (exercises the converted exportDXF handler).
func TestEvcovExportDXFWritesActiveSketch(t *testing.T) {
	r, s := seededSession(t)
	evcovEnterFirstSketch(t, s)
	out := filepath.Join(t.TempDir(), "out.dxf")
	var res wire.ExportDXFResult
	call(t, r, s, "export.dxf", mustJSON(t, wire.ExportDXFArgs{Path: out, Version: "r2000"}), &res)
	if res.EntityCount != 4 {
		t.Fatalf("export.dxf entityCount = %d, want 4 (rectangle edges)", res.EntityCount)
	}
}

// evcovEnterFirstSketch makes the active part's first sketch the session's active 2D sketch,
// the state the head sets when a user opens a sketch for edit.
func evcovEnterFirstSketch(t *testing.T, s *app.Session) {
	t.Helper()
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatalf("active part: %v", err)
	}
	s.EnterSketch(part.Sketches().Item(0))
}

// TestEvcovImportMissingFileErrors drives the import.dwg/dxf/pdf handlers' error branch: a
// non-existent path must surface a wrapped error from each converted handler.
func TestEvcovImportMissingFileErrors(t *testing.T) {
	r, s := seededSession(t)
	missing := filepath.Join(t.TempDir(), "nope")
	for method, args := range evcovImportCases(missing) {
		if _, err := r.Handle(s, method, []byte(args)); err == nil {
			t.Errorf("%s on missing file returned nil error, want failure", method)
		}
	}
}

// evcovImportCases pairs each import method with a request pointing at a missing file.
func evcovImportCases(path string) map[string]string {
	return map[string]string{
		"import.dwg": `{"path":"` + path + `.dwg"}`,
		"import.dxf": `{"path":"` + path + `.dxf"}`,
		"import.pdf": `{"path":"` + path + `.pdf"}`,
	}
}

// TestEvcovViewportDebugToggles drives viewport.setNormalDebug and viewport.setMeshColors: each
// converted handler must echo the requested state back in its reply.
func TestEvcovViewportDebugToggles(t *testing.T) {
	r, s := seededSession(t)
	var nd wire.NormalDebugResult
	call(t, r, s, "viewport.setNormalDebug", `{"on":true}`, &nd)
	if !nd.On {
		t.Fatalf("setNormalDebug reply = %+v, want On=true", nd)
	}
	var mc wire.MeshColorsResult
	call(t, r, s, "viewport.setMeshColors", `{"on":true,"perTriangle":true}`, &mc)
	if !mc.On || !mc.PerTriangle {
		t.Fatalf("setMeshColors reply = %+v, want On+PerTriangle true", mc)
	}
}
