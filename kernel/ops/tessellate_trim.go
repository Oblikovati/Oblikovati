// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
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
	if _, isSpline := s.(geom.BSplineSurface); isSpline {
		if m := nurbsPcurveMesh(f, q); m != nil { // M25: metric-aware (u,v) triangulation
			return m
		}
	}
	outer3D := faceOuterBoundary(f, q)
	holes3D := faceHoleBoundaries(f, q)
	if len(outer3D) < 3 {
		return fullDomainGridMesh(s, q)
	}
	if m, isFan := coneApexFan(s, outer3D); isFan {
		return m // a cone closing to its apex (a drill point): a fan from the apex to the rim
	}
	if m, isCap := sphereCapFan(s, outer3D, q); isCap {
		return m // a sphere cut by one plane (a cap): rings from the rim to the enclosed pole
	}
	outerUV, holesUV, ok := toUVLoops(s, outer3D, holes3D)
	if !ok {
		return meshSeamCrossingFace(f, s, outer3D, holes3D, q) // a loop wrapping the seam: band/cap fallbacks
	}
	if us, vs, isRect := isoRectangleGrid(outerUV); len(holesUV) == 0 && isRect {
		return structuredGridMesh(s, us, vs) // cylinder/cone wall, fillet face: exact area
	}
	return nonRectangularMesh(s, q, outer3D, holes3D, outerUV, holesUV)
}

// meshSeamCrossingFace meshes a curved face whose boundary loop wraps the periodic seam (so toUVLoops
// can't unwrap it): a full cylinder/cone side or a torus rim-fillet band closes the seam watertight via
// closedDomainMesh; a singly-periodic sphere cap straddling the pole goes through the best-fit-plane CDT
// (the full-domain grid tears there); a doubly-periodic torus we can't reduce keeps the full-domain grid.
func meshSeamCrossingFace(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) *Mesh {
	if us, vs, isBand := periodicBandGrid(s, outer3D, holes3D); isBand {
		return closedDomainMesh(s, us, vs) // full cylinder/cone side: wraps the seam watertight
	}
	if _, _, isBand := doublyPeriodicBandGrid(s, outer3D, holes3D); isBand {
		if m, ok := closedBandLoftMesh(f, s, q); ok {
			return m // torus rim-fillet band: loft so each edge ring keeps its own (differing) tessellation
		}
		return fullDomainGridMesh(s, q) // shouldn't reach: a doubly-periodic band that isn't two circles + a seam
	}
	if isPeriodic(s.UDomain()) != isPeriodic(s.VDomain()) {
		return trimmedPatchMesh(s, outer3D, holes3D) // sphere cap on the pole: CDT in the best-fit plane
	}
	return fullDomainGridMesh(s, q) // doubly-periodic / aperiodic seam face we can't reduce
}

// nonRectangularMesh meshes a non-iso-rectangular curved trim. Every analytic curved surface (torus,
// sphere, cylinder, cone) AND the B-spline trim go through metricPatchMesh — a trim-local metric-scaled
// (u,v) CDT WITH curvature-adaptive interior Steiner points (#585, #1323 L3) — so a larger freeform
// trim's interior is refined to the chord tolerance, not chorded flat across the boundary loops.
// Anything else keeps the best-fit-plane ear-clip (boundaryPatchMesh).
func nonRectangularMesh(s geom.Surface, q Quality, outer3D []math.Point3, holes3D [][]math.Point3, outerUV []math.Point2, holesUV [][]math.Point2) *Mesh {
	switch s.(type) {
	case geom.Torus, geom.Sphere, geom.Cylinder, geom.Cone, geom.BSplineSurface:
		// These trims fold when flattened to a best-fit plane (boundaryPatchMesh) or meshed over a plain
		// anisotropic (u,v) (gridPatchMesh): a torus's ring-vs-tube, a sphere near its poles, a trimmed
		// cyl/cone, a freeform B-spline. metricPatchMesh triangulates in a TRIM-LOCAL metric-scaled (u,v)
		// (√E,√G over the trim's own (u,v) bbox, so even a cone — whose metric degenerates only toward
		// the far-off apex — stays well conditioned) with deflection-adaptive interior nodes kept
		// strictly inside the trim (adaptiveInteriorNodes/clearOfTrim), plus repairFolds and a
		// boundary-only fallback. This was the bulk of the EDF over-enclosure (#585) and, for B-splines,
		// removes the interior chord error of the old boundary-only triangulation (#1323 L3).
		return metricPatchMesh(s, q, outer3D, holes3D, outerUV, holesUV)
	}
	return boundaryPatchMesh(s, outer3D, holes3D)
}

