// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A NURBS surface primitive (M36-F03) is a base freeform surface to start styling from — the
// Class-A workflow needs a clean NURBS quilt to push control points on, and until now the only
// freeform start was the sub-D cage. This feature creates a flat, evenly-knotted degree-3 plane
// patch of a given size and control-point count as a one-face surface body; the control net is
// then shaped by [ControlPointEditFeature]. It serializes by its size/resolution recipe.

// NurbsPlaneDefinition is the recipe for a flat NURBS plane patch: its width/height (X/Y span) and
// the control-point count in each direction (>= 4 for the cubic).
type NurbsPlaneDefinition struct {
	Width, Height  float64
	UCount, VCount int
}

// NurbsPlaneFeature creates a flat degree-3 NURBS plane patch as a surface body.
type NurbsPlaneFeature struct {
	def      *NurbsPlaneDefinition
	featName string
}

// Definition returns the plane-patch recipe.
func (n *NurbsPlaneFeature) Definition() *NurbsPlaneDefinition { return n.def }

// Kind implements [Feature].
func (n *NurbsPlaneFeature) Kind() string { return "nurbs-plane" }

// Recompute builds the flat NURBS plane patch and appends it as a surface body. It errors on an
// invalid recipe (non-positive size or a control count below the cubic minimum of 4).
func (n *NurbsPlaneFeature) Recompute(in Input) (Output, error) {
	surf, err := flatNurbsPlane(n.def)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: appendBody(in.Bodies, nurbsSurfaceBody(surf, n.featName))}, nil
}

// flatNurbsPlane builds the z=0 degree-3 B-spline plane of the given size and control-point count,
// with evenly spaced control points and a clamped uniform knot vector.
func flatNurbsPlane(def *NurbsPlaneDefinition) (geom.BSplineSurface, error) {
	if def.Width <= 0 || def.Height <= 0 {
		return geom.BSplineSurface{}, fmt.Errorf("nurbs plane: size %gx%g must be positive", def.Width, def.Height)
	}
	if def.UCount < 4 || def.VCount < 4 {
		return geom.BSplineSurface{}, fmt.Errorf("nurbs plane: control counts %dx%d must be >= 4 (cubic)", def.UCount, def.VCount)
	}
	ctrl := make([][]math.Point3, def.UCount)
	weights := make([][]float64, def.UCount)
	for i := 0; i < def.UCount; i++ {
		ctrl[i] = make([]math.Point3, def.VCount)
		weights[i] = make([]float64, def.VCount)
		for j := 0; j < def.VCount; j++ {
			x := def.Width * float64(i) / float64(def.UCount-1)
			y := def.Height * float64(j) / float64(def.VCount-1)
			ctrl[i][j] = math.P3(math.Scalar(x), math.Scalar(y), 0)
			weights[i][j] = 1
		}
	}
	return geom.NewBSplineSurface(3, 3, ctrl, weights, clampedCubicKnots(def.UCount), clampedCubicKnots(def.VCount))
}

// clampedCubicKnots returns a clamped degree-3 knot vector with evenly spaced interior knots for
// nctrl control points.
func clampedCubicKnots(nctrl int) []float64 {
	const p = 3
	knots := make([]float64, nctrl+p+1)
	interior := nctrl - p - 1
	for j := 1; j <= interior; j++ {
		knots[p+j] = float64(j) / float64(interior+1)
	}
	for i := nctrl; i < nctrl+p+1; i++ {
		knots[i] = 1
	}
	return knots
}

// nurbsSurfaceBody wraps a B-spline surface in a one-face surface body with straight boundary edges
// between the patch corners (the flat plane's boundary iso-curves are straight lines).
func nurbsSurfaceBody(surf geom.BSplineSurface, feat string) *topo.Body {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok(feat, "body", 0)))
	corners := [4]math.Point3{surf.PointAt(0, 0), surf.PointAt(1, 0), surf.PointAt(1, 1), surf.PointAt(0, 1)}
	v := make([]*topo.Vertex, 4)
	for i, p := range corners {
		v[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok(feat, "vertex", i)))
	}
	uses := make([]topo.Use, 4)
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		e := bld.AddEdge(geom.NewLineSegment(corners[i], corners[j]), v[i], v[j], topo.NewLineage(topo.Tok(feat, "edge", i)))
		uses[i] = topo.Fwd(e)
	}
	bld.AddFace(surf, topo.NewLineage(topo.Tok(feat, "face", 0)), topo.OuterLoop(uses...))
	return bld.Build()
}

// NurbsPlaneFeatures adds NURBS plane primitives into the engine.
type NurbsPlaneFeatures struct{ engine *PartFeatures }

// NewNurbsPlaneFeatures binds the collection to a feature engine.
func NewNurbsPlaneFeatures(engine *PartFeatures) *NurbsPlaneFeatures {
	return &NurbsPlaneFeatures{engine: engine}
}

// Add creates a flat degree-3 NURBS plane patch (width×height, uCount×vCount control points).
func (c *NurbsPlaneFeatures) Add(width, height float64, uCount, vCount int) *PartFeature {
	def := &NurbsPlaneDefinition{Width: width, Height: height, UCount: uCount, VCount: vCount}
	nf := &NurbsPlaneFeature{def: def, featName: "NurbsPlane"}
	pf := c.engine.Add(nf)
	nf.featName = pf.name
	return pf
}
