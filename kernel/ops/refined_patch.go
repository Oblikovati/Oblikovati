// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati/kernel/geom"
	"oblikovati/math"
)

// gridPatchMesh meshes an analytic curved patch (a sphere cap) over its OWN (u,v) parameter space
// with INTERIOR nodes, not just a boundary fan. The trim loops (exact 3D points, so the patch stays
// watertight with its neighbours) plus a staggered interior (u,v) grid are constrained-Delaunay
// triangulated in (u,v); interior points + per-vertex surface normals make the cap read as a smooth
// curved surface instead of the flat radiating fan a boundary-only triangulation produces (the EDF
// inner bell-mouth slivers). Mirrors OpenCASCADE's BRepMesh range-splitter approach (interior nodes
// on a deflection-spaced staggered grid within the trimmed range). Caller must have a valid (u,v)
// (toUVLoops ok) — i.e. the patch does not straddle a pole/seam.
func gridPatchMesh(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, outerUV []math.Point2, holesUV [][]math.Point2) *Mesh {
	uv, pos, nrm, loops := patchLoops2D(s, outer3D, holes3D, outerUV, holesUV)
	for _, g := range interiorUVGrid(outerUV, holesUV) {
		uv = append(uv, g)
		pos = append(pos, s.PointAt(g[0], g[1]))
		nrm = append(nrm, s.NormalAt(g[0], g[1]))
	}
	tris := constrainedDelaunay(uv, loops)
	if len(tris) == 0 {
		return boundaryPatchMesh(s, outer3D, holes3D)
	}
	m := patchMeshFrom(pos, nrm, tris)
	if !patchIsManifold(m, loops) {
		// The cap's (u,v) is degenerate — it reaches the sphere pole (v=±π/2, where all u collapse) or
		// wraps the seam — so the CDT in that distorted space tears the interior into a non-manifold
		// mesh (the filleted_box corner caps). Fall back to the best-fit-plane boundary triangulation,
		// which is watertight (no interior nodes, but a small corner octant reads fine).
		return boundaryPatchMesh(s, outer3D, holes3D)
	}
	return m
}

// patchIsManifold reports whether the patch mesh is no LESS watertight than its input boundary: after
// welding coincident 3D vertices, its unpaired (degree≠2) edges must not EXCEED the loops' edge count
// (a clean cap's only unpaired edges ARE its boundary, == the loops). A pole/seam-degenerate cap whose
// CDT tore leaves interior holes (extra unpaired edges) or pole overlaps (degree>2) — both push the
// count over the boundary and trigger the fallback. Welds because the tear shows only in 3D: distinct
// (u,v) nodes coincide at the sphere pole. (A benign over-extraction, count < boundary, is kept.)
func patchIsManifold(m *Mesh, loops [][]int) bool {
	want := 0
	for _, l := range loops {
		want += len(l)
	}
	return weldedFreeEdgeCount(m) <= want
}

// weldedFreeEdgeCount welds coincident vertices (by [weldKey]) and counts edges not shared by exactly
// two triangles — the watertightness metric for a single mesh.
func weldedFreeEdgeCount(m *Mesh) int {
	canon := map[[3]int64]int{}
	weld := make([]int, len(m.Positions))
	for i, p := range m.Positions {
		k := weldKey(p)
		if c, ok := canon[k]; ok {
			weld[i] = c
		} else {
			canon[k], weld[i] = i, i
		}
	}
	deg := map[[2]int]int{}
	for t := 0; 3*t+2 < len(m.Indices); t++ {
		v := [3]int{weld[m.Indices[3*t]], weld[m.Indices[3*t+1]], weld[m.Indices[3*t+2]]}
		for k := 0; k < 3; k++ {
			a, b := v[k], v[(k+1)%3]
			if a > b {
				a, b = b, a
			}
			deg[[2]int{a, b}]++
		}
	}
	free := 0
	for _, d := range deg {
		if d != 2 {
			free++
		}
	}
	return free
}

// metricWallMesh triangulates a trimmed analytic wall (a cylinder/cone) in its METRIC-SCALED (u,v) —
// each axis scaled by its mean 3D length (√E, √G) so the strongly anisotropic parameter space (u angle,
// v length) becomes ≈isometric to 3D. That gives a well-shaped, deflection-bounded interior (the same
// metric triangulation the NURBS mesher uses), where the best-fit-plane ear-clip tore a curved wall
// that lies in no plane (the EDF trimmed-cylinder leaks) and the ISOTROPIC interior grid exploded the
// node count. Exact 3D boundary points keep it watertight with neighbours; folds are repaired; falls
// back to the plane ear-clip if the CDT is empty or tears (patchIsManifold).
func metricWallMesh(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, outerUV []math.Point2, holesUV [][]math.Point2, q Quality) *Mesh {
	su, sv := metricScale(s)
	b := newPatchBuilder(s, su, sv)
	loops := [][]int{b.addLoop(outer3D, outerUV)}
	for i := range holes3D {
		loops = append(loops, b.addLoop(holes3D[i], holesUV[i]))
	}
	for _, g := range adaptiveInteriorNodes(s, outerUV, holesUV, q) {
		b.addInterior(g)
	}
	tris := constrainedDelaunay(b.scaled, loops)
	if len(tris) == 0 {
		return boundaryPatchMesh(s, outer3D, holes3D)
	}
	m := patchMeshFrom(b.pos, b.nrm, tris)
	repairFolds(m, 8)
	if !patchIsManifold(m, loops) {
		return boundaryPatchMesh(s, outer3D, holes3D)
	}
	return m
}

