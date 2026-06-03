// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
)

// Shell hollows a planar-faceted solid to wall thickness t, leaving the removed faces as
// openings. It builds the inner cavity solid by offsetting every KEPT face inward by t
// (each face's plane shifted along −normal; a vertex moves to where its faces' offset
// planes meet) while leaving the REMOVED faces in place — so the cavity stays flush with
// them and the difference opens them (the coplanar B-rep rule). The result is solid − cavity.
// Inward shell only; outward/both-sides are a follow-up.
func Shell(solid *topo.Body, removedKeys [][]byte, t float64) (*topo.Body, error) {
	if t <= 0 {
		return nil, fmt.Errorf("shell: thickness %g must be > 0", t)
	}
	removed, err := resolveFaceSet(solid, removedKeys)
	if err != nil {
		return nil, err
	}
	cavity := offsetCavity(solid, removed, t)
	return Boolean(Cut, solid, cavity)
}

// resolveFaceSet turns face reference keys into the set of face IDs to leave open, erroring
// if a key no longer resolves (the feature must go Sick honestly).
func resolveFaceSet(solid *topo.Body, keys [][]byte) (map[uint64]bool, error) {
	set := make(map[uint64]bool, len(keys))
	for _, k := range keys {
		f, ok := solid.FindFaceByKey(k)
		if !ok {
			return nil, fmt.Errorf("shell: face reference lost")
		}
		set[f.ID()] = true
	}
	return set, nil
}

// offsetCavity clones the solid's topology with every vertex moved inward to the meeting
// point of its faces' offset planes (removed faces contribute their original plane), giving
// the inner solid whose subtraction hollows the shell.
func offsetCavity(solid *topo.Body, removed map[uint64]bool, t float64) *topo.Body {
	vf := vertexFaceMap(solid)
	lin := topo.NewLineage(topo.Tok("shell", "cavity", 0))
	bld := topo.NewBuilder(true, lin)
	nv := make(map[uint64]*topo.Vertex, len(solid.Vertices()))
	for i, v := range solid.Vertices() {
		nv[v.ID()] = bld.AddVertex(innerVertex(v, vf[v.ID()], removed, t), topo.NewLineage(topo.Tok("shell", "v", i)))
	}
	ne := make(map[uint64]*topo.Edge, len(solid.Edges()))
	for i, e := range solid.Edges() {
		a, b := nv[e.StartVertex().ID()], nv[e.EndVertex().ID()]
		ne[e.ID()] = bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, topo.NewLineage(topo.Tok("shell", "e", i)))
	}
	for i, f := range solid.Faces() {
		bld.AddFace(offsetPlane(f, removed, t), topo.NewLineage(topo.Tok("shell", "f", i)), cloneLoops(f, ne)...)
	}
	return bld.Build()
}

// cloneLoops rebuilds a face's loop specs against the inner-solid edges, preserving each
// edge use's direction and the outer/inner role.
func cloneLoops(f *topo.Face, ne map[uint64]*topo.Edge) []topo.LoopSpec {
	specs := make([]topo.LoopSpec, 0, len(f.Loops()))
	for _, l := range f.Loops() {
		uses := make([]topo.Use, 0, len(l.EdgeUses()))
		for _, u := range l.EdgeUses() {
			uses = append(uses, topo.Use{Edge: ne[u.Edge().ID()], Reversed: u.Reversed()})
		}
		if l.IsOuter() {
			specs = append(specs, topo.OuterLoop(uses...))
		} else {
			specs = append(specs, topo.InnerLoop(uses...))
		}
	}
	return specs
}

// offsetPlane returns a face's plane shifted inward by t (origin moved along −normal),
// or the unchanged plane for a removed face.
func offsetPlane(f *topo.Face, removed map[uint64]bool, t float64) geom.Plane {
	pl := f.Geometry().(geom.Plane)
	if removed[f.ID()] {
		return pl
	}
	n := pl.Normal()
	moved, _ := geom.NewPlaneFromAxes(pl.Origin.TranslateBy(n.Scale(-t)), pl.UAxis.AsVector(), pl.VAxis.AsVector())
	return moved
}

// vertexFaceMap returns, per vertex ID, the faces meeting at that vertex.
func vertexFaceMap(solid *topo.Body) map[uint64][]*topo.Face {
	m := map[uint64][]*topo.Face{}
	seen := map[[2]uint64]bool{}
	for _, f := range solid.Faces() {
		for _, e := range f.Edges() {
			for _, v := range e.Vertices() {
				if key := [2]uint64{v.ID(), f.ID()}; !seen[key] {
					seen[key] = true
					m[v.ID()] = append(m[v.ID()], f)
				}
			}
		}
	}
	return m
}

// innerVertex returns where a vertex moves under the shell: the least-squares meeting point
// of its (offset) face planes — exact for a 3-face corner, best-fit for more. Falls back to
// the original position if the planes are degenerate (parallel normals).
func innerVertex(v *topo.Vertex, faces []*topo.Face, removed map[uint64]bool, t float64) math.Point3 {
	var a [3][3]float64
	var b [3]float64
	for _, f := range faces {
		pl := f.Geometry().(geom.Plane)
		n := pl.Normal()
		d := n.Dot(pl.Origin.AsVector())
		if !removed[f.ID()] {
			d -= t
		}
		nv := [3]float64{n.X, n.Y, n.Z}
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				a[i][j] += nv[i] * nv[j]
			}
			b[i] += nv[i] * d
		}
	}
	x, ok := solve3(a, b)
	if !ok {
		return v.Point()
	}
	return math.P3(x[0], x[1], x[2])
}

// solve3 solves the 3×3 system a·x = b by Cramer's rule, ok=false when a is singular.
func solve3(a [3][3]float64, b [3]float64) ([3]float64, bool) {
	det := det3(a)
	if det < 1e-12 && det > -1e-12 {
		return [3]float64{}, false
	}
	var x [3]float64
	for c := 0; c < 3; c++ {
		m := a
		for r := 0; r < 3; r++ {
			m[r][c] = b[r]
		}
		x[c] = det3(m) / det
	}
	return x, true
}

// det3 returns the determinant of a 3×3 matrix.
func det3(m [3][3]float64) float64 {
	return m[0][0]*(m[1][1]*m[2][2]-m[1][2]*m[2][1]) -
		m[0][1]*(m[1][0]*m[2][2]-m[1][2]*m[2][0]) +
		m[0][2]*(m[1][0]*m[2][1]-m[1][1]*m[2][0])
}
