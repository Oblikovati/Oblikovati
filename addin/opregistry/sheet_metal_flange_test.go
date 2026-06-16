// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"math"
	"testing"

	"oblikovati.org/model/compdef"
)

// TestSheetMetalFlangeApply seeds a sheet-metal wall, flanges a top edge, and confirms one
// merged solid results; then checks the error paths (non-sheet-metal part, missing edge).
func TestSheetMetalFlangeApply(t *testing.T) {
	s := sheetMetalProfiledPart(t)
	if _, err := apply(t, s, "sheetMetalFace", `{"sketchIndex":0}`); err != nil {
		t.Fatalf("seed face: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	edge := topEdgeKey(t, def)

	out, err := applyMap(t, s, "sheetMetalFlange", map[string]any{"edge": edge, "height": "10 mm", "angle": "90 deg", "radius": "2 mm"})
	if err != nil {
		t.Fatalf("flange apply: %v", err)
	}
	var res struct {
		Bodies  int  `json:"bodies"`
		Healthy bool `json:"healthy"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if res.Bodies != 1 || !res.Healthy {
		t.Errorf("flange: bodies=%d healthy=%v, want 1 healthy", res.Bodies, res.Healthy)
	}

	// Error paths.
	if _, err := apply(t, sheetMetalProfiledPart(t), "sheetMetalFlange", `{"height":"5 mm"}`); err == nil {
		t.Error("flange without an edge must error")
	}
	if _, err := apply(t, profiledPart(t), "sheetMetalFlange", `{"edge":"x","height":"5 mm"}`); err == nil {
		t.Error("flange on a non-sheet-metal part must error")
	}
	if _, err := applyMap(t, s, "sheetMetalFlange", map[string]any{"edge": edge, "height": "bad"}); err == nil {
		t.Error("flange with a bad height must error")
	}
}

// topEdgeKey returns the reference key of a top-face edge of the part's first body — a
// deterministic edge to flange from.
func topEdgeKey(t *testing.T, def *compdef.PartComponentDefinition) string {
	t.Helper()
	if def.SurfaceBodies().Count() == 0 {
		t.Fatal("no body to flange")
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
