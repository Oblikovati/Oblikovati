// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Trimmed curved-face tessellation — the SPECIAL-CASE curved meshers (M48 #2224 split of
// tessellate_trim.go). These surface-specific meshers precede the generic (u,v) trim path because the
// generic path mis-meshes them. specialCurvedMeshers lists them in priority order; this file also carries
// the cone-apex fans (a drill point / oblique apex cut, and an apex-collapsed sector), which fan the
// developable cone from its apex for exact, orientation-independent area. The sphere/band meshers the
// table also names live in their own tessellate_*.go files.

// specialCurvedMesh tries the surface-specific meshers that precede the generic (u,v) trim path: a cone
// closing to its apex (a fan), a sphere cut by one plane (a cap fan) or by several (a gnomonic patch),
// and a periodic developable side with one full-circle rim plus a notched rim (a band loft). It returns
// (mesh, true) on the first that applies, or (nil, false) so the caller falls through to toUVLoops.
func specialCurvedMesh(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) (*Mesh, bool) {
	// Each entry is one surface-specific mesher, tried in order; the WHY for each shape lives beside
	// its case. Collapsed into a table (rather than a chain of ifs) so this dispatcher stays under
	// the function-length budget as the special-case roster grows — see tessellate_wedge_band.go's
	// entry for the newest one (A1/D4's oblique wedge band).
	for _, try := range specialCurvedMeshers(f, s, outer3D, holes3D, q) {
		if m, ok := try(); ok {
			return m, true
		}
	}
	return nil, false
}

// specialCurvedMeshers lists the surface-specific meshers specialCurvedMesh tries, in priority order.
func specialCurvedMeshers(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) []func() (*Mesh, bool) {
	return []func() (*Mesh, bool){
		// a cone apex CAP (a drill point / oblique apex cut) or an apex-collapsed SECTOR (a partial
		// sweep): both fan the developable cone from its apex for exact, orientation-independent area
		// — see coneApexMesh. A holed cone face is never an apex topology (its inner rim is the hole).
		func() (*Mesh, bool) { return coneApexMesh(f, s, outer3D, holes3D) },
		// a sphere cut by one plane (a cap): rings from the rim to the enclosed pole
		func() (*Mesh, bool) { return sphereCapFan(s, outer3D, holes3D, q) },
		// a large sphere zone reaching an enclosed pole (one off-centre plane cut, seam to the pole):
		// fan on the rim CIRCLE's exact normal, not the seam-biased newellUnit (J2)
		func() (*Mesh, bool) { return sphereZoneCapFan(f, s, q) },
		// a seamed cap: coplanar MULTI-ARC rim (a subdivided boss equator) + one doubled seam edge to
		// the pole (S6/S7's hemisphere): latitude-ring fan, not the density-capped stereo CDT
		func() (*Mesh, bool) { return sphereSeamedCapFan(f, s, q) },
		// a BELT between two coaxial full-circle rims (a ball an axle passes through, a revolved
		// meridian arc off the axis at both ends): latitude rings about the RIMS' axis. It must precede
		// the gnomonic patch, whose chart covers less than a hemisphere and so cannot hold a belt that
		// straddles its own equator (#2061)
		func() (*Mesh, bool) { return sphereZoneBandFan(f, s, q) },
		// a sphere bounded by several arcs (a box cut): gnomonic CDT, pole/seam-safe
		func() (*Mesh, bool) { return spherePatchMesh(f, s, outer3D, holes3D, q) },
		// a periodic side with one full-circle rim and one notched rim (a frustum flat that fades): loft
		func() (*Mesh, bool) { return notchedRimBandMesh(f, s, q) },
		// a developable side with two CLOSED full-wrap rims (e.g. a circle + an oblique-cut ellipse): loft
		func() (*Mesh, bool) { return twoClosedRimBandMesh(f, s, q) },
		// a torus cut through the hole: a tube-wrapping band between two spiric ovals (#1375)
		func() (*Mesh, bool) { return spiricBandMesh(f, s, q) },
		// a two-rim band (a notched rim + an intact rim) carrying lens holes: bridge the seam, unroll
		func() (*Mesh, bool) { return twoRimHoledBandMesh(s, outer3D, holes3D, q) },
		// an OPEN oblique-ended cylinder wedge band (pyramid slant fillet, A1/D4): one exact zipped
		// strip between the two end chains — the generic CDT's flat end slivers double-cover the
		// neighbour plane and collide on its ear diagonals (deg-4 mesh edges), see tessellate_wedge_band.go
		func() (*Mesh, bool) { return wedgeBandLoftMesh(f, s, q) },
	}
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