// boundaryPatchMesh triangulates a curved face from its boundary loops alone (no interior
// Steiner points): the loops are flattened onto their best-fit plane (NOT the surface's own
// (u,v), which can be degenerate — e.g. a sphere patch corner landing on the lat/long pole),
// ear-clipped there, and lifted back to their exact 3D boundary points, each triangle wound
// outward. A coarse but watertight covering of the exact trim region — right for small patches
// (corner blends); larger non-rectangular curved faces would want a refined triangulation.
func boundaryPatchMesh(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) *Mesh {
	outer2D, holes2D := patchProjection(s, outer3D, holes3D)
	pos := outer3D
	var tris [][3]int
	if len(holes2D) > 0 {
		// earcut's indices address the outer loop followed by the holes concatenated, so the
		// 3D buffer is built in that same order (matches the robust planar path, see earcut.go).
		pos = append([]math.Point3(nil), outer3D...)
		for _, h := range holes3D {
			pos = append(pos, h...)
		}
		tris = earcut(outer2D, holes2D)
	} else {
		tris = earClip(outer2D)
	}
	// The ear-clip output is consistently oriented in the projection plane; patchMeshFrom keeps that
	// winding and flips the whole patch once if it faces inward overall. Winding each triangle to its
	// own vertex normals instead flips slivers inconsistently — the back-facing hole walls in Normal-Debug.
	nrm := make([]math.Vector3, len(pos))
	for i, p := range pos {
		u, v := s.ParamAt(p)
		nrm[i] = s.NormalAt(u, v)
	}
	m := patchMeshFrom(pos, nrm, tris)
	repairFolds(m, 8) // a curved cap's boundary triangulation can crease; flip the folding diagonals (#585)
	return m
}

