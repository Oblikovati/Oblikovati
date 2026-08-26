// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
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

// FreeEdgeCount welds coincident vertices at the MODEL's own resolution and counts mesh edges not used
// by exactly two triangles — the watertightness metric (0 = a closed manifold surface). It is
// FoldEdgeCount's sibling: a mesh must be free-edge-free to be closed and fold-free to bound a
// well-defined volume.
//
// The weld grid is model-relative (ADR-0042), and that is load-bearing rather than stylistic. A fixed
// absolute grid — the 1e-6 several test-local copies used to carry — over-merges whenever the model's
// own feature separation drops below it, and an over-merge reports as a free edge just like a crack
// does: on the #1818 near-pinch crossing (R=3, |Δr|=2e-5) at PropertyQuality the fixed grid collapsed
// pairs of distinct neck vertices measured 6.3e-7 apart and produced EIGHT 4-incident edges, every one
// of degree 4 (an over-merge) and none of degree 1 (a crack). At the model's own weld resolution
// (1.039e-8 there) not one edge is anything but 2-incident: that mesh is watertight.
//
// Example: FreeEdgeCount(mesh) == 0 is required for a tessellation to be a closed surface.
func FreeEdgeCount(m *Mesh) int {
	return weldedFreeEdgeCount(m)
}

