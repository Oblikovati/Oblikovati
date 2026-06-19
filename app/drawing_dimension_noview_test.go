// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// dimensionCommitter is the slice of the Tool interface the dimension tools share.
type dimensionCommitter interface {
	Start(*Session)
	Commit(*Session) error
	Params() ToolParams
}

// TestDimensionToolsRejectNoBaseView covers each dimension tool's "no base view to dimension"
// guard and its Params() base-view choice: started against a drawing with no base view, Commit
// must fail, and Params must still expose at least one choice row.
func TestDimensionToolsRejectNoBaseView(t *testing.T) {
	tools := map[string]dimensionCommitter{
		"Linear":    NewLinearDimensionTool(),
		"Radial":    NewRadialDimensionTool(),
		"Angular":   NewAngularDimensionTool(),
		"Set":       NewDimensionSetTool(),
		"ArcLength": NewArcLengthDimensionTool(),
		"Ordinate":  NewOrdinateDimensionTool(),
	}
	for name, tool := range tools {
		s := drawingWithModelSession(t) // a drawing with NO base view
		tool.Start(s)
		if err := tool.Commit(s); err == nil {
			t.Errorf("%s dimension tool: Commit with no base view should error", name)
		}
		if len(tool.Params().Choices) == 0 {
			t.Errorf("%s dimension tool: Params() exposed no choices", name)
		}
	}
}
