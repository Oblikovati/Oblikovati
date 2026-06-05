// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// The head highlights every tool's picks uniformly (head/ui/tool_highlight.go: toolPicks) by
// looking for one of these pick-accessor interfaces. A tool that picks B-rep geometry in the
// viewport (a face, edge or sketch region) but exposes none of them would get no highlight — an
// inconsistency. This test pins the contract: keep it green by giving any new view-picking tool
// the matching accessor (Edges/Faces/PickedProfiles/PickedProfile/PickedFace).

type edgesAccessor interface{ Edges() []EdgeHandle }
type facesAccessor interface{ Faces() []FaceHandle }
type profilesAccessor interface{ PickedProfiles() []ProfileHandle }
type profileAccessor interface{ PickedProfile() (ProfileHandle, bool) }
type faceAccessor interface{ PickedFace() (FaceHandle, bool) }

// exposesPickAccessor mirrors the head's toolPicks: does the tool expose any pick accessor the
// unified highlight can read?
func exposesPickAccessor(t Tool) bool {
	switch t.(type) {
	case edgesAccessor, facesAccessor, profilesAccessor, profileAccessor, faceAccessor:
		return true
	default:
		return false
	}
}

// viewGeometryPickers are the tools that pick a B-rep face/edge or a sketch region in the
// viewport (filters SelectFace/SelectEdge/SelectProfile). Body- and plane-picking tools are
// excluded: bodies are highlighted by the per-body recolor and planes by the plane overlay.
func viewGeometryPickers() []Tool {
	return []Tool{
		NewExtrudeTool(), NewRevolveTool(), NewSweepTool(), NewCoilTool(), NewLoftTool(),
		NewEmbossTool(), NewPatchTool(), NewDecalTool(),
		NewChamferTool(), NewFilletTool(), NewExtendTool(), NewHoleTool(),
		NewShellTool(), NewDraftTool(), NewFaceOffsetTool(), NewReplaceFaceTool(),
		NewDeleteFaceTool(), NewMoveFaceTool(),
	}
}

func TestViewPickingToolsExposeHighlightAccessor(t *testing.T) {
	for _, tool := range viewGeometryPickers() {
		if !exposesPickAccessor(tool) {
			t.Errorf("tool %q picks viewport geometry but exposes no highlight accessor "+
				"(want one of Edges/Faces/PickedProfiles/PickedProfile/PickedFace)", tool.Name())
		}
	}
}
