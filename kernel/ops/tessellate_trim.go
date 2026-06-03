// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"sort"

	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
)

// Trimmed curved-face tessellation (third piece of the curved-B-rep stack). A curved
// face is meshed over its trim region, not the surface's whole UV domain, by mapping its
// boundary loops (shared edge discretization, so they match neighbours exactly) into
// (u,v) space with Surface.ParamAt. When that region is an iso-aligned rectangle whose
// opposite edges sample identically — the shape of every analytic fillet/blend face and
// of axial cylinder/cone walls — a STRUCTURED grid of thin iso quads tessellates it
// watertight and with correct curved area. (Ear-clipping the boundary instead would chord
// long triangles across the curvature and get the area wrong.) Anything else — holes, a
// non-rectangular trim, a seam-crossing periodic loop — falls back to the full-domain grid
// (a follow-up; needs a constrained triangulation).

const trimBorderTol = 1e-6

// tessellateCurvedFace meshes a curved face's trimmed region (see file doc).
func tessellateCurvedFace(f *topo.Face, q Quality) *Mesh {
	s := f.Geometry()
	outer3D := faceOuterBoundary(f, q)
	holes3D := faceHoleBoundaries(f, q)
	if len(outer3D) < 3 {
		return fullDomainGridMesh(s, q)
	}
	outerUV, holesUV, ok := toUVLoops(s, outer3D, holes3D)
	if !ok {
		return fullDomainGridMesh(s, q) // seam-crossing periodic face
	}
	if len(holesUV) == 0 {
		if us, vs, isRect := isoRectangleGrid(outerUV); isRect {
			return structuredGridMesh(s, us, vs) // cylinder/cone wall, fillet face: exact area
		}
	}
	// A non-rectangular trim (e.g. a corner sphere patch): triangulate the boundary.
	return boundaryPatchMesh(s, outer3D, holes3D)
}

// boundaryPatchMesh triangulates a curved face from its boundary loops alone (no interior
// Steiner points): the loops are flattened onto their best-fit plane (NOT the surface's own
// (u,v), which can be degenerate — e.g. a sphere patch corner landing on the lat/long pole),
// ear-clipped there, and lifted back to their exact 3D boundary points, each triangle wound
// outward. A coarse but watertight covering of the exact trim region — right for small patches
// (corner blends); larger non-rectangular curved faces would want a refined triangulation.
func boundaryPatchMesh(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) *Mesh {
	outer2D, holes2D := projectToPlane(outer3D, holes3D)
	uv, pos := outer2D, outer3D
	if len(holes2D) > 0 {
		uv, pos = mergeHoles(outer2D, outer3D, holes2D, holes3D)
	}
	m := &Mesh{}
	for _, p := range pos {
		u, v := s.ParamAt(p)
		m.addVertex(p, s.NormalAt(u, v))
	}
	for _, tri := range earClip(uv) {
		a, b, c := tri[0], tri[1], tri[2]
		if triangleFlipped(s, pos[a], pos[b], pos[c]) {
			b, c = c, b
		}
		m.addTriangle(a, b, c)
	}
	return m
}

// projectToPlane flattens the boundary loops onto the outer loop's best-fit plane (Newell
// normal + an in-plane basis) — a non-degenerate 2D embedding of a single-valued patch.
func projectToPlane(outer3D []math.Point3, holes3D [][]math.Point3) ([]math.Point2, [][]math.Point2) {
	n := newellUnit(outer3D)
	e1, e2 := planeBasis(n)
	o := outer3D[0]
	flat := func(p math.Point3) math.Point2 {
		d := o.VectorTo(p)
		return math.P2(d.Dot(e1), d.Dot(e2))
	}
	outer2D := make([]math.Point2, len(outer3D))
	for i, p := range outer3D {
		outer2D[i] = flat(p)
	}
	holes2D := make([][]math.Point2, len(holes3D))
	for i, h := range holes3D {
		hp := make([]math.Point2, len(h))
		for j, p := range h {
			hp[j] = flat(p)
		}
		holes2D[i] = hp
	}
	return outer2D, holes2D
}

// newellUnit returns a loop's unit normal by Newell's method (robust for non-planar loops).
func newellUnit(loop []math.Point3) math.Vector3 {
	var nx, ny, nz float64
	n := len(loop)
	for i := 0; i < n; i++ {
		c, d := loop[i], loop[(i+1)%n]
		nx += (c.Y - d.Y) * (c.Z + d.Z)
		ny += (c.Z - d.Z) * (c.X + d.X)
		nz += (c.X - d.X) * (c.Y + d.Y)
	}
	u, err := math.UnitVector3FromVector(math.V3(nx, ny, nz))
	if err != nil {
		return math.V3(0, 0, 1)
	}
	return u.AsVector()
}

