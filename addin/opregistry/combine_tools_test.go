// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The combine's authoring surface (#1894): the tool COLLECTION and keep-tool-bodies. Both change
// what the feature is rather than what one boolean computes, so the tests read the definition back
// instead of only checking that the request was accepted.

// lastCombineDef returns the definition of the part's most recently added combine.
func lastCombineDef(t *testing.T, s *app.Session) *feature.CombineDefinition {
	t.Helper()
	fs := s.ActiveDocument().Content().(*compdef.PartComponentDefinition).Features()
	for i := fs.Count() - 1; i >= 0; i-- {
		if c, ok := fs.Item(i).Definition().(*feature.CombineFeature); ok {
			return c.Definition()
		}
	}
	t.Fatal("no combine feature on the part")
	return nil
}

// TestCombineToolIndicesReachTheDefinition: the list must arrive whole. A combine that silently
// kept only the first tool would still build a valid solid, just not the authored one.
func TestCombineToolIndicesReachTheDefinition(t *testing.T) {
	s := twoBodyPart(t)
	if _, err := applyMap(t, s, "combine", map[string]any{
		"targetIndex": 0, "toolIndices": []int{1}, "operation": "cut", "keepToolBodies": true,
	}); err != nil {
		t.Fatalf("combine with toolIndices: %v", err)
	}
	def := lastCombineDef(t, s)
	if len(def.ToolIndices) != 1 || def.ToolIndices[0] != 1 {
		t.Errorf("toolIndices reached the definition as %v, want [1]", def.ToolIndices)
	}
	if !def.KeepTools {
		t.Error("keepToolBodies did not reach the definition")
	}
}

// TestCombineToolIndexStillWorks: the shipped single-index spelling keeps working on its own, and
// consumes its tool as it always did.
func TestCombineToolIndexStillWorks(t *testing.T) {
	s := twoBodyPart(t)
	if _, err := applyMap(t, s, "combine", map[string]any{
		"targetIndex": 0, "toolIndex": 1, "operation": "join",
	}); err != nil {
		t.Fatalf("combine with toolIndex: %v", err)
	}
	def := lastCombineDef(t, s)
	if len(def.ToolIndices) != 1 || def.ToolIndices[0] != 1 || def.KeepTools {
		t.Errorf("single-index combine = %+v, want tools [1] consuming", def)
	}
}
