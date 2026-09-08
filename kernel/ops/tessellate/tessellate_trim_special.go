// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Trimmed curved-face tessellation — the SURFACE-SPECIFIC curved meshers (M48 #2224 split of
// tessellate_trim.go). These meshers precede the generic (u,v) trim path because the generic path
// mis-meshes their shapes. Which one a face gets is a CLASSIFICATION, not an ordered try-list: the
// switch below reads the one kind classifyCurvedTrim names (curved_trim_classify.go) and selects
// exactly one mesher. This file also carries the cone-apex fan, which fans the developable cone from
// its apex for exact, orientation-independent area; the sphere and band meshers the switch names live
// in their own tessellate_*.go files.

// specialCurvedMesh meshes a curved face whose trim is one of the surface-specific shapes the generic
// (u,v) path mis-meshes, or a face that carries its own parametric trim (kindChart, meshed from the
// region ADR-0063 records). It returns (nil,false) only for kindUncharted — a face with no chart and
// no special shape — so the caller falls through to toUVLoops. It selects ONE mesher, the kind
// classifyCurvedTrim names, and if that mesher declines on its own conditioning the face demotes to
// the generic path rather than to a second special case.
func specialCurvedMesh(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) (*Mesh, bool) {
	switch classifyCurvedTrim(f, s, outer3D, holes3D, q) {
	case kindConeApexFan:
		return coneApexMesh(f, s, outer3D, holes3D)
	case kindSphereCapFan:
		return sphereCapFanMesh(f, s, outer3D, holes3D, q)
	case kindSphereZoneBand:
		return SphereZoneBandFan(f, s, q)
	case kindSpherePatch:
		return SpherePatchMesh(f, s, outer3D, holes3D, q)
	case kindRuledBandLoft:
		return saddleBandLoftMesh(f, s, q)
	case kindSpiricBand:
		return spiricBandMesh(f, s, q)
	case kindTwoRimHoledBand:
		return twoRimHoledBandMesh(f.Chart(), s, outer3D, holes3D, q)
	case kindWedgeBand:
		return wedgeBandLoftMesh(f, s, q)
	default:
		// kindChart meshes the region the face itself records; kindUncharted records none, so the
		// same call declines and the face falls through to the generic (u,v) trim path.
		return chartFaceMesh(f, s, q)
	}
}

// coneApexMesh meshes a cone face that closes to its apex — a closed conic apex CAP (a drill point or
// an oblique apex cut) or an apex-collapsed SECTOR (a partial angular sweep). Both exploit that a cone
// is developable, so a triangle fan from the apex gives exact area, orientation-independently.
func coneApexMesh(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) (*Mesh, bool) {
	cone, rim, closed, ok := coneApexFanRim(f, s, outer3D, holes3D)
	if !ok {
		return nil, false
	}
	return apexFan(cone, rim, closed), true
}

// coneApexFanRim returns the cone, the rim the apex fan sweeps and whether that rim CLOSES (a cap's
// full conic rim wraps; a sector's base arc does not). ok=false for every cone face that is not an
// apex topology — a frustum band, a saddle-bounded stub, a holed face whose inner rim is a hole, or a
// seamed apex face whose loop already spans the apex.
//
// Example: a drill point's single circular rim → (rim = the circle, closed = true).
func coneApexFanRim(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) (geom.Cone, []math.Point3, bool, bool) {
	cone, isCone := s.(geom.Cone)
	if !isCone || len(holes3D) != 0 {
		return cone, nil, false, false
	}
	if faceIsConeApexCap(f) {
		return cone, outer3D, true, len(outer3D) >= 3
	}
	if len(f.Loops()) != 1 {
		return cone, nil, false, false
	}
	rim := rimExcludingApex(outer3D, cone.Apex, geom.ResolutionForPoints(outer3D).Weld())
	// No apex vertex on the loop (a frustum/stub), or too few rim points for a fan.
	return cone, rim, false, len(rim) != len(outer3D) && len(rim) >= 2
}

// apexFan builds the apex→rim triangle fan for a cone (rim in path order, apex excluded), each
// triangle wound to agree with the cone's outward normal — a reversed face then flips it. A CLOSED rim
// (a conic cap) gets the wrap-around triangle; an open one (a sector's base arc) does not, so the
// fan spans only the real sector and its free boundary is the base arc plus the two meridian rulings.
// The apex is not a topology vertex but the surface's geometric tip, and it carries the axial normal.
func apexFan(cone geom.Cone, rim []math.Point3, closed bool) *Mesh {
	m := &Mesh{}
	apex := m.AddVertex(cone.Apex, cone.AxisDir.AsVector().Scale(-1)) // axial normal at the pole
	idx := make([]int, len(rim))
	for i, p := range rim {
		u, v := cone.ParamAt(p)
		idx[i] = m.AddVertex(p, cone.NormalAt(u, v))
	}
	for i := range fanSpanCount(len(rim), closed) {
		b, c := idx[i], idx[(i+1)%len(rim)]
		if triangleFlipped(cone, cone.Apex, m.Positions[b], m.Positions[c]) {
			b, c = c, b
		}
		m.AddTriangle(apex, b, c)
	}
	return m
}

// fanSpanCount is how many rim spans a fan covers: every span of a closed rim, one fewer for an open
// one, whose last point has no successor to pair with.
func fanSpanCount(n int, closed bool) int {
	if closed {
		return n
	}
	return n - 1
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

// faceIsConeApexCap reports whether a cone face is a SEAM-FREE apex cap — a single loop that is one closed
// conic rim (a circle or an oblique-cut ellipse) encircling the axis, so an apex fan tiles it cleanly. This
// is the brep drill point and the oblique apex cut (Oblikovati#1375). A SEAMED apex face (an imported cone
// whose loop carries seam rulings down to the apex) is excluded: its boundary already spans the apex, so it
// is meshed by the seamed closed-domain mesher, not the fan. A frustum band or a crossing-cone stub
// (bounded by saddle curves) also fails the single-closed-conic test and is not fanned to a far-off apex.
// The caller has already established the surface is a cone; what is left is a purely topological test.
func faceIsConeApexCap(f *topo.Face) bool {
	if len(f.Loops()) != 1 {
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
