// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestDeriveToolsDraftFeature: the Derive and Shrinkwrap tools draft the feature they would
// commit for the commit gate (#1626) once Start has captured a source assembly, and refuse
// before one is chosen.
func TestDeriveToolsDraftFeature(t *testing.T) {
	s, _ := partAndSourceAssembly(t)
	for _, tool := range []PartFeatureTool{NewDeriveAssemblyTool(), NewShrinkwrapTool()} {
		if _, ok := tool.DraftFeature(s); ok {
			t.Errorf("%s: should not draft before Start captures a source", tool.Name())
		}
		tool.Start(s) // captures the open assembly as the selected source
		if draft, ok := tool.DraftFeature(s); !ok || draft == nil {
			t.Errorf("%s: tool with a source should draft its feature", tool.Name())
		}
	}
}