// weldedFreeEdgeCount welds coincident vertices (by [weldKey]) and counts edges not shared by exactly
// two triangles — the watertightness metric for a single mesh.
func weldedFreeEdgeCount(m *Mesh) int {
	grid := ResolutionForPoints(m.Positions).Weld()
	canon := map[[3]int64]int{}
	weld := make([]int, len(m.Positions))
	for i, p := range m.Positions {
		k := weldKey(p, grid)
		if c, ok := canon[k]; ok {
			weld[i] = c
		} else {
			canon[k], weld[i] = i, i
		}
	}
	deg := map[[2]int]int{}
	for t := 0; 3*t+2 < len(m.Indices); t++ {
		v := [3]int{weld[m.Indices[3*t]], weld[m.Indices[3*t+1]], weld[m.Indices[3*t+2]]}
		for k := range 3 {
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

// metricPatchMesh meshes an analytic curved trim (torus/sphere/cone/cylinder) over its OWN (u,v) with
// a METRIC-SCALED constrained-Delaunay triangulation — the same anti-fold approach nurbsPcurveMesh
// uses for B-splines. gridPatchMesh's plain (u,v) CDT folds where the surface metric is anisotropic
// (a torus's tube vs ring, a sphere near its poles): u and v have very different 3D scales, so an
// isotropic Delaunay twists in 3D. Scaling each axis by its mean 3D length (√E, √G via metricScale)
// makes the parameter space ≈ isometric to 3D, so the triangulation is well-shaped and fold-free.
// Boundary loops keep their exact 3D edge points (watertight); interior nodes are deflection-adaptive;
// residual folds are swept by repairFolds. Falls back to the best-fit-plane boundary triangulation if
// the CDT degenerates or tears (a pole/seam-distorted cap).
func metricPatchMesh(s geom.Surface, q Quality, outer3D []math.Point3, holes3D [][]math.Point3, outerUV []math.Point2, holesUV [][]math.Point2) *Mesh {
	su, sv := trimMetricScale(s, outerUV)
	m, loops := metricCDTPatch(s, su, sv, q, outer3D, outerUV, holes3D, holesUV, 1)
	if m == nil {
		return boundaryPatchMesh(s, outer3D, holes3D)
	}
	if patchIsManifold(m, loops) && FoldEdgeCount(m) == 0 {
		return m
	}
	// The metric CDT tore (a pole/seam-distorted cap) or still folds (a cap whose boundary touches
	// the sphere pole — the degenerate sliver where all u collapse, which repairFolds can't flip).
	// The boundary-only best-fit-plane triangulation is watertight and, for such a small cap,
	// fold-free; keep the metric mesh only when it is manifold and folds no more than the fallback.
	fallback := boundaryPatchMesh(s, outer3D, holes3D)
	if !patchIsManifold(m, loops) || FoldEdgeCount(fallback) < FoldEdgeCount(m) {
		return fallback
	}
	return m
}

// metricCDTPatch is the shared metric-scaled CDT builder behind both the B-spline fold-driven loop
// and the analytic metricPatchMesh: it lays the exact 3D boundary loops plus deflection-adaptive
// interior nodes (at the given refine factor) into metric-scaled (u,v), constrained-Delaunay
// triangulates them, lifts to 3D and sweeps residual folds. It returns the mesh and the loop index
// sequences (so a caller can run patchIsManifold), or a nil mesh when the CDT yields nothing.
func metricCDTPatch(s geom.Surface, su, sv float64, q Quality, outer3D []math.Point3, outerUV []math.Point2, holes3D [][]math.Point3, holesUV [][]math.Point2, refine float64) (*Mesh, [][]int) {
	b := newPatchBuilder(s, su, sv)
	loops := [][]int{b.addLoop(outer3D, outerUV)}
	for i := range holes3D {
		loops = append(loops, b.addLoop(holes3D[i], holesUV[i]))
	}
	nodes, saturated := adaptiveInteriorNodes(s, outerUV, holesUV, q, refine, true)
	for _, g := range nodes {
		b.addInterior(g)
	}
	tris := constrainedDelaunay(b.scaled, loops)
	if len(tris) == 0 {
		return nil, loops
	}
	m := patchMeshFrom(b.pos, b.nrm, tris)
	repairFolds(m, 8)
	recordCapSaturation(m, saturated, q)
	return m, loops
}

// trimMetricScale returns the per-axis (u,v) metric (mean 3D length of a unit u/v step) sampled over
// the TRIM region's (u,v) bounding box, not the surface's whole domain. This matters where a surface's
// metric varies across the domain: a cone's u-step length (the circumference) shrinks to zero toward
// the apex, so the whole-domain mean (metricScale) is dominated by the degenerate apex and the scaled
// CDT folds — but a cone-frustum trim sits away from the apex, where the local metric is well behaved.
// Falls back to the whole-domain metricScale when the trim bbox is empty/degenerate.
func trimMetricScale(s geom.Surface, outerUV []math.Point2) (su, sv float64) {
	if len(outerUV) == 0 {
		return metricScale(s)
	}
	umin, umax, vmin, vmax := uvBBox(outerUV)
	if umax <= umin || vmax <= vmin {
		return metricScale(s)
	}
	var sumU, sumV float64
	const n = 4
	for i := 0; i <= n; i++ {
		for j := 0; j <= n; j++ {
			du, dv := s.DerivativesAt(umin+(umax-umin)*float64(i)/n, vmin+(vmax-vmin)*float64(j)/n)
			sumU += du.Length()
			sumV += dv.Length()
		}
	}
	su, sv = sumU/float64((n+1)*(n+1)), sumV/float64((n+1)*(n+1))
	if su <= 0 {
		su = 1
	}
	if sv <= 0 {
		sv = 1
	}
	return su, sv
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
	m := patchMeshFrom(pos, nrm, tris)
	repairFolds(m, 8) // a pole/seam-distorted cap's CDT can crease; flip the folding diagonals (#585)
	return m
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

// patchMeshFrom builds the 3D mesh from a CDT's triangles, winding each CONSISTENTLY by its 2D (u,v)
// orientation: a triangle wound CCW in (u,v) faces +NormalAt in 3D (the surface contract is NormalAt ≡
// du×dv normalised, and a CCW (u,v) triangle's geometric normal lies along +du×dv), so the whole patch
// is oriented one way and TessellateFace flips it once for a reversed face. Winding per-triangle by the
// 3D vertex normals instead (windingOpposesNormals) flips UNRELIABLY on a high-curvature freeform patch,
// where a flat triangle's geometric normal is nearly perpendicular to its averaged vertex normals — the
// back-facing red triangles seen on the EDF duct's NURBS walls. uv is the same 2D space the CDT ran in
// (the surface (u,v) or a positively-scaled metric (u,v), so its orientation matches (u,v)).
func patchMeshFrom(pos []math.Point3, nrm []math.Vector3, tris [][3]int) *Mesh {
	m := &Mesh{}
	for i := range pos {
		m.addVertex(pos[i], nrm[i])
	}
	// The CDT returns a CONSISTENTLY-oriented triangulation (every interior edge traversed once each way).
	// Re-winding per-triangle to agree with its vertex normals BREAKS that consistency on a curved patch:
	// a thin sliver's flat geometric normal opposes its smooth vertex normals, so it flips and folds
	// against its neighbours — the back-facing red triangles on the EDF duct's NURBS walls. So keep the
	// CDT's winding and flip the WHOLE patch ONCE if it faces inward overall (an aggregate, area-weighted
	// vote that no single sliver can sway). TessellateFace flips again for a reversed face.
	var agree float64
	for _, t := range tris {
		gn := pos[t[0]].VectorTo(pos[t[1]]).Cross(pos[t[0]].VectorTo(pos[t[2]]))
		agree += float64(gn.Dot(nrm[t[0]].Add(nrm[t[1]]).Add(nrm[t[2]])))
	}
	flip := agree < 0
	for _, t := range tris {
		if flip {
			m.addTriangle(t[0], t[2], t[1])
		} else {
			m.addTriangle(t[0], t[1], t[2])
		}
	}
	return m
}
