// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/math"
)

// A bare closed surface — a whole sphere imported as ONE face with no seam edge (0 loops) — has no
// trim to reduce, so it falls to the full-domain grid. A naive UV grid is NOT watertight on a closed
// surface: it emits a separate column for the periodic seam (u=0 and u=2π trace the same points →
// duplicated edges, welded degree 4) and collapses a pole row to many coincident vertices joined by
// zero-area quads (a high-degree pole). closedDomainMesh fixes both — it WRAPS a periodic seam back
// onto the first column (no duplicate seam) and shares ONE vertex per pole row, fanning its neighbour
// ring (no zero-area triangles) — so the closed surface meshes watertight. On a non-closed surface (no
// periodic seam, no pole row) it reduces to the plain UV grid. (M25 PBI-330)
func closedDomainMesh(s geom.Surface, us, vs []float64) *Mesh {
	cols := len(us)
	// One model-relative seam/pole weld for this surface (ADR-0042, #1399). Seam columns and pole
	// rows are coincident BY CONSTRUCTION, so the tolerance cannot come from their own (degenerate)
	// extent — it is derived from the whole surface's extent, so a km-scale sphere's seam still reads
	// as coincident instead of cracking under a cm-anchored 1e-9.
	weld := surfaceGridWeld(s, us, vs)
	uWrap := cols > 2 && columnsCoincide(s, us[0], us[cols-1], vs, weld)
	if uWrap {
		cols-- // the last column repeats the first (the seam); cells wrap onto column 0 instead
	}
	m := &Mesh{}
	poles := poleRowVertices(m, s, us[:cols], vs, weld)
	idx := closedGridVertices(m, s, us[:cols], vs, poles)
	emitClosedGrid(m, idx, uWrap)
	return m
}

// surfaceGridWeld is the surface's model-relative vertex-weld tolerance (#1399): derived from the
// extent of the parameter-grid boundary (the surface's own size), so seam and pole coincidence tests
// scale with the body rather than reading a cm-anchored absolute epsilon.
func surfaceGridWeld(s geom.Surface, us, vs []float64) float64 {
	pts := make([]math.Point3, 0, 2*len(us)+2*len(vs))
	for _, u := range us {
		pts = append(pts, s.PointAt(u, vs[0]), s.PointAt(u, vs[len(vs)-1]))
	}
	for _, v := range vs {
		pts = append(pts, s.PointAt(us[0], v), s.PointAt(us[len(us)-1], v))
	}
	return geom.ResolutionForPoints(pts).Weld()
}

// columnsCoincide reports whether the surface columns at parameters u0 and u1 trace the same points at
// every v — a periodic seam (u0=0, u1=2π on a sphere/cylinder). weld is the surface-relative tolerance.
func columnsCoincide(s geom.Surface, u0, u1 float64, vs []float64, weld float64) bool {
	for _, v := range vs {
		if s.PointAt(u0, v).DistanceTo(s.PointAt(u1, v)) > weld {
			return false
		}
	}
	return true
}

// poleRowVertices returns, per v-row, a single shared vertex index when that row degenerates to one
// point (a pole: every column coincides within weld), or -1 for a normal row. Sharing one vertex (not
// one per column) is what removes the high-degree pole the naive grid produces.
func poleRowVertices(m *Mesh, s geom.Surface, us, vs []float64, weld float64) []int {
	out := make([]int, len(vs))
	for j, v := range vs {
		out[j] = -1
		if rowIsPole(s, us, v, weld) {
			out[j] = m.AddVertex(s.PointAt(us[0], v), s.NormalAt(us[0], v))
		}
	}
	return out
}

// rowIsPole reports whether every column of the v-row collapses to one point (a surface pole) within
// the surface-relative weld tolerance.
func rowIsPole(s geom.Surface, us []float64, v float64, weld float64) bool {
	p0 := s.PointAt(us[0], v)
	for _, u := range us[1:] {
		if s.PointAt(u, v).DistanceTo(p0) > weld {
			return false
		}
	}
	return true
}

// closedGridVertices builds the cols×rows vertex-index grid, reusing the shared pole vertex on a pole
// row so adjacent cells fan to one point instead of meshing zero-area quads.
func closedGridVertices(m *Mesh, s geom.Surface, us, vs []float64, poles []int) [][]int {
	idx := make([][]int, len(us))
	for i, u := range us {
		idx[i] = make([]int, len(vs))
		for j, v := range vs {
			if poles[j] >= 0 {
				idx[i][j] = poles[j]
				continue
			}
			idx[i][j] = m.AddVertex(s.PointAt(u, v), s.NormalAt(u, v))
		}
	}
	return idx
}

// emitClosedGrid triangulates the vertex grid, wrapping the last column onto the first when uWrap, and
// skipping the zero-area triangle of a cell that touches a collapsed pole vertex.
func emitClosedGrid(m *Mesh, idx [][]int, uWrap bool) {
	cols := len(idx)
	for i := range cols {
		ni := i + 1
		if ni >= cols {
			if !uWrap {
				break // open seam: no cell past the last column
			}
			ni = 0
		}
		for j := 0; j+1 < len(idx[i]); j++ {
			emitClosedTri(m, idx[i][j], idx[ni][j], idx[ni][j+1])
			emitClosedTri(m, idx[i][j], idx[ni][j+1], idx[i][j+1])
		}
	}
}

// emitClosedTri adds triangle abc wound to agree with its vertices' surface normals, skipping it when a
// collapsed pole vertex makes two corners coincide (a zero-area triangle).
func emitClosedTri(m *Mesh, a, b, c int) {
	if a == b || b == c || a == c {
		return
	}
	if probe.WindingOpposesNormals(m.Positions[a], m.Positions[b], m.Positions[c], m.Normals[a], m.Normals[b], m.Normals[c]) {
		b, c = c, b
	}
	m.AddTriangle(a, b, c)
}
