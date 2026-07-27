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

// seamAngularTol snaps periodic parameter samples at the 0/2π seam. It compares RADIANS,
// so it is scale-free by construction; the metric (non-periodic) grid direction derives
// its tolerance from its own span instead (gridTol) so µm and metre parts grid
// identically (#1610 — the old shared 1e-6 was absolute in the metric direction).
const seamAngularTol = 1e-6 // tol:angular (radians; periodic-seam snap)

// trimGridRelTol is the dimensionless fraction of a grid axis' span within which two
// parameter samples count as the same grid line.
const trimGridRelTol = 1e-6 // tol:numeric (dimensionless fraction of the axis span)

// gridTol is the same-grid-line tolerance for one axis' samples: a fixed fraction of the
// samples' span (zero for a degenerate axis — near() is inclusive, so exact duplicates
// still collapse).
func gridTol(xs []float64) float64 {
	lo, hi := minMax(xs)
	return trimGridRelTol * (hi - lo)
}

// tessellateCurvedFace meshes a curved face's trimmed region (see file doc).
func tessellateCurvedFace(f *topo.Face, q Quality) *Mesh {
	s := f.Geometry()
	if m := splineFaceMesh(f, s, q); m != nil {
		return m // M25: a B-spline face via the metric-aware (u,v) triangulation
	}
	outer3D := faceOuterBoundary(f, q)
	holes3D := faceHoleBoundaries(f, q)
	if t, ok := s.(geom.Torus); ok && len(outer3D) < 3 && len(holes3D) > 0 {
		// A torus face with hole loops but NO outer loop wraps the whole closed surface minus the holes —
		// the genus-1 COMPLEMENT of an oval cap (a torus-minus-disk). The full-domain grid would ignore the
		// hole; torusComplementMesh charts the torus minus the oval window instead (Oblikovati#1375).
		return torusComplementMesh(t, holes3D, q)
	}
	if len(outer3D) < 3 {
		return fullDomainGridMesh(s, q)
	}
	if m, special := specialCurvedMesh(f, s, outer3D, holes3D, q); special {
		return m // a cone-apex/sphere fan or cap, sphere box-cut patch, or notched-rim band
	}
	outerUV, holesUV, ok := toUVLoops(s, outer3D, holes3D)
	if !ok {
		return meshSeamCrossingFace(f, s, outer3D, holes3D, q) // a loop wrapping the seam: band/cap fallbacks
	}
	if us, vs, isRect := isoRectangleGrid(outerUV); len(holesUV) == 0 && isRect {
		return structuredGridMesh(s, us, vs) // cylinder/cone wall, fillet face: exact area
	}
	if us, vs, skip, isCells := isoRectilinearGrid(outerUV); len(holesUV) == 0 && isCells {
		// A band the obstacle imprint notched (fillet_band_imprint.go): still bounded entirely by
		// iso-lines, so it is a union of grid cells and needs no triangulator — see
		// tessellate_rectilinear.go for what the generic CDT does with it instead.
		return structuredGridMeshSkip(s, us, vs, skip)
	}
	return nonRectangularMesh(s, q, outer3D, holes3D, outerUV, holesUV)
}

// specialCurvedMesh tries the surface-specific meshers that precede the generic (u,v) trim path: a cone
// closing to its apex (a fan), a sphere cut by one plane (a cap fan) or by several (a gnomonic patch),
// and a periodic developable side with one full-circle rim plus a notched rim (a band loft). It returns
// (mesh, true) on the first that applies, or (nil, false) so the caller falls through to toUVLoops.
func specialCurvedMesh(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) (*Mesh, bool) {
	if m, isApex := coneApexMesh(f, s, outer3D, holes3D); isApex {
		return m, true // a cone apex CAP (a drill point / oblique apex cut) or an apex-collapsed SECTOR (a
		// partial sweep): both fan the developable cone from its apex for exact, orientation-independent area
		// — see coneApexMesh. A holed cone face is never an apex topology (its inner rim is the hole).
	}
	if m, isCap := sphereCapFan(s, outer3D, q); isCap {
		return m, true // a sphere cut by one plane (a cap): rings from the rim to the enclosed pole
	}
	if m, isZone := sphereZoneCapFan(f, s, q); isZone {
		return m, true // a large sphere zone reaching an enclosed pole (one off-centre plane cut, seam to
		// the pole): fan on the rim CIRCLE's exact normal, not the seam-biased newellUnit (J2)
	}
	if m, isPatch := spherePatchMesh(s, outer3D, holes3D, q); isPatch {
		return m, true // a sphere bounded by several arcs (a box cut): gnomonic CDT, pole/seam-safe
	}
	if m, isBand := notchedRimBandMesh(f, s, q); isBand {
		return m, true // a periodic side with one full-circle rim and one notched rim (a frustum flat that fades): loft
	}
	if m, isBand := twoClosedRimBandMesh(f, s, q); isBand {
		return m, true // a developable side with two CLOSED full-wrap rims (e.g. a circle + an oblique-cut ellipse): loft
	}
	if m, isBand := spiricBandMesh(f, s, q); isBand {
		return m, true // a torus cut through the hole: a tube-wrapping band between two spiric ovals (#1375)
	}
	if m, isBand := twoRimHoledBandMesh(s, outer3D, holes3D, q); isBand {
		return m, true // a two-rim band (a notched rim + an intact rim) carrying lens holes: bridge the seam, unroll
	}
	return nil, false
}

