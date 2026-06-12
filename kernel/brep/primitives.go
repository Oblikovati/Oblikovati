// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Analytic solid primitives for the transient B-rep factory (M07-F05,
// Oblikovati/Oblikovati#628): block, cylinder/cone, sphere, torus. Sphere and
// torus are CLOSED surfaces — one boundary-less face each; the tessellator's
// closed-domain mesher wraps their seams and poles watertight (M25 PBI-330).

// SolidBlock builds the axis-aligned box [min, max] as six planar faces.
//
// Example: b, err := brep.SolidBlock(math.P3(0,0,0), math.P3(4,2,1), "block")
func SolidBlock(min, max math.Point3, feat string) (*topo.Body, error) {
	sx, sy, sz := float64(max.X-min.X), float64(max.Y-min.Y), float64(max.Z-min.Z)
	if sx <= 0 || sy <= 0 || sz <= 0 {
		return nil, fmt.Errorf("brep.SolidBlock: box %v→%v has a non-positive extent (%g, %g, %g)",
			min, max, sx, sy, sz)
	}
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok(feat, "body", 0)))
	v := blockVertices(bld, min, max, feat)
	e := blockEdges(bld, v, feat)
	blockFaces(bld, v, e, feat)
	return bld.Build(), nil
}

// blockCorner index convention: bit 0 = +X, bit 1 = +Y, bit 2 = +Z.
func blockVertices(bld *topo.Builder, min, max math.Point3, feat string) [8]*topo.Vertex {
	var v [8]*topo.Vertex
	for i := 0; i < 8; i++ {
		p := math.P3(pick(i&1 != 0, max.X, min.X), pick(i&2 != 0, max.Y, min.Y), pick(i&4 != 0, max.Z, min.Z))
		v[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok(feat, "vertex", i)))
	}
	return v
}

func pick(hi bool, a, b math.Scalar) math.Scalar {
	if hi {
		return a
	}
	return b
}

// blockEdgePairs lists the cube's 12 edges by corner index.
var blockEdgePairs = [12][2]int{
	{0, 1}, {2, 3}, {4, 5}, {6, 7}, // along X
	{0, 2}, {1, 3}, {4, 6}, {5, 7}, // along Y
	{0, 4}, {1, 5}, {2, 6}, {3, 7}, // along Z
}

func blockEdges(bld *topo.Builder, v [8]*topo.Vertex, feat string) map[[2]int]*topo.Edge {
	out := map[[2]int]*topo.Edge{}
	for i, pair := range blockEdgePairs {
		out[pair] = bld.AddEdge(geom.NewLineSegment(v[pair[0]].Point(), v[pair[1]].Point()),
			v[pair[0]], v[pair[1]], topo.NewLineage(topo.Tok(feat, "edge", i)))
	}
	return out
}

// blockFaceCorners lists each face's corners wound CCW seen from OUTSIDE.
var blockFaceCorners = [6][4]int{
	{0, 2, 3, 1}, // -Z (bottom): outward −Z
	{4, 5, 7, 6}, // +Z (top)
	{0, 1, 5, 4}, // -Y (front)
	{2, 6, 7, 3}, // +Y (back)
	{0, 4, 6, 2}, // -X (left)
	{1, 3, 7, 5}, // +X (right)
}

func blockFaces(bld *topo.Builder, v [8]*topo.Vertex, e map[[2]int]*topo.Edge, feat string) {
	for fi, corners := range blockFaceCorners {
		uses := make([]topo.Use, 4)
		for k := 0; k < 4; k++ {
			a, b := corners[k], corners[(k+1)%4]
			if edge, ok := e[[2]int{a, b}]; ok {
				uses[k] = topo.Fwd(edge)
			} else {
				uses[k] = topo.Rev(e[[2]int{b, a}])
			}
		}
		n := v[corners[0]].Point().VectorTo(v[corners[1]].Point()).
			Cross(v[corners[1]].Point().VectorTo(v[corners[2]].Point()))
		surf, _ := geom.NewPlane(v[corners[0]].Point(), n)
		bld.AddFace(surf, topo.NewLineage(topo.Tok(feat, "face", fi)), topo.OuterLoop(uses...))
	}
}

// SolidCylinderCone builds the solid of revolution between two circular
// sections: equal radii give a cylinder, differing radii a cone frustum, a
// zero radius the full cone. Elliptical sections are not supported — the
// kernel's surfaces of revolution are circular.
//
// Example: b, err := brep.SolidCylinderCone(math.P3(0,0,0), math.P3(0,0,5), 2, 2, "cyl")
func SolidCylinderCone(bottom, top math.Point3, bottomRadius, topRadius float64, feat string) (*topo.Body, error) {
	axis := bottom.VectorTo(top)
	h := float64(axis.Length())
	if h == 0 {
		return nil, fmt.Errorf("brep.SolidCylinderCone: bottom and top coincide at %v", bottom)
	}
	if bottomRadius <= 0 && topRadius <= 0 {
		return nil, fmt.Errorf("brep.SolidCylinderCone: radii (%g, %g) need at least one positive",
			bottomRadius, topRadius)
	}
	if bottomRadius == topRadius {
		return SolidCylinder(bottom, axis.Scale(math.Scalar(1/h)), bottomRadius, h)
	}
	return SolidOfRevolution(bottom, axis, coneMeridian(bottomRadius, topRadius, h), feat)
}

// coneMeridian is the CCW (radius, height) polygon of a cone/frustum.
func coneMeridian(r0, r1, h float64) []math.Point2 {
	if r1 <= 0 {
		return []math.Point2{math.P2(0, 0), math.P2(math.Scalar(r0), 0), math.P2(0, math.Scalar(h))}
	}
	if r0 <= 0 {
		return []math.Point2{math.P2(0, 0), math.P2(math.Scalar(r1), math.Scalar(h)), math.P2(0, math.Scalar(h))}
	}
	return []math.Point2{
		math.P2(0, 0), math.P2(math.Scalar(r0), 0),
		math.P2(math.Scalar(r1), math.Scalar(h)), math.P2(0, math.Scalar(h)),
	}
}

// SolidSphere builds the full sphere as one boundary-less analytic face.
//
// Example: b, err := brep.SolidSphere(math.P3(0,0,0), 3, "sphere")
func SolidSphere(center math.Point3, radius float64, feat string) (*topo.Body, error) {
	surf, err := geom.NewSphere(center, radius)
	if err != nil {
		return nil, err
	}
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok(feat, "body", 0)))
	bld.AddFace(surf, topo.NewLineage(topo.Tok(feat, "face", 0)))
	return bld.Build(), nil
}

// SolidTorus builds the full torus (axis along axisDir) as one boundary-less
// analytic face.
//
// Example: b, err := brep.SolidTorus(math.P3(0,0,0), math.V3(0,0,1), 5, 1, "torus")
func SolidTorus(center math.Point3, axisDir math.Vector3, majorRadius, minorRadius float64, feat string) (*topo.Body, error) {
	if minorRadius >= majorRadius {
		return nil, fmt.Errorf("brep.SolidTorus: minor radius %g must be below major %g (self-intersecting otherwise)",
			minorRadius, majorRadius)
	}
	surf, err := geom.NewTorus(center, axisDir, majorRadius, minorRadius)
	if err != nil {
		return nil, err
	}
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok(feat, "body", 0)))
	bld.AddFace(surf, topo.NewLineage(topo.Tok(feat, "face", 0)))
	return bld.Build(), nil
}
