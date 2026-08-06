// SPDX-License-Identifier: GPL-2.0-only

// Package degenerate builds bodies that are deliberately BAD INPUT for the tessellator, so tests
// that need a mesh degradation can produce one on purpose instead of hunting for a model where the
// kernel happens to misbehave. That distinction matters: a body the kernel builds and then meshes
// badly is a tessellation bug to fix (CLAUDE.md ranks those above features), never a fixture to
// enshrine — whereas a self-crossing trim boundary is malformed geometry no triangulation can cover,
// so the tessellator's only correct answer is to ship what it can AND say so.
//
// Used by the Oblikovati/Oblikovati#2058 regression tests, which assert that the diagnostic those
// bodies raise reaches the feature reply instead of dying inside kernel/ops.
package degenerate

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// crossedTrimUV is one bow-tie in (u,v): the loop visits the corners out of order, so edge 0→1
// crosses edge 2→3. Its signed area is near zero and no triangulation of it covers a well-defined
// region, which is exactly what the coverage gate is there to catch (kernel/ops/patch_acceptance.go).
var crossedTrimUV = [][2]float64{{0, 0}, {0.6, 1.2}, {0.1, 4}, {0.7, 0.4}}

// CrossedTrimBody returns an open body whose faces each carry a self-crossing trim boundary, so
// tessellating it raises ops.CodePatchCoverage — one Defect per face. It has TWO such faces (the
// same bow-tie at two heights on one elliptical cone) so a consumer that aggregates per-face
// diagnostics is exercised on a repeat, not just on a single occurrence.
//
// The surface is an elliptical cone because it is the analytic surface whose trims go through the
// best-fit-plane ear-clip that runs the coverage gate; a cylinder or sphere takes the metric-CDT
// path, which reports the same malformed trim under a different code.
//
// Example:
//
//	m, _ := ops.TessellateBody(degenerate.CrossedTrimBody(), ops.DefaultQuality())
//	// m.Diagnostics carries two tessellate.patch-coverage defects
func CrossedTrimBody() *topo.Body {
	cone, err := geom.NewEllipticalCone(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 0.5, 0.3)
	if err != nil {
		panic(fmt.Sprintf("degenerate: NewEllipticalCone(apex 0,0,0 axis +Z major +X, angles 0.5/0.3): %v", err))
	}
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("degenerate", "crossed-trim", 0)))
	for i, vOffset := range []float64{0, 6} {
		addCrossedTrimFace(bld, cone, vOffset, i)
	}
	return bld.Build()
}

// addCrossedTrimFace lays the bow-tie loop on s, shifted along v, as face number i of the body.
func addCrossedTrimFace(bld *topo.Builder, s geom.Surface, vOffset float64, i int) {
	pts := make([]math.Point3, len(crossedTrimUV))
	for j, c := range crossedTrimUV {
		pts[j] = s.PointAt(c[0], c[1]+vOffset)
	}
	uses := make([]topo.Use, len(pts))
	verts := make([]*topo.Vertex, len(pts))
	for j, p := range pts {
		verts[j] = bld.AddVertex(p, topo.NewLineage(topo.Tok("degenerate", "v", i*len(pts)+j)))
	}
	for j := range pts {
		k := (j + 1) % len(pts)
		e := bld.AddEdge(geom.NewLineSegment(pts[j], pts[k]), verts[j], verts[k],
			topo.NewLineage(topo.Tok("degenerate", "e", i*len(pts)+j)))
		uses[j] = topo.Fwd(e)
	}
	bld.AddFace(s, topo.NewLineage(topo.Tok("degenerate", "f", i)), topo.OuterLoop(uses...))
}