// splineFaceMesh meshes a B-spline face through the metric-aware (u,v) triangulation (M25), or nil when
// the face is not a B-spline (so the caller falls through to the analytic-surface paths).
func splineFaceMesh(f *topo.Face, s geom.Surface, q Quality) *Mesh {
	if _, isSpline := s.(geom.BSplineSurface); !isSpline {
		return nil
	}
	// The CLOSED elliptic-rim canal band (two closed rails + a seam used twice) has no usable planar
	// (u,v) trim, so it is lofted rail-to-rail instead. Gated on that exact loop shape, which no open
	// canal arm or corner patch has.
	if m, ok := canalRimBandMesh(f, s, q); ok {
		return m
	}
	// A closed-in-u (periodic) B-spline face whose trim straddles the seam tangles the planar seam-cut
	// loop; the covering-space periodic CDT un-seams it. It defers (nil,false) for the ordinary open patch.
	if m, ok := periodicNurbsFaceMesh(f, q); ok {
		return m
	}
	return nurbsPcurveMesh(f, q)
}

// meshSeamCrossingFace meshes a curved face whose boundary loop wraps the periodic seam (so toUVLoops
// can't unwrap it): a full cylinder/cone side or a torus rim-fillet band closes the seam watertight via
// closedDomainMesh; a singly-periodic sphere cap straddling the pole goes through the best-fit-plane CDT
// (the full-domain grid tears there); a doubly-periodic torus we can't reduce keeps the full-domain grid.
func meshSeamCrossingFace(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) *Mesh {
	if us, vs, isBand := periodicBandGrid(s, outer3D, holes3D); isBand {
		return closedDomainMesh(s, us, vs) // full cylinder/cone side with circular rims: grid the period
	}
	if m, ok := holedConicWallMesh(s, outer3D, holes3D, q); ok {
		return m // a drilled cylinder/cone wall: full-period side with lens holes — unroll + metric CDT
	}
	if m, ok := saddleBandLoftMesh(f, s, q); ok {
		return m // a cylinder/cone band with non-circular (saddle) rims — a crossing cylinder: loft v(u)
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
	case geom.Torus, geom.Sphere, geom.Cylinder, geom.EllipticalCylinder, geom.Cone, geom.BSplineSurface:
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
	// Ear clipping is only guaranteed on a simple polygon; a degenerate/self-touching trim makes it
	// break early (a hole) or emit count-complete but OVERLAPPING triangles — the coverage gate
	// detects both in the projection plane and retriangulates through the CDT, whose
	// split-at-vertex recovery handles the self-touching case exactly (#1605; recovery tier = #1604).
	tris, accepted := patchCoverageGate(outer2D, holes2D, tris)
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
	diagnosePatchCoverage(m, accepted)
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
	tolU := trimGridRelTol * float64(uMax-uMin)
	tolV := trimGridRelTol * float64(vMax-vMin)
	var bottomU, topU, leftV, rightV []float64
	for _, p := range loop {
		onB, onT := near(p.Y, vMin, tolV), near(p.Y, vMax, tolV)
		onL, onR := near(p.X, uMin, tolU), near(p.X, uMax, tolU)
		if !onB && !onT && !onL && !onR {
			return nil, nil, false // a vertex off the bbox border — not a rectangle
		}
		appendIf(&bottomU, p.X, onB)
		appendIf(&topU, p.X, onT)
		appendIf(&leftV, p.Y, onL)
		appendIf(&rightV, p.Y, onR)
	}
	bottomU, topU = sortUnique(bottomU, tolU), sortUnique(topU, tolU)
	leftV, rightV = sortUnique(leftV, tolV), sortUnique(rightV, tolV)
	if !sameGrid(bottomU, topU, tolU) || !sameGrid(leftV, rightV, tolV) {
		return nil, nil, false // opposite edges sample differently → would leave T-junctions
	}
	return bottomU, leftV, true
}

// structuredGridMesh tessellates the surface over the us×vs parameter grid as thin iso
// quads (two triangles each), wound outward with true per-vertex normals. Border grid
// points reproduce the exact boundary vertices (ParamAt is the inverse of PointAt), so the
// mesh conforms to the shared edge discretization.
func structuredGridMesh(s geom.Surface, us, vs []float64) *Mesh {
	return structuredGridMeshSkip(s, us, vs, nil)
}

// structuredGridMeshSkip is [structuredGridMesh] with an optional per-cell skip predicate (omit a cell
// when skip(i,j) is true) — the torus complement grid omits the window cells the local patch fills.
func structuredGridMeshSkip(s geom.Surface, us, vs []float64, skip func(i, j int) bool) *Mesh {
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
			if skip != nil && skip(i, j) {
				continue
			}
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
		us, vs = bracketPeriod(uu), sortUnique(vv, gridTol(vv))
	} else {
		us, vs = sortUnique(uu, gridTol(uu)), bracketPeriod(vv)
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
		us, vs = bracketPeriod(uu), sortUnique(vv, gridTol(vv))
	} else {
		us, vs = sortUnique(uu, gridTol(uu)), bracketPeriod(vv)
	}
	if len(us) < 2 || len(vs) < 2 {
		return nil, nil, false
	}
	return us, vs, true
}

