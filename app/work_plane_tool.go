// SPDX-License-Identifier: GPL-2.0-only

package app

// The work-plane flavours of the guided-pick datum tool (see DatumPickTool). Six were reachable
// from the ribbon; the model implements seventeen and api/wire exposed them all, so everything
// past Offset / Midplane / Three Points / Tangent / Normal-to-Axis was API-only (#2044).

func newMidplaneWorkPlaneTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Midplane", prompt: "Select two planes to bisect",
		filter: NewSelectionFilter(SelectWorkPlane),
		ready:  canMidplaneWorkPlane, create: discardResult((*Session).CreateMidplaneWorkPlane),
	}
}

func newThreePointWorkPlaneTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Three Points", prompt: "Select three points or model vertices",
		filter: NewSelectionFilter(SelectWorkPoint, SelectVertex),
		ready:  canThreePointWorkPlane, create: discardResult((*Session).CreateThreePointWorkPlane),
	}
}

func newTangentWorkPlaneTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Tangent to Face", prompt: "Select a plane, then a cylindrical/spherical face",
		filter: NewSelectionFilter(SelectWorkPlane, SelectFace),
		ready:  canTangentWorkPlane, create: discardResult((*Session).CreateTangentWorkPlane),
	}
}

func newNormalToAxisWorkPlaneTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Normal to Axis", prompt: "Select an axis, then a point on it",
		filter: NewSelectionFilter(SelectWorkAxis, SelectWorkPoint, SelectVertex),
		ready:  canNormalToAxisWorkPlane, create: discardResult((*Session).CreateNormalToAxisWorkPlane),
	}
}

func newParallelThroughPointWorkPlaneTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Parallel to Plane through Point", prompt: "Select a plane, then a point or vertex",
		filter: NewSelectionFilter(SelectWorkPlane, SelectWorkPoint, SelectVertex),
		ready:  canParallelThroughPointWorkPlane, create: discardResult((*Session).CreateParallelThroughPointWorkPlane),
	}
}

func newLineAndPointWorkPlaneTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Through Axis and Point", prompt: "Select an axis, then a point or vertex not on it",
		filter: NewSelectionFilter(SelectWorkAxis, SelectWorkPoint, SelectVertex),
		ready:  canLineAndPointWorkPlane, create: discardResult((*Session).CreateLineAndPointWorkPlane),
	}
}

func newTwoLinesWorkPlaneTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Through Two Axes", prompt: "Select two axes lying in one plane",
		filter: NewSelectionFilter(SelectWorkAxis),
		ready:  canTwoLinesWorkPlane, create: discardResult((*Session).CreateTwoLinesWorkPlane),
	}
}

func newPointAndTangentWorkPlaneTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Tangent to Face through Point", prompt: "Select a cylindrical/spherical face, then a point",
		filter: NewSelectionFilter(SelectFace, SelectWorkPoint, SelectVertex),
		ready:  canPointAndTangentWorkPlane, create: discardResult((*Session).CreatePointAndTangentWorkPlane),
	}
}

func newLineAndTangentWorkPlaneTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Tangent to Face through Axis", prompt: "Select a cylindrical/spherical face, then an axis",
		filter: NewSelectionFilter(SelectFace, SelectWorkAxis),
		ready:  canLineAndTangentWorkPlane, create: discardResult((*Session).CreateLineAndTangentWorkPlane),
	}
}

func newTorusMidPlaneWorkPlaneTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Torus Midplane", prompt: "Select a toroidal face",
		filter: NewSelectionFilter(SelectFace),
		ready:  canTorusMidPlaneWorkPlane, create: discardResult((*Session).CreateTorusMidPlaneWorkPlane),
	}
}
