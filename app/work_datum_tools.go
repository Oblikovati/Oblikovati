// SPDX-License-Identifier: GPL-2.0-only

package app

// The work-axis and work-point flavours of the guided-pick datum tool (see DatumPickTool).
// There was no Work Axis command at all, and Work Point was reachable only through the
// point-cloud snap, so 0 of 9 axis and 1 of 10 point constructors had a UI path (#2043).

func newEdgeWorkAxisTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Axis on Edge", prompt: "Select a linear edge",
		filter: NewSelectionFilter(SelectEdge),
		ready:  canEdgeWorkAxis, create: discardResult((*Session).CreateEdgeWorkAxis),
	}
}

func newTwoPointWorkAxisTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Axis through Two Points", prompt: "Select two points or model vertices",
		filter: NewSelectionFilter(SelectWorkPoint, SelectVertex),
		ready:  canTwoPointWorkAxis, create: discardResult((*Session).CreateTwoPointWorkAxis),
	}
}

func newRevolvedFaceWorkAxisTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Axis of Revolved Face", prompt: "Select a cylindrical, conical, spherical or toroidal face",
		filter: NewSelectionFilter(SelectFace),
		ready:  canRevolvedFaceWorkAxis, create: discardResult((*Session).CreateRevolvedFaceWorkAxis),
	}
}

func newPlaneIntersectionWorkAxisTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Axis at Two Planes", prompt: "Select two planes that intersect",
		filter: NewSelectionFilter(SelectWorkPlane),
		ready:  canPlaneIntersectionWorkAxis, create: discardResult((*Session).CreatePlaneIntersectionWorkAxis),
	}
}

func newNormalToPlaneWorkAxisTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Axis Normal to Plane", prompt: "Select a plane, then a point on it",
		filter: NewSelectionFilter(SelectWorkPlane, SelectWorkPoint, SelectVertex),
		ready:  canNormalToPlaneWorkAxis, create: discardResult((*Session).CreateNormalToPlaneWorkAxis),
	}
}

func newParallelToAxisWorkAxisTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Axis Parallel to Axis", prompt: "Select an axis, then a point to pass through",
		filter: NewSelectionFilter(SelectWorkAxis, SelectWorkPoint, SelectVertex),
		ready:  canParallelToAxisWorkAxis, create: discardResult((*Session).CreateParallelToAxisWorkAxis),
	}
}

func newAxisOnPlaneWorkAxisTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Axis Projected to Plane", prompt: "Select an axis, then the plane to project it onto",
		filter: NewSelectionFilter(SelectWorkAxis, SelectWorkPlane),
		ready:  canAxisOnPlaneWorkAxis, create: discardResult((*Session).CreateAxisOnPlaneWorkAxis),
	}
}

func newVertexWorkPointTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Point at Vertex", prompt: "Select a vertex or datum point",
		filter: NewSelectionFilter(SelectVertex, SelectWorkPoint),
		ready:  canVertexWorkPoint, create: discardResult((*Session).CreateVertexWorkPoint),
	}
}

func newMidpointWorkPointTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Point at Edge Midpoint", prompt: "Select an edge",
		filter: NewSelectionFilter(SelectEdge),
		ready:  canMidpointWorkPoint, create: discardResult((*Session).CreateMidpointWorkPoint),
	}
}

func newCentroidWorkPointTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Point at Centroid", prompt: "Select the edges to take the centroid of, then click OK",
		filter: NewSelectionFilter(SelectEdge),
		ready:  canCentroidWorkPoint, create: discardResult((*Session).CreateCentroidWorkPoint),
	}
}

func newFaceCenterWorkPointTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Point at Face Centre", prompt: "Select a spherical or toroidal face",
		filter: NewSelectionFilter(SelectFace),
		ready:  canFaceCenterWorkPoint, create: discardResult((*Session).CreateFaceCenterWorkPoint),
	}
}

func newThreePlaneWorkPointTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Point at Three Planes", prompt: "Select three planes that meet at a point",
		filter: NewSelectionFilter(SelectWorkPlane),
		ready:  canThreePlaneWorkPoint, create: discardResult((*Session).CreateThreePlaneWorkPoint),
	}
}

func newTwoAxisWorkPointTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Point at Two Axes", prompt: "Select two axes that intersect",
		filter: NewSelectionFilter(SelectWorkAxis),
		ready:  canTwoAxisWorkPoint, create: discardResult((*Session).CreateTwoAxisWorkPoint),
	}
}

func newPlaneAndAxisWorkPointTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Point at Plane and Axis", prompt: "Select a plane, then the axis piercing it",
		filter: NewSelectionFilter(SelectWorkPlane, SelectWorkAxis),
		ready:  canPlaneAndAxisWorkPoint, create: discardResult((*Session).CreatePlaneAndAxisWorkPoint),
	}
}

func newCurveAndEntityWorkPointTool() *DatumPickTool {
	return &DatumPickTool{
		name: "Point at Curve and Surface", prompt: "Select an edge, then the plane or face it crosses",
		filter: NewSelectionFilter(SelectEdge, SelectWorkPlane, SelectFace),
		ready:  canCurveAndEntityWorkPoint, create: discardResult((*Session).CreateCurveAndEntityWorkPoint),
	}
}