// spansFullPeriod reports whether a periodic parameter's samples cover essentially its whole [0,2π]
// period (a band that wraps the seam), versus a bounded sub-range (the tube of a rim-fillet torus, or a
// perpendicular torus cut's kept arc). It measures the OCCUPIED span as 2π minus the single largest gap —
// so a sub-arc that merely STRADDLES the 0/2π seam (min near 0, max near 2π, but a wide gap in between) is
// correctly seen as bounded, not full (Oblikovati#1375).
func spansFullPeriod(g []float64) bool {
	s := sortUnique(g, seamAngularTol)
	if len(s) < 2 {
		return false
	}
	maxGap := s[0] + 2*stdmath.Pi - s[len(s)-1] // the wrap gap (last sample → first, across the seam)
	for i := 0; i+1 < len(s); i++ {
		if gap := s[i+1] - s[i]; gap > maxGap {
			maxGap = gap
		}
	}
	return 2*stdmath.Pi-maxGap > 2*stdmath.Pi-0.5 // occupied span (all but the largest gap) is nearly full
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

// coneApexMesh handles the two seam-free cone-apex topologies that precede the generic (u,v) trim
// path: a closed conic apex CAP (a drill point or an oblique apex cut — coneApexFan) and an
// apex-collapsed SECTOR (a partial angular sweep — coneApexSectorMesh). Both exploit that a cone is
// developable, so a triangle fan from the apex gives exact area. A holed cone face is never an apex
// topology (its inner rim is the hole, not a fan); returns (nil,false) so the caller falls through.
func coneApexMesh(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) (*Mesh, bool) {
	if _, isCone := s.(geom.Cone); !isCone || len(holes3D) != 0 {
		return nil, false
	}
	if faceIsConeApexCap(f, s) {
		if m, isFan := coneApexFan(s, outer3D); isFan {
			return m, true // a stub (saddle-bounded) is rejected by faceIsConeApexCap, not fanned to a far apex
		}
	}
	return coneApexSectorMesh(f, s, outer3D)
}

// coneApexSectorMesh meshes an apex-collapsed cone SECTOR — a partial angular sweep of a cone whose
// boundary is one base arc plus two meridian rulings meeting at the apex (NOT a full closed conic cap,
// which faceIsConeApexCap/coneApexFan handle). Because a cone is developable, a triangle fan from the
// apex to the base-arc discretization reproduces the sector's EXACT area (OCCT H6 270° sector: 133286),
// orientation-independently. The generic (u,v) trim path mis-meshes it: the apex row collapses u, so
// ParamAt's degenerate apex angle spuriously reads a 270° sector's loop as seam-crossing on ONE
// orientation (top vs bottom cone), routing it to the full-period band grid (closedDomainMesh) which
// over-covers the partial sweep as if it were a full 2π cone (H6 top cone 167927, ×1.26 — the tape
// measure). Returns (nil,false) for any cone face that is NOT an apex-reaching sector (a frustum band,
// a saddle-bounded stub, or a closed conic cap), so every other cone face keeps its existing path.
func coneApexSectorMesh(f *topo.Face, s geom.Surface, outer3D []math.Point3) (*Mesh, bool) {
	cone, ok := s.(geom.Cone)
	if !ok || len(f.Loops()) != 1 || faceIsConeApexCap(f, s) {
		return nil, false
	}
	rim := rimExcludingApex(outer3D, cone.Apex, ResolutionForPoints(outer3D).Weld())
	if len(rim) == len(outer3D) || len(rim) < 2 {
		return nil, false // no apex vertex on the loop (a frustum/stub), or too few rim points for a fan
	}
	return coneSectorFan(cone, rim), true
}

// rimExcludingApex returns the boundary points with the apex vertex removed, re-ordered to start
// immediately AFTER the apex so the remaining points read as the open base-arc path (one meridian
// base → base arc → the other meridian base) rather than a loop closing across the sector's void.
// tol is the model-relative coincidence scale (ResolutionForPoints.Weld). Empty when no apex is found.
func rimExcludingApex(loop []math.Point3, apex math.Point3, tol float64) []math.Point3 {
	k := -1
	for i, p := range loop {
		if p.DistanceTo(apex) <= tol {
			k = i
			break
		}
	}
	if k < 0 {
		return nil
	}
	rim := make([]math.Point3, 0, len(loop))
	for off := 1; off <= len(loop); off++ {
		if p := loop[(k+off)%len(loop)]; p.DistanceTo(apex) > tol {
			rim = append(rim, p)
		}
	}
	return rim
}

// coneSectorFan builds the apex→rim triangle fan for a cone sector (rim in base-arc order, apex
// excluded), each triangle wound to agree with the cone's outward normal. No wrap-around triangle is
// emitted, so the fan spans only the real sector — its free boundary is the base arc plus the two
// meridian rulings, watertight with the sector's neighbour cap and ring faces (shared discretization).
func coneSectorFan(cone geom.Cone, rim []math.Point3) *Mesh {
	m := &Mesh{}
	apex := m.addVertex(cone.Apex, cone.AxisDir.AsVector().Scale(-1)) // axial normal at the pole
	idx := make([]int, len(rim))
	for i, p := range rim {
		u, v := cone.ParamAt(p)
		idx[i] = m.addVertex(p, cone.NormalAt(u, v))
	}
	for i := 0; i+1 < len(rim); i++ {
		b, c := idx[i], idx[i+1]
		if triangleFlipped(cone, cone.Apex, rim[i], rim[i+1]) {
			b, c = c, b
		}
		m.addTriangle(apex, b, c)
	}
	return m
}

// faceIsConeApexCap reports whether a cone face is a SEAM-FREE apex cap — a single loop that is one closed
// conic rim (a circle or an oblique-cut ellipse) encircling the axis, so an apex fan tiles it cleanly. This
// is the brep drill point and the oblique apex cut (Oblikovati#1375). A SEAMED apex face (an imported cone
// whose loop carries seam rulings down to the apex) is excluded: its boundary already spans the apex, so it
// is meshed by the seamed closed-domain mesher, not the fan. A frustum band or a crossing-cone stub
// (bounded by saddle curves) also fails the single-closed-conic test and is not fanned to a far-off apex.
func faceIsConeApexCap(f *topo.Face, s geom.Surface) bool {
	if _, ok := s.(geom.Cone); !ok || len(f.Loops()) != 1 {
		return false
	}
	uses := f.Loops()[0].EdgeUses()
	if len(uses) != 1 {
		return false
	}
	e := uses[0].Edge()
	switch e.Geometry().(type) {
	case geom.Circle, geom.EllipseFull, geom.EllipticalArc:
		return e.StartVertex() == e.EndVertex() // a single closed conic rim around the axis
	}
	return false
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
		if x > 2*stdmath.Pi-seamAngularTol {
			x = 0 // a seam sample read back as ~2π is the 0 column
		}
		out = append(out, x)
	}
	out = sortUnique(out, seamAngularTol)
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
	return out, hi-lo < 2*stdmath.Pi-1e-6 // tol:angular (radians)
}

