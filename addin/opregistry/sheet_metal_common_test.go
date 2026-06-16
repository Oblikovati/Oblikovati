// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"math"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
)

// seedSheetMetalSheet builds a sheet-metal part, lays down a base wall, and returns the
// session plus a top-edge reference key — the common fixture the bend-family operation tests
// (flange/hem) flange/hem/bend from.
func seedSheetMetalSheet(t *testing.T) (*app.Session, string) {
	t.Helper()
	s := sheetMetalProfiledPart(t)
	if _, err := apply(t, s, "sheetMetalFace", `{"sketchIndex":0}`); err != nil {
		t.Fatalf("seed face: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	return s, topEdgeKey(t, def)
}

// topEdgeKey returns the reference key of a top-face edge of the part's first body — a
// deterministic edge to fold from.
func topEdgeKey(t *testing.T, def *compdef.PartComponentDefinition) string {
	t.Helper()
	if def.SurfaceBodies().Count() == 0 {
		t.Fatal("no body to fold")
	}
	b := def.SurfaceBodies().Item(0)
	maxZ := math.Inf(-1)
	for _, e := range b.Edges() {
		if z := e.StartVertex().Point().Z; z > maxZ {
			maxZ = z
		}
	}
	for _, e := range b.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if math.Abs(a.X-c.X) > 1e-6 && math.Abs(a.Y-c.Y) < 1e-6 && math.Abs(a.Z-maxZ) < 1e-6 && math.Abs(c.Z-maxZ) < 1e-6 {
			return string(e.ReferenceKey())
		}
	}
	t.Fatal("no top X-edge found")
	return ""
}

// expectMergedSolid decodes a feature-add result and asserts it produced one healthy solid.
func expectMergedSolid(t *testing.T, out json.RawMessage, what string) {
	t.Helper()
	var res struct {
		Bodies  int  `json:"bodies"`
		Healthy bool `json:"healthy"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("%s: decode: %v", what, err)
	}
	if res.Bodies != 1 || !res.Healthy {
		t.Errorf("%s: bodies=%d healthy=%v, want 1 healthy", what, res.Bodies, res.Healthy)
	}
}
