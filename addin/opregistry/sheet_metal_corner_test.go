// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"math"
	"testing"

	"oblikovati.org/model/compdef"
)

// verticalCornerEdgeKey returns the reference key of a through-thickness corner edge of the
// part's first body.
func verticalCornerEdgeKey(t *testing.T, def *compdef.PartComponentDefinition) string {
	t.Helper()
	b := def.SurfaceBodies().Item(0)
	for _, e := range b.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if math.Abs(a.X-c.X) < 1e-6 && math.Abs(a.Y-c.Y) < 1e-6 && math.Abs(a.Z-c.Z) > 1e-6 {
			return string(e.ReferenceKey())
		}
	}
	t.Fatal("no vertical corner edge found")
	return ""
}

// TestSheetMetalCornerApply seeds a sheet-metal wall and rounds a corner, confirming one
// healthy solid; then checks the error paths.
func TestSheetMetalCornerApply(t *testing.T) {
	s := sheetMetalProfiledPart(t)
	if _, err := apply(t, s, "sheetMetalFace", `{"sketchIndex":0}`); err != nil {
		t.Fatalf("seed face: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	edge := verticalCornerEdgeKey(t, def)

	out, err := applyMap(t, s, "sheetMetalCorner", map[string]any{"edges": []string{edge}, "treatment": "round", "size": "3 mm"})
	if err != nil {
		t.Fatalf("corner apply: %v", err)
	}
	expectMergedSolid(t, out, "corner")

	// A distance-and-angle chamfer plumbs its angle through to a healthy solid.
	if _, err := applyMap(t, s, "sheetMetalCorner", map[string]any{
		"edges": []string{edge}, "treatment": "chamfer", "size": "3 mm", "chamferType": "distanceAndAngle", "angle": "30 deg",
	}); err != nil {
		t.Fatalf("distance-and-angle chamfer apply: %v", err)
	}

	// Error paths.
	if _, err := apply(t, profiledPart(t), "sheetMetalCorner", `{"edges":["x"],"treatment":"round","size":"1 mm"}`); err == nil {
		t.Error("corner on a non-sheet-metal part must error")
	}
	if _, err := apply(t, s, "sheetMetalCorner", `{"edges":[],"treatment":"round","size":"1 mm"}`); err == nil {
		t.Error("corner with no edges must error")
	}
	if _, err := applyMap(t, s, "sheetMetalCorner", map[string]any{"edges": []string{edge}, "treatment": "seam", "size": "1 mm"}); err == nil {
		t.Error("corner with an unknown treatment must error")
	}
	if _, err := applyMap(t, s, "sheetMetalCorner", map[string]any{"edges": []string{edge}, "treatment": "round", "size": "bad"}); err == nil {
		t.Error("corner with a bad size must error")
	}
	if _, err := applyMap(t, s, "sheetMetalCorner", map[string]any{"edges": []string{edge}, "treatment": "chamfer", "size": "1 mm", "chamferType": "beveloid"}); err == nil {
		t.Error("corner with an unknown chamferType must error")
	}
}
