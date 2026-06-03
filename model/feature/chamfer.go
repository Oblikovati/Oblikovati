// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
)

// chamferEdges bevels each selected edge by cutting a triangular wedge tool that runs
// along it. All wedge tools are built from the original body up front (a boolean rebuilds
// topology with new lineage, so a reference key would not survive the first cut), then
// each is subtracted in turn. Convex edges only in phase A — a concave edge would add
// material, which is a follow-up.
func chamferEdges(in Input, keys [][]byte, dist float64, feat string) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	if dist <= 0 {
		return Output{}, fmt.Errorf("chamfer: distance %g must be > 0", dist)
	}
	tools, err := chamferTools(body, keys, dist, feat)
	if err != nil {
		return Output{}, err
	}
	result := body
	for _, tool := range tools {
		if result, err = ops.Boolean(ops.Cut, result, tool); err != nil {
			return Output{}, err
		}
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result)}, nil
}

// chamferTools resolves every edge key against the original body and builds its wedge,
// erroring if a key is lost (so the feature goes sick honestly).
func chamferTools(body *topo.Body, keys [][]byte, dist float64, feat string) ([]*topo.Body, error) {
	tools := make([]*topo.Body, 0, len(keys))
	for i, k := range keys {
		edge, ok := body.FindEdgeByKey(k)
		if !ok {
			return nil, fmt.Errorf("chamfer: edge reference lost")
		}
		tool, err := chamferWedge(edge, dist, fmt.Sprintf("%s/w%d", feat, i))
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

// chamferWedge builds the triangular prism removed to bevel an edge: a right-triangle
// cross-section with legs `dist` along each adjacent face's interior, swept along the
// edge (with a small overhang past each end so the boolean is clean).
func chamferWedge(edge *topo.Edge, dist float64, feat string) (*topo.Body, error) {
	faces := edge.Faces()
	if len(faces) != 2 {
		return nil, fmt.Errorf("chamfer: edge bounds %d faces, need 2", len(faces))
	}
	v0, v1 := edge.StartVertex().Point(), edge.EndVertex().Point()
	e, err := math.UnitVector3FromVector(v0.VectorTo(v1))
	if err != nil {
		return nil, fmt.Errorf("chamfer: degenerate edge")
	}
	mid := v0.Midpoint(v1)
	t1 := interiorDir(faces[0], mid, e)
	t2 := interiorDir(faces[1], mid, e)
	if t1.LengthSquared() == 0 || t2.LengthSquared() == 0 {
		return nil, fmt.Errorf("chamfer: cannot orient the wedge against the edge faces")
	}
	plane := planePerp(v0, e)
	u, v := plane.XAxis().AsVector(), plane.YAxis().AsVector()
	proj := func(w math.Vector3) math.Point2 { return math.P2(w.Dot(u), w.Dot(v)) }
	poly := []math.Point2{{X: 0, Y: 0}, proj(t1.Scale(dist)), proj(t2.Scale(dist))}
	const overhang = 1e-2
	return buildPrism(poly, plane, span{near: -overhang, far: v0.DistanceTo(v1) + overhang}, 0, feat), nil
}

// interiorDir returns the unit direction, perpendicular to the edge, pointing from the
// edge into the face's interior — the direction the chamfer sets back along that face.
func interiorDir(f *topo.Face, edgeMid math.Point3, e math.UnitVector3) math.Vector3 {
	toCentroid := edgeMid.VectorTo(centroidOf(faceVertexPoints(f)))
	perp := toCentroid.Sub(e.AsVector().Scale(toCentroid.Dot(e.AsVector())))
	u, err := math.UnitVector3FromVector(perp)
	if err != nil {
		return math.V3(0, 0, 0)
	}
	return u.AsVector()
}
