// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The split's TOOL over the wire (#1891). The tool is orthogonal to the split type, so the tests
// check both arrive independently — and that a surface split no longer needs a work plane, which
// is what made the tool unauthorable before.

// lastSplitDef returns the definition of the part's most recently added split.
func lastSplitDef(t *testing.T, s *app.Session) *feature.SplitSolidDefinition {
	t.Helper()
	fs := s.ActiveDocument().Content().(*compdef.PartComponentDefinition).Features()
	for i := fs.Count() - 1; i >= 0; i-- {
		if sp, ok := fs.Item(i).Definition().(*feature.SplitSolidFeature); ok {
			return sp.Definition()
		}
	}
	t.Fatal("no split feature on the part")
	return nil
}

// TestSplitToolReachesTheDefinition: each tool spelling maps to its kind and carries its index.
func TestSplitToolReachesTheDefinition(t *testing.T) {
	for _, c := range []struct {
		spelling string
		want     feature.SplitToolKind
	}{
		{"workSurface", feature.SplitByWorkSurface},
		{"surfaceBody", feature.SplitBySurfaceBody},
		{"path", feature.SplitByPath},
	} {
		t.Run(c.spelling, func(t *testing.T) {
			s, _, _ := extrudedSolid(t)
			if _, err := applyMap(t, s, "splitSolid", map[string]any{
				"tool": c.spelling, "toolIndex": 1,
			}); err != nil {
				t.Fatalf("splitSolid tool %q: %v", c.spelling, err)
			}
			def := lastSplitDef(t, s)
			if def.Tool != c.want || def.ToolIndex != 1 {
				t.Errorf("tool %q reached the definition as %v/%d, want %v/1", c.spelling, def.Tool, def.ToolIndex, c.want)
			}
		})
	}
}

// TestSurfaceSplitNeedsNoWorkPlane: a surface tool has no plane to resolve, so demanding one
// would reject the split on any part that has not been given a work plane it does not use.
func TestSurfaceSplitNeedsNoWorkPlane(t *testing.T) {
	s, _, _ := extrudedSolid(t)
	if _, err := applyMap(t, s, "splitSolid", map[string]any{
		"tool": "surfaceBody", "toolIndex": 0, "type": "splitBody",
	}); err != nil {
		t.Fatalf("surface split on a part with no work planes: %v", err)
	}
	if def := lastSplitDef(t, s); def.Plane != nil {
		t.Error("a surface split resolved a work plane it should not have")
	}
}

// TestSplitToolIsIndependentOfType: the type says what the split DOES and the tool says what it
// cuts with; setting one must not overwrite the other.
func TestSplitToolIsIndependentOfType(t *testing.T) {
	s, _, _ := extrudedSolid(t)
	if _, err := applyMap(t, s, "splitSolid", map[string]any{
		"tool": "surfaceBody", "toolIndex": 2, "type": "splitFaces",
	}); err != nil {
		t.Fatalf("splitSolid: %v", err)
	}
	def := lastSplitDef(t, s)
	if def.Tool != feature.SplitBySurfaceBody || def.ToolIndex != 2 || !def.FacesOnly {
		t.Errorf("split = %+v, want the surfaceBody tool at 2 in faces-only mode", def)
	}
}

// TestUnknownSplitToolIsRefused: a misspelled tool must not fall through to the work plane and
// cut along a datum nobody asked for.
func TestUnknownSplitToolIsRefused(t *testing.T) {
	s, _, _ := extrudedSolid(t)
	if _, err := applyMap(t, s, "splitSolid", map[string]any{"tool": "surface"}); err == nil {
		t.Error("an unknown split tool should error")
	}
}
