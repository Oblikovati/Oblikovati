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
// no special shape — so the caller falls through to toUVLoops. It selects ONE builder, the kind
// classifyCurvedTrim named, and hands it that classification's own recognition rather than making it
// read the face again; if the builder declines on its own conditioning the face demotes to the generic
// path rather than to a second special case.
// refused, when it declines, names WHY if a mesher recognised the face and gave it up on its own
// conditioning — so the reporter at the end of the router says more than "nothing recognised it".
func specialCurvedMesh(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) (m *Mesh, ok bool, refused string) {
	t := classifyCurvedTrim(f, s, outer3D, holes3D, q)
	switch t.kind {
	case kindConeApexFan:
		return apexFan(t.cone), true, ""
	case kindSphereCapFan:
		return buildSphereCap(t.cap.sph, t.cap.rim, t.cap.axis, q), true, ""
	case kindSphereZoneBand:
		return SphereZoneBandFan(t.belt, q), true, ""
	case kindSpherePatch:
		return withNoRefusal(SpherePatchMesh(t.patch, outer3D, holes3D, q))
	case kindRuledBandLoft:
		return withNoRefusal(saddleBandLoftMesh(f, s, q))
	case kindSpiricBand:
		return spiricBandMesh(f, t.tube, q)
	case kindTwoRimHoledBand:
		return withNoRefusal(twoRimHoledBandMesh(f.Chart(), s, outer3D, t.holed, q))
	case kindWedgeBand:
		return wedgeBandLoftMesh(t.wedge), true, ""
	default:
		// kindChart meshes the region the face itself records; kindUncharted records none, so the
		// same call declines and the face falls through to the generic (u,v) trim path.
		return withNoRefusal(chartFaceMesh(f, s, q))
	}
}

// withNoRefusal adapts a mesher that cannot refuse on conditioning to the arm's three-value answer: it
// either builds the face or was never the right mesher for it, and neither is a shape it gave up on.
func withNoRefusal(m *Mesh, ok bool) (*Mesh, bool, string) { return m, ok, "" }

// apexFan builds the apex→rim triangle fan for a cone (rim in path order, apex excluded), each
// triangle wound to agree with the cone's outward normal — a reversed face then flips it. A CLOSED rim
// (a conic cap) gets the wrap-around triangle; an open one (a sector's base arc) does not, so the
// fan spans only the real sector and its free boundary is the base arc plus the two meridian rulings.
// The apex is not a topology vertex but the surface's geometric tip, and it carries the axial normal.
func apexFan(c coneApexTrim) *Mesh {
	cone, rim := c.cone, c.rim
	m := &Mesh{}
	apex := m.AddVertex(cone.Apex, cone.AxisDir.AsVector().Scale(-1)) // axial normal at the pole
	idx := make([]int, len(rim))
	for i, p := range rim {
		u, v := cone.ParamAt(p)
		idx[i] = m.AddVertex(p, cone.NormalAt(u, v))
	}
	for i := range fanSpanCount(len(rim), c.closed) {
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