// patchProjection picks the 2D embedding to ear-clip a curved patch's boundary in. A B-spline
// patch is flattened in its OWN (u,v) parameter space, where the trim loops are a simple polygon;
// the best-fit PLANE projection (used for the analytic surfaces, whose (u,v) can be degenerate at
// a pole/seam) self-intersects for a large freeform patch and makes ear-clipping bail partway,
// tearing the surface (the jagged duct lips). Lifting uses the exact 3D boundary points either way.
func patchProjection(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) ([]math.Point2, [][]math.Point2) {
	if _, isSpline := s.(geom.BSplineSurface); !isSpline {
		return projectToPlane(outer3D, holes3D)
	}
	uv := func(pts []math.Point3) []math.Point2 {
		out := make([]math.Point2, len(pts))
		for i, p := range pts {
			u, v := s.ParamAt(p)
			out[i] = math.P2(math.Scalar(u), math.Scalar(v))
		}
		return out
	}
	holes2D := make([][]math.Point2, len(holes3D))
	for i, h := range holes3D {
		holes2D[i] = uv(h)
	}
	return uv(outer3D), holes2D
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

// windingOpposesNormals reports whether triangle abc's geometric (cross-product) normal opposes
// the triangle's per-vertex shading normals (their sum). Used to wind a patch consistently with
// the normals each vertex actually carries — robust on a curved patch where the flat triangle's
// centroid is off the surface (unlike triangleFlipped's centroid sample).
func windingOpposesNormals(a, b, c math.Point3, na, nb, nc math.Vector3) bool {
	gn := a.VectorTo(b).Cross(a.VectorTo(c))
	return gn.Dot(na.Add(nb).Add(nc)) < 0
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

// periodicBandGrid handles a full-period curved band — a complete cylinder/cone side whose
// loop runs up the seam, around a full circle, down the seam and back, so toUVLoops can't
// unwrap it (the loop spans the whole 2π). Its trim is simply the entire period in the
// periodic parameter and the boundary's own range in the other. The grid reuses the
// boundary's parameter samples (the shared circle-edge discretization) so the band stays
// watertight with its caps, and closes the period so the last cell wraps back to the seam.
// ok=false when it isn't a single-periodic band or carries holes (those need a constrained
// triangulation — still the full-domain fallback).
func periodicBandGrid(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) (us, vs []float64, ok bool) {
	uPer, vPer := isPeriodic(s.UDomain()), isPeriodic(s.VDomain())
	if uPer == vPer || len(holes3D) != 0 {
		return nil, nil, false // need exactly one periodic direction and no holes
	}
	uu := make([]float64, len(outer3D))
	vv := make([]float64, len(outer3D))
	for i, p := range outer3D {
		uu[i], vv[i] = s.ParamAt(p)
	}
	if uPer {
		us, vs = bracketPeriod(uu), sortUnique(vv)
	} else {
		us, vs = sortUnique(uu), bracketPeriod(vv)
	}
	if len(us) < 2 || len(vs) < 2 {
		return nil, nil, false // degenerate (e.g. a cone closing to its apex — one rim circle)
	}
	// A genuine band's boundary is two full-period edge circles, so the NON-periodic direction has
	// exactly two distinct values. More than two means the boundary wanders in that direction (a
	// sphere cap straddling the pole, not a latitude band): gridding its full us×vs bounding box
	// would cover non-trim area and tear at the pole, so reject it for the boundary triangulator.
	if (uPer && len(vs) != 2) || (vPer && len(us) != 2) {
		return nil, nil, false
	}
	return us, vs, true
}

// doublyPeriodicBandGrid meshes a band on a DOUBLY-periodic surface (a torus) whose boundary wraps
// exactly ONE parameter direction around its full period — two edge circles joined by a seam — and is
// bounded in the other. The torus rim fillet is such a band: it wraps the axis (u over [0,2π]) and
// spans only a quarter of the tube (v). periodicBandGrid rejects it (a torus is periodic in BOTH
// directions), so the wrap direction is read from the boundary's own parameter span here instead.
func doublyPeriodicBandGrid(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) (us, vs []float64, ok bool) {
	if !isPeriodic(s.UDomain()) || !isPeriodic(s.VDomain()) || len(holes3D) != 0 {
		return nil, nil, false // need a doubly-periodic surface and no holes
	}
	uu := make([]float64, len(outer3D))
	vv := make([]float64, len(outer3D))
	for i, p := range outer3D {
		uu[i], vv[i] = s.ParamAt(p)
	}
	uWrap, vWrap := spansFullPeriod(uu), spansFullPeriod(vv)
	if uWrap == vWrap {
		return nil, nil, false // need exactly one direction wrapping the full period
	}
	if uWrap {
		us, vs = bracketPeriod(uu), sortUnique(vv)
	} else {
		us, vs = sortUnique(uu), bracketPeriod(vv)
	}
	if len(us) < 2 || len(vs) < 2 {
		return nil, nil, false
	}
	return us, vs, true
}

// spansFullPeriod reports whether a periodic parameter's samples cover essentially its whole [0,2π]
// period (a band that wraps the seam), versus a bounded sub-range (the tube of a rim-fillet torus).
func spansFullPeriod(g []float64) bool {
	u := sortUnique(g)
	return len(u) > 1 && u[len(u)-1]-u[0] > 2*stdmath.Pi-0.5
}

// coneApexFan tessellates a cone face that closes to its apex (a drill point): its single rim
// circle (outer3D) fans to the apex pole, which is not a topology vertex but the surface's
// geometric tip. Each triangle is wound to agree with the cone normal (a reversed face then
// flips it). ok=false unless the surface is a cone whose boundary is a SINGLE circle at one
// axial level (a frustum, with two rim circles, spans v and is handled as a band instead).
//
//nolint:funlen // builds a triangle fan to the cone apex vertex-by-vertex; length is the tessellation, not logic.
func coneApexFan(s geom.Surface, outer3D []math.Point3) (*Mesh, bool) {
	cone, ok := s.(geom.Cone)
	if !ok || len(outer3D) < 3 {
		return nil, false
	}
	vMin, vMax := stdmath.Inf(1), stdmath.Inf(-1)
	for _, p := range outer3D {
		_, v := cone.ParamAt(p)
		vMin, vMax = stdmath.Min(vMin, v), stdmath.Max(vMax, v)
	}
	if vMax-vMin > trimBorderTol {
		return nil, false // a frustum (two rim circles), not an apex cap
	}
	m := &Mesh{}
	apex := m.addVertex(cone.Apex, cone.AxisDir.AsVector().Scale(-1)) // axial normal at the pole
	rim := make([]int, len(outer3D))
	for i, p := range outer3D {
		u, v := cone.ParamAt(p)
		rim[i] = m.addVertex(p, cone.NormalAt(u, v))
	}
	for i := range outer3D {
		b, c := rim[i], rim[(i+1)%len(outer3D)]
		if triangleFlipped(cone, cone.Apex, m.Positions[b], m.Positions[c]) {
			b, c = c, b
		}
		m.addTriangle(apex, b, c)
	}
	return m, true
}

// bracketPeriod normalises a periodic direction's samples to span a CLOSED [0, 2π]: a seam sample whose
// angle wrapped to just under 2π (a seam vertex at angle 0 read back as 2π−ε from a tiny negative
// coordinate) is snapped to 0 — it is the 0 column — and an explicit 2π closer is appended. So the
// first and last columns are the shared seam and EVERY segment, including the one across the seam, is a
// grid cell. Without this the seam column landed only at ~2π, dropping the [0, first-sample] cell and
// leaving a one-cell crack against the caps. closedDomainMesh then wraps [0, 2π] onto one seam column.
func bracketPeriod(g []float64) []float64 {
	out := make([]float64, 0, len(g)+1)
	for _, x := range g {
		if x > 2*stdmath.Pi-trimBorderTol {
			x = 0 // a seam sample read back as ~2π is the 0 column
		}
		out = append(out, x)
	}
	out = sortUnique(out)
	return append(out, 2*stdmath.Pi)
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
	us := adaptiveParams(func(u float64) math.Point3 { return s.PointAt(u, (vLo+vHi)/2) }, uLo, uHi, q.tol(), q.angleTol())
	vs := adaptiveParams(func(v float64) math.Point3 { return s.PointAt((uLo+uHi)/2, v) }, vLo, vHi, q.tol(), q.angleTol())
	return closedDomainMesh(s, us, vs) // watertight on a closed surface (periodic seam + poles); else == gridMesh
}