// planeBasis returns two orthonormal in-plane vectors for the plane with unit normal n.
func planeBasis(n math.Vector3) (e1, e2 math.Vector3) {
	seed := math.V3(1, 0, 0)
	if stdmath.Abs(n.X) > 0.9 {
		seed = math.V3(0, 1, 0)
	}
	a, err := math.UnitVector3FromVector(n.Cross(seed))
	if err != nil {
		return math.V3(1, 0, 0), math.V3(0, 1, 0)
	}
	return a.AsVector(), n.Cross(a.AsVector())
}

// triangleFlipped reports whether triangle abc winds against the surface normal at its
// centroid (so it should be reversed to face outward).
func triangleFlipped(s geom.Surface, a, b, c math.Point3) bool {
	n := a.VectorTo(b).Cross(a.VectorTo(c))
	cen := math.P3((a.X+b.X+c.X)/3, (a.Y+b.Y+c.Y)/3, (a.Z+b.Z+c.Z)/3)
	u, v := s.ParamAt(cen)
	return n.Dot(s.NormalAt(u, v)) < 0
}

// isoRectangleGrid returns the sorted u and v grid lines when the UV boundary is an
// iso-aligned rectangle whose opposite edges carry matching parameter samples (so a
// structured grid is watertight and conforms to the boundary). ok=false otherwise.
func isoRectangleGrid(loop []math.Point2) (us, vs []float64, ok bool) {
	uMin, uMax, vMin, vMax := bounds2D(loop)
	var bottomU, topU, leftV, rightV []float64
	for _, p := range loop {
		onB, onT := near(p.Y, vMin), near(p.Y, vMax)
		onL, onR := near(p.X, uMin), near(p.X, uMax)
		if !onB && !onT && !onL && !onR {
			return nil, nil, false // a vertex off the bbox border — not a rectangle
		}
		appendIf(&bottomU, p.X, onB)
		appendIf(&topU, p.X, onT)
		appendIf(&leftV, p.Y, onL)
		appendIf(&rightV, p.Y, onR)
	}
	bottomU, topU = sortUnique(bottomU), sortUnique(topU)
	leftV, rightV = sortUnique(leftV), sortUnique(rightV)
	if !sameGrid(bottomU, topU) || !sameGrid(leftV, rightV) {
		return nil, nil, false // opposite edges sample differently → would leave T-junctions
	}
	return bottomU, leftV, true
}

// structuredGridMesh tessellates the surface over the us×vs parameter grid as thin iso
// quads (two triangles each), wound outward with true per-vertex normals. Border grid
// points reproduce the exact boundary vertices (ParamAt is the inverse of PointAt), so the
// mesh conforms to the shared edge discretization.
func structuredGridMesh(s geom.Surface, us, vs []float64) *Mesh {
	m := &Mesh{}
	idx := make([][]int, len(us))
	for i, u := range us {
		idx[i] = make([]int, len(vs))
		for j, v := range vs {
			idx[i][j] = m.addVertex(s.PointAt(u, v), s.NormalAt(u, v))
		}
	}
	for i := 0; i+1 < len(us); i++ {
		for j := 0; j+1 < len(vs); j++ {
			emitCellOutward(m, s, us[i], us[i+1], vs[j], vs[j+1], idx[i][j], idx[i+1][j], idx[i+1][j+1], idx[i][j+1])
		}
	}
	return m
}

// emitCellOutward adds the two triangles of a grid cell, winding them so their geometric
// normal agrees with the surface normal at the cell centre.
func emitCellOutward(m *Mesh, s geom.Surface, u0, u1, v0, v1 float64, a, b, c, d int) {
	flip := m.cellNormal(a, b, c).Dot(s.NormalAt((u0+u1)/2, (v0+v1)/2)) < 0
	if flip {
		m.addTriangle(a, c, b)
		m.addTriangle(a, d, c)
		return
	}
	m.addTriangle(a, b, c)
	m.addTriangle(a, c, d)
}

// cellNormal returns the (unnormalized) normal of triangle abc by position.
func (m *Mesh) cellNormal(a, b, c int) math.Vector3 {
	pa, pb, pc := m.Positions[a], m.Positions[b], m.Positions[c]
	return pa.VectorTo(pb).Cross(pa.VectorTo(pc))
}