// isPeriodic reports whether a [0, 2π] parameter domain wraps.
func isPeriodic(lo, hi float64) bool {
	return stdmath.Abs(lo) < 1e-9 && stdmath.Abs(hi-2*stdmath.Pi) < 1e-9 // tol:angular (radians)
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

// near reports whether two parameter samples coincide within tol (inclusive, so exact
// duplicates collapse even on a degenerate zero-span axis).
func near(a, b, tol float64) bool { return stdmath.Abs(a-b) <= tol }

func appendIf(dst *[]float64, x float64, cond bool) {
	if cond {
		*dst = append(*dst, x)
	}
}

// sortUnique returns the values sorted ascending with near-duplicates (within tol) collapsed.
func sortUnique(xs []float64, tol float64) []float64 {
	sort.Float64s(xs)
	out := xs[:0:0]
	for _, x := range xs {
		if len(out) == 0 || !near(out[len(out)-1], x, tol) {
			out = append(out, x)
		}
	}
	return out
}

// sameGrid reports whether two sorted grids have the same lines (within tol).
func sameGrid(a, b []float64, tol float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !near(a[i], b[i], tol) {
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
	us := unionIsoparmParams(s, uLo, uHi, vLo, vHi, true, q)
	vs := unionIsoparmParams(s, uLo, uHi, vLo, vHi, false, q)
	return closedDomainMesh(s, us, vs) // watertight on a closed surface (periodic seam + poles); else == gridMesh
}

// isoparmSampleFractions are the fixed-direction positions at which unionIsoparmParams samples the
// varying direction — the ends, quarters and middle, enough to catch a curvature feature anywhere
// across the domain that a single mid-line would miss.
var isoparmSampleFractions = []float64{0, 0.25, 0.5, 0.75, 1}

// unionIsoparmParams returns the adaptive parameter breakpoints of the VARYING direction, unioned over
// SEVERAL isoparms of the fixed direction instead of a single mid-line — so the grid resolves curvature
// wherever it occurs in the domain (a torus tube, an off-centre bump), not only where the mid-line
// happens to look (#1412). A uniformly-curved surface yields the same breakpoints on every isoparm, so
// the union equals the old single-line result and nothing is densified (no flat-face regression).
func unionIsoparmParams(s geom.Surface, uLo, uHi, vLo, vHi float64, alongU bool, q Quality) []float64 {
	lo, hi, fixedLo, fixedHi := uLo, uHi, vLo, vHi
	if !alongU {
		lo, hi, fixedLo, fixedHi = vLo, vHi, uLo, uHi
	}
	var merged []float64
	for _, frac := range isoparmSampleFractions {
		fixed := fixedLo + frac*(fixedHi-fixedLo)
		eval := func(t float64) math.Point3 {
			if alongU {
				return s.PointAt(t, fixed)
			}
			return s.PointAt(fixed, t)
		}
		merged = mergeSortedParams(merged, adaptiveParams(eval, lo, hi, q.tol(), q.angleTol()))
	}
	return merged
}

// mergeSortedParams merges two ascending parameter slices into one, dropping a value within a tiny
// fraction of the span of one already kept (so unioning many isoparms does not pile near-coincident
// grid lines that would make sliver cells).
func mergeSortedParams(a, b []float64) []float64 {
	if len(a) == 0 {
		return b
	}
	out := append([]float64(nil), a...)
	out = append(out, b...)
	sort.Float64s(out)
	span := out[len(out)-1] - out[0]
	eps := span * 1e-9 // tol:numeric (relative to the parameter span)
	deduped := out[:1]
	for _, v := range out[1:] {
		if v-deduped[len(deduped)-1] > eps {
			deduped = append(deduped, v)
		}
	}
	return deduped
}
