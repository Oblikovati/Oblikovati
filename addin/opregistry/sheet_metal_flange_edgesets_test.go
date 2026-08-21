// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"math"
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// topXEdgeKeys returns the reference keys of every X-aligned top-face edge of the part's first body
// (the two opposite sides), so a test can flange several edges in one feature.
func topXEdgeKeys(t *testing.T, def *compdef.PartComponentDefinition) []string {
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
	var keys []string
	for _, e := range b.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if math.Abs(a.X-c.X) > 1e-6 && math.Abs(a.Y-c.Y) < 1e-6 &&
			math.Abs(a.Z-maxZ) < 1e-6 && math.Abs(c.Z-maxZ) < 1e-6 {
			keys = append(keys, string(e.ReferenceKey()))
		}
	}
	return keys
}

// TestFlangeEdgeSetsReachTheDefinition drives a multi-edge flange over the wire (#2071): the
// edgeSets collection, each with its own edges and width, must reach the definition and build a
// merged solid.
func TestFlangeEdgeSetsReachTheDefinition(t *testing.T) {
	s, _ := seedSheetMetalSheet(t)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	keys := topXEdgeKeys(t, def)
	if len(keys) < 2 {
		t.Fatalf("seed plate exposed %d top X-edges, want ≥2", len(keys))
	}
	out, err := applyMap(t, s, "sheetMetalFlange", map[string]any{
		"height": "10 mm",
		"edgeSets": []any{
			map[string]any{"edges": []any{keys[0]}},
			map[string]any{"edges": []any{keys[1]}, "width": map[string]any{"type": "centered", "width": "20 mm"}},
		},
	})
	if err != nil {
		t.Fatalf("multi-edge flange apply: %v", err)
	}
	expectMergedSolid(t, out, "multi-edge flange")

	fdef := lastFlangeDef(t, s)
	if len(fdef.EdgeSets) != 2 {
		t.Fatalf("definition has %d edge sets, want 2", len(fdef.EdgeSets))
	}
	if string(fdef.EdgeSets[0].EdgeKeys[0]) != keys[0] {
		t.Errorf("edge set 0 key = %q, want %q", string(fdef.EdgeSets[0].EdgeKeys[0]), keys[0])
	}
	if fdef.EdgeSets[1].Width.Type != feature.WidthCentered {
		t.Errorf("edge set 1 width type = %v, want centered (its per-set width did not reach the def)", fdef.EdgeSets[1].Width.Type)
	}
}
