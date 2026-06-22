// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// migratedSelectingTools constructs one of every tool that declares its selection through the
// engine (ADR-0041). Constructors that take a build closure get nil — it is only invoked on
// commit, which this test does not reach.
func migratedSelectingTools() []Tool {
	return []Tool{
		NewDraftTool(), NewDeleteFaceTool(), NewShellTool(), NewDecalTool(),
		NewFaceFilletTool(), NewFullRoundFilletTool(), NewFaceOffsetTool(), NewReplaceFaceTool(),
		NewGripSnapTool(), NewHoleTool(), NewThreadTool(), NewPatchTool(),
		NewChamferTool(), NewFilletTool(), NewLipTool(), NewExtendTool(),
		NewCoilTool(), NewEmbossTool(), NewGrillTool(), NewRestTool(), NewRuledSurfaceTool(),
		NewSheetMetalFaceTool(), NewCreateSketchTool(), NewExtrudeTool(), NewMeasureTool(),
		NewSweepTool(), NewProjectGeometryTool(), NewSplitTool(), NewSurfaceTrimTool(),
		NewOffsetWorkPlaneTool(), NewRevolveTool(), NewLoftTool(), NewBossTool(),
		NewDirectEditTool(), NewMoveFaceTool(), NewCombineTool(), NewMoveBodyTool(),
		NewSheetMetalFlangeTool(), NewSheetMetalLipTool(), NewSheetMetalRipTool(),
		NewSheetMetalPunchTool(), NewSheetMetalCosmeticBendTool(), NewSheetMetalBendTool(),
		NewSheetMetalFoldTool(), NewSheetMetalCornerTool(), NewSheetMetalCornerSeamTool(),
		NewSheetMetalCutTool(), NewSheetMetalHemTool(), NewSheetMetalContourFlangeTool(),
		NewSheetMetalLoftedFlangeTool(), NewSheetMetalContourRollTool(),
		NewIncludeGeometry3DTool(), NewSurfaceCurve3DTool(), NewThickenTool(),
		NewAssemblyConstraintTool("Mate", 2, nil), NewAssemblyJointTool("Rigid", nil),
		NewAssemblyChamferTool(), NewAssemblyFilletTool(),
	}
}

// TestEverySelectingToolDeclaresAndReportsPicks drives the engine over every migrated tool: starting
// it installs the filter from AcceptedKinds (restricted when the tool declares kinds), reporting
// Picks is empty before any pick, and cancelling hands the filter back to the ambient state. This is
// the uniform contract the host relies on (ADR-0041) — and the regression guard that a tool's
// declared kinds actually take effect, the bug class the engine removes.
func TestEverySelectingToolDeclaresAndReportsPicks(t *testing.T) {
	for _, tool := range migratedSelectingTools() {
		s := NewSession()
		s.StartTool(tool)

		if sel, ok := tool.(Selecting); ok && len(sel.AcceptedKinds()) > 0 {
			if !s.Selection().Filter().IsRestricted() {
				t.Errorf("%s declares kinds %v but the engine left the filter unrestricted",
					tool.Name(), sel.AcceptedKinds())
			}
		}
		if picks := s.ToolPicks(); len(picks) != 0 {
			t.Errorf("%s reports %d picks before any pick, want 0", tool.Name(), len(picks))
		}

		s.CancelTool()
		if s.Selection().Filter().IsRestricted() {
			t.Errorf("%s left the filter restricted after cancel", tool.Name())
		}
	}
}