// interiorUVGrid returns staggered (u,v) points strictly inside the trim (inside the outer loop,
// outside the holes), on a grid sized to the outer loop's median edge length so the interior density
// matches the boundary's. Alternate rows are offset half a step for better-shaped triangles.
func interiorUVGrid(outer []math.Point2, holes [][]math.Point2) [][2]float64 {
	umin, umax, vmin, vmax, step := uvBounds(outer)
	if step <= 0 {
		return nil
	}
	var pts [][2]float64
	row := 0
	for v := vmin + step/2; v < vmax; v += step {
		off := 0.0
		if row%2 == 1 {
			off = step / 2
		}
		row++
		for u := umin + step/2 + off; u < umax; u += step {
			p := [2]float64{u, v}
			if insideUVTrim(outer, holes, p) {
				pts = append(pts, p)
			}
		}
	}
	return pts
}

func uvBounds(outer []math.Point2) (umin, umax, vmin, vmax, step float64) {
	umin, vmin = stdmath.Inf(1), stdmath.Inf(1)
	umax, vmax = stdmath.Inf(-1), stdmath.Inf(-1)
	var lens []float64
	for i, p := range outer {
		x, y := float64(p.X), float64(p.Y)
		umin, umax = stdmath.Min(umin, x), stdmath.Max(umax, x)
		vmin, vmax = stdmath.Min(vmin, y), stdmath.Max(vmax, y)
		q := outer[(i+1)%len(outer)]
		lens = append(lens, stdmath.Hypot(float64(q.X)-x, float64(q.Y)-y))
	}
	return umin, umax, vmin, vmax, medianLen(lens)
}

func medianLen(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	c := append([]float64(nil), xs...)
	for i := range c {
		for j := i + 1; j < len(c); j++ {
			if c[j] < c[i] {
				c[i], c[j] = c[j], c[i]
			}
		}
	}
	return c[len(c)/2]
}

func insideUVTrim(outer []math.Point2, holes [][]math.Point2, p [2]float64) bool {
	if !pointInUVPoly(outer, p) {
		return false
	}
	for _, h := range holes {
		if pointInUVPoly(h, p) {
			return false
		}
	}
	return true
}

func pointInUVPoly(poly []math.Point2, p [2]float64) bool {
	in := false
	n := len(poly)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		yi, yj := float64(poly[i].Y), float64(poly[j].Y)
		if (yi > p[1]) != (yj > p[1]) {
			xi, xj := float64(poly[i].X), float64(poly[j].X)
			if p[0] < (xj-xi)*(p[1]-yi)/(yj-yi)+xi {
				in = !in
			}
		}
	}
	return in
}

// trimmedPatchMesh meshes a non-rectangular curved patch via a constrained Delaunay triangulation
// of its boundary loops. The 2D embedding to triangulate in is chosen by patchProjection: the
// surface's own (u,v) for a B-spline (where the trim loops are a simple polygon), or the boundary's
// best-fit plane for an analytic surface (whose (u,v) degenerates at a pole/seam). The CDT is robust
// where boundary-only ear-clipping tears (boundary segments are recovered by edge flips) and exact
// on concave trims and holes (the domain flood respects the constrained edges); it meshes the real
// trim region, not the surface's whole UV domain (which fullDomainGridMesh did — the torn full-sphere
// fan). No interior Steiner points: ParamAt's distorted (u,v) makes a freshly sampled interior
// point's PointAt land off the patch and inflate the mesh, so refinement stays boundary-only and the
// exact 3D boundary points are kept (watertight with neighbour faces). Falls back to
// boundaryPatchMesh if the CDT yields nothing.
func trimmedPatchMesh(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) *Mesh {
	outer2D, holes2D := patchProjection(s, outer3D, holes3D)
	uv, pos, nrm, loops := patchLoops2D(s, outer3D, holes3D, outer2D, holes2D)
	tris := constrainedDelaunay(uv, loops)
	if len(tris) == 0 {
		return boundaryPatchMesh(s, outer3D, holes3D)
	}
	return patchMeshFrom(pos, nrm, tris)
}

// patchLoops2D pairs each boundary loop's chosen 2D embedding (outer2D/holes2D) with its exact 3D
// point and surface normal, returning the (u,v/plane) coords, parallel positions and normals, and
// the loops as index sequences for the CDT.
func patchLoops2D(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, outer2D []math.Point2, holes2D [][]math.Point2) (uv [][2]float64, pos []math.Point3, nrm []math.Vector3, loops [][]int) {
	loops3D := append([][]math.Point3{outer3D}, holes3D...)
	loops2D := append([][]math.Point2{outer2D}, holes2D...)
	for li, loop := range loops3D {
		idx := make([]int, len(loop))
		for i, p := range loop {
			u, v := s.ParamAt(p)
			idx[i] = len(uv)
			uv = append(uv, [2]float64{float64(loops2D[li][i].X), float64(loops2D[li][i].Y)})
			pos = append(pos, p)
			nrm = append(nrm, s.NormalAt(u, v))
		}
		loops = append(loops, idx)
	}
	return uv, pos, nrm, loops
}

// patchMeshFrom builds the 3D mesh from the CDT triangles, winding each to agree with its own
// vertex normals (consistent on a curved patch — see windingOpposesNormals).
func patchMeshFrom(pos []math.Point3, nrm []math.Vector3, tris [][3]int) *Mesh {
	m := &Mesh{}
	for i := range pos {
		m.addVertex(pos[i], nrm[i])
	}
	for _, t := range tris {
		a, b, c := t[0], t[1], t[2]
		if windingOpposesNormals(pos[a], pos[b], pos[c], nrm[a], nrm[b], nrm[c]) {
			b, c = c, b
		}
		m.addTriangle(a, b, c)
	}
	return m
}