// toUVLoops maps the boundary loops to parameter space, unwrapping periodic parameters so
// a loop reads as a contiguous polygon; ok=false if a loop wraps the full seam.
func toUVLoops(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) (outer []math.Point2, holes [][]math.Point2, ok bool) {
	uPer, vPer := isPeriodic(s.UDomain()), isPeriodic(s.VDomain())
	if outer, ok = toUVLoop(s, outer3D, uPer, vPer); !ok {
		return nil, nil, false
	}
	for _, h := range holes3D {
		hu, hok := toUVLoop(s, h, uPer, vPer)
		if !hok {
			return nil, nil, false
		}
		holes = append(holes, hu)
	}
	return outer, holes, true
}

// toUVLoop inverts each 3D loop point to (u,v) and unwraps periodic parameters.
func toUVLoop(s geom.Surface, loop []math.Point3, uPer, vPer bool) ([]math.Point2, bool) {
	us := make([]float64, len(loop))
	vs := make([]float64, len(loop))
	for i, p := range loop {
		us[i], vs[i] = s.ParamAt(p)
	}
	if uPer {
		var ok bool
		if us, ok = unwrap(us); !ok {
			return nil, false
		}
	}
	if vPer {
		var ok bool
		if vs, ok = unwrap(vs); !ok {
			return nil, false
		}
	}
	out := make([]math.Point2, len(loop))
	for i := range loop {
		out[i] = math.P2(us[i], vs[i])
	}
	return out, true
}

// unwrap removes 2π jumps so a periodic parameter reads continuously; ok=false when the
// total span reaches 2π (the loop wraps the seam and is not a simple polygon).
func unwrap(a []float64) ([]float64, bool) {
	out := make([]float64, len(a))
	out[0] = a[0]
	lo, hi := a[0], a[0]
	for i := 1; i < len(a); i++ {
		d := a[i] - a[i-1]
		for d > stdmath.Pi {
			d -= 2 * stdmath.Pi
		}
		for d <= -stdmath.Pi {
			d += 2 * stdmath.Pi
		}
		out[i] = out[i-1] + d
		lo, hi = stdmath.Min(lo, out[i]), stdmath.Max(hi, out[i])
	}
	return out, hi-lo < 2*stdmath.Pi-1e-6
}

// isPeriodic reports whether a [0, 2π] parameter domain wraps.
func isPeriodic(lo, hi float64) bool {
	return stdmath.Abs(lo) < 1e-9 && stdmath.Abs(hi-2*stdmath.Pi) < 1e-9
}

// bounds2D returns the UV bounding box of the points.
func bounds2D(pts []math.Point2) (uMin, uMax, vMin, vMax float64) {
	uMin, vMin = stdmath.Inf(1), stdmath.Inf(1)
	uMax, vMax = stdmath.Inf(-1), stdmath.Inf(-1)
	for _, p := range pts {
		uMin, uMax = stdmath.Min(uMin, p.X), stdmath.Max(uMax, p.X)
		vMin, vMax = stdmath.Min(vMin, p.Y), stdmath.Max(vMax, p.Y)
	}
	return uMin, uMax, vMin, vMax
}

func near(a, b float64) bool { return stdmath.Abs(a-b) < trimBorderTol }

func appendIf(dst *[]float64, x float64, cond bool) {
	if cond {
		*dst = append(*dst, x)
	}
}

// sortUnique returns the values sorted ascending with near-duplicates collapsed.
func sortUnique(xs []float64) []float64 {
	sort.Float64s(xs)
	out := xs[:0:0]
	for _, x := range xs {
		if len(out) == 0 || !near(out[len(out)-1], x) {
			out = append(out, x)
		}
	}
	return out
}

// sameGrid reports whether two sorted grids have the same lines (within tolerance).
func sameGrid(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !near(a[i], b[i]) {
			return false
		}
	}
	return true
}

// fullDomainGridMesh samples the surface over its whole (clamped) UV domain — the fallback
// for a face whose trim can't be reduced to a conforming iso rectangle.
func fullDomainGridMesh(s geom.Surface, q Quality) *Mesh {
	uLo, uHi := clampSpan(s.UDomain())
	vLo, vHi := clampSpan(s.VDomain())
	us := adaptiveParams(func(u float64) math.Point3 { return s.PointAt(u, (vLo+vHi)/2) }, uLo, uHi, q.tol())
	vs := adaptiveParams(func(v float64) math.Point3 { return s.PointAt((uLo+uHi)/2, v) }, vLo, vHi, q.tol())
	return gridMesh(s, us, vs)
}
