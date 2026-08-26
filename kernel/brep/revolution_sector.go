// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// SolidOfRevolutionSector builds a PARTIAL analytic revolve: a meridian swept through angle radians
// (0 < angle < 2π) about the axis, starting from the ref0 radial half-plane and sweeping toward
// +axis. Unlike the full SolidOfRevolution — where each off-axis vertex traces a closed CIRCLE and
// each edge a periodic wall — a sector traces an ARC per vertex and a TRIMMED surface-of-revolution
// per edge (a partial cylinder / cone / disk sector), closed off by TWO planar caps: the meridian in
// the start (angle 0) half-plane and in the end (angle) half-plane. So a partial revolve of a tube
// keeps true cylindrical walls that thread/chamfer/fillet attach to, and a projected face reads real
// arcs — not the ~48-facet-per-turn swept prism (#2019 / #2164 class).
//
// It returns (nil, nil) — a signal to keep the faceted revolve — for fewer than three vertices, a
// non-partial angle, or a meridian with a CURVED (arc) edge (a torus/sphere sector, not yet built
// here; the caller keeps the faceted revolve for those). Booleans re-facet the sector on demand
// (combine → planarized), like every analytic tool.
func SolidOfRevolutionSector(axisOrigin math.Point3, axisDir, ref0 math.Vector3, angle float64, verts []RevolveVertex, feat string) (*topo.Body, error) {
	if len(verts) < 3 || angle <= 0 || angle >= 2*stdmath.Pi {
		return nil, nil
	}
	if meridianHasArc(verts) {
		return nil, nil // curved meridian → torus/sphere sector: keep faceted for now
	}
	a, err := math.UnitVector3FromVector(axisDir)
	if err != nil {
		return nil, err
	}
	r0, err := math.UnitVector3FromVector(perpToAxis(ref0, a))
	if err != nil {
		return nil, err
	}
	verts = ccwMeridianVerts(verts) // right-hand face normals must point out of the solid
	return buildRevolutionSector(axisOrigin, a, r0, angle, verts, feat), nil
}

// meridianHasArc reports whether any meridian edge is an arc (a fillet/curved edge → torus or sphere
// of revolution). The sector builder handles straight edges only for now.
func meridianHasArc(verts []RevolveVertex) bool {
	for i := range verts {
		if verts[i].ArcCenter != nil {
			return true
		}
	}
	return false
}

// perpToAxis returns the component of ref perpendicular to the axis — the radial direction the sector
// starts from. A ref parallel to the axis yields the zero vector (the caller errors on it).
func perpToAxis(ref math.Vector3, a math.UnitVector3) math.Vector3 {
	return ref.Sub(a.AsVector().Scale(ref.Dot(a.AsVector())))
}

// sectorNode is one meridian vertex realized in 3D for a sector: the arc it traces (angle 0 → angle),
// or an apex when the vertex sits on the axis (r == 0), with the arc's two endpoints.
type sectorNode struct {
	center math.Point3
	r      float64
	arc    *topo.Edge   // vertex arc angle 0→angle; nil when r == 0 (an axis apex)
	v0, vA *topo.Vertex // arc endpoints at angle 0 and at angle (== the apex when r == 0)
}

// buildRevolutionSector assembles the sector body: one arc per off-axis vertex, one meridian line per
// edge at each of the two cap angles, one trimmed surface-of-revolution face per edge, and the two
// planar caps.
func buildRevolutionSector(axisOrigin math.Point3, a, r0 math.UnitVector3, angle float64, verts []RevolveVertex, feat string) *topo.Body {
	bld := topo.NewBuilder(true, revLin(feat, "sector-body", 0))
	res := revolveResolution(verts)
	nodes := make([]sectorNode, len(verts))
	for i, vrt := range verts {
		nodes[i] = makeSectorNode(bld, axisOrigin, a, r0, float64(vrt.P.X), float64(vrt.P.Y), angle, res.Plane(), feat, i)
	}
	m0, mA := sectorMeridianEdges(bld, nodes, feat)
	for i := range verts {
		j := (i + 1) % len(verts)
		addSectorFace(bld, nodes[i], nodes[j], a, r0, m0[i], mA[i], res.Weld(), feat, i)
	}
	addSectorCaps(bld, a, r0, angle, axisOrigin, m0, mA, feat)
	return bld.Build()
}

// makeSectorNode builds the vertex arc (and its two endpoints) for one off-axis meridian vertex, or a
// single apex vertex when the vertex lies on the axis. The arc runs angle 0 → angle in the (r0, axis)
// frame, so its endpoints seed the two cap meridians.
func makeSectorNode(bld *topo.Builder, axisOrigin math.Point3, a, r0 math.UnitVector3, r, z, angle, axisTol float64, feat string, i int) sectorNode {
	center := axisOrigin.TranslateBy(a.AsVector().Scale(math.Scalar(z)))
	if r <= axisTol {
		apex := bld.AddVertex(center, revLin(feat, "sector-apex", i))
		return sectorNode{center: center, r: 0, v0: apex, vA: apex}
	}
	arc, err := geom.NewArc3d(center, a.AsVector(), r0.AsVector(), r, 0, angle)
	if err != nil {
		apex := bld.AddVertex(center, revLin(feat, "sector-apex", i))
		return sectorNode{center: center, r: 0, v0: apex, vA: apex}
	}
	v0 := bld.AddVertex(arc.PointAt(0), revLin(feat, "sector-start", i))
	vA := bld.AddVertex(arc.PointAt(1), revLin(feat, "sector-end", i))
	return sectorNode{center: center, r: r, arc: bld.AddEdge(arc, v0, vA, revLin(feat, "sector-arc", i)), v0: v0, vA: vA}
}

// sectorMeridianEdges builds the two cap meridians: a line per edge from vertex i to vertex j at the
// start angle (m0) and at the end angle (mA). An edge whose BOTH endpoints lie on the axis is the seam
// ALONG the axis shared by the two caps, so m0 and mA are the SAME edge there (used once per cap).
func sectorMeridianEdges(bld *topo.Builder, nodes []sectorNode, feat string) (m0, mA []*topo.Edge) {
	n := len(nodes)
	m0, mA = make([]*topo.Edge, n), make([]*topo.Edge, n)
	for i := range nodes {
		j := (i + 1) % n
		m0[i] = bld.AddEdge(geom.NewLineSegment(nodes[i].v0.Point(), nodes[j].v0.Point()), nodes[i].v0, nodes[j].v0, revLin(feat, "sector-m0", i))
		if nodes[i].r == 0 && nodes[j].r == 0 { // an on-axis edge: one seam shared by both caps
			mA[i] = m0[i]
			continue
		}
		mA[i] = bld.AddEdge(geom.NewLineSegment(nodes[i].vA.Point(), nodes[j].vA.Point()), nodes[i].vA, nodes[j].vA, revLin(feat, "sector-mA", i))
	}
	return m0, mA
}

// addSectorFace adds the trimmed surface-of-revolution face for one straight meridian edge: a partial
// cylinder (axis-parallel), a disk/annulus SECTOR (axis-perpendicular), or a partial cone (oblique).
// An edge lying on the axis traces no surface (the two caps meet along it) and is skipped.
func addSectorFace(bld *topo.Builder, ni, nj sectorNode, a, r0 math.UnitVector3, m0i, mAi *topo.Edge, weld float64, feat string, i int) {
	switch classifySectorEdge(ni, nj, a, weld) {
	case revEdgeOnAxis:
		return
	case revEdgeCylinder:
		addSectorCylinder(bld, ni, nj, a, r0, m0i, mAi, feat, i)
	case revEdgePlane:
		addSectorPlane(bld, ni, nj, a, m0i, mAi, feat, i)
	default:
		addSectorCone(bld, ni, nj, a, r0, m0i, mAi, feat, i)
	}
}

// classifySectorEdge dispatches a straight meridian edge on its direction, mirroring
// classifyRevolveEdge but reading the sector nodes (r == 0 marks an axis apex).
func classifySectorEdge(ni, nj sectorNode, a math.UnitVector3, weld float64) revEdgeClass {
	if ni.r == 0 && nj.r == 0 {
		return revEdgeOnAxis
	}
	dr := stdmath.Abs(ni.r - nj.r)
	dz := stdmath.Abs(float64(ni.center.VectorTo(nj.center).Dot(a.AsVector())))
	switch {
	case stdmath.Hypot(dr, dz) <= weld:
		return revEdgeCylinder
	case dr <= revSlopeTol*dz:
		return revEdgeCylinder
	case dz <= revSlopeTol*dr:
		return revEdgePlane
	default:
		return revEdgeCone
	}
}

// sectorLoop builds the four-edge boundary of a sector face — arc_i, the end-angle meridian, arc_j
// reversed, the start-angle meridian reversed — collapsing an apex end (no arc) to a triangle, the
// same way coneLoop collapses the full-revolution cone tip.
func sectorLoop(ni, nj sectorNode, m0i, mAi *topo.Edge) topo.LoopSpec {
	switch {
	case ni.arc == nil: // apex at i
		return topo.OuterLoop(topo.Fwd(mAi), topo.Rev(nj.arc), topo.Rev(m0i))
	case nj.arc == nil: // apex at j
		return topo.OuterLoop(topo.Fwd(ni.arc), topo.Fwd(mAi), topo.Rev(m0i))
	default:
		return topo.OuterLoop(topo.Fwd(ni.arc), topo.Fwd(mAi), topo.Rev(nj.arc), topo.Rev(m0i))
	}
}

// addSectorCylinder adds the partial cylindrical wall for an axis-parallel edge, bounded by the two
// vertex arcs and the two cap meridians. Like the full builder an upward edge (dz>0) faces +radial
// (AddFace); a downward edge faces −radial (a bore, AddReversedFace).
func addSectorCylinder(bld *topo.Builder, ni, nj sectorNode, a, r0 math.UnitVector3, m0i, mAi *topo.Edge, feat string, i int) {
	cyl, err := geom.NewCylinderWithRef(ni.center, a.AsVector(), r0.AsVector(), ni.r)
	if err != nil {
		return
	}
	loop := sectorLoop(ni, nj, m0i, mAi)
	if float64(ni.center.VectorTo(nj.center).Dot(a.AsVector())) > 0 { // dz>0: i below j → outer wall
		bld.AddFace(cyl, revLin(feat, "sector-wall", i), loop)
		return
	}
	bld.AddReversedFace(cyl, revLin(feat, "sector-wall", i), loop)
}

// addSectorCone adds the partial conical wall for an oblique edge, sharing the meridian (r0) frame so
// its seam lines up with the neighbours, bounded by the sector's arcs and meridians.
func addSectorCone(bld *topo.Builder, ni, nj sectorNode, a, r0 math.UnitVector3, m0i, mAi *topo.Edge, feat string, i int) {
	apex := sectorConeApex(ni, nj)
	base := ni
	if ni.arc == nil {
		base = nj
	}
	axisDir := apex.VectorTo(base.center)
	half := stdmath.Atan2(base.r, stdmath.Abs(float64(axisDir.Dot(a.AsVector()))))
	cone, err := geom.NewConeWithRef(apex, axisDir, r0.AsVector(), half)
	if err != nil {
		return
	}
	loop := sectorLoop(ni, nj, m0i, mAi)
	if float64(ni.center.VectorTo(nj.center).Dot(a.AsVector())) > 0 { // dz>0: outward-facing cone
		bld.AddFace(cone, revLin(feat, "sector-cone", i), loop)
		return
	}
	bld.AddReversedFace(cone, revLin(feat, "sector-cone", i), loop)
}

// sectorConeApex is the axis point where the oblique edge's slant line meets the axis (r=0): an apex
// endpoint IS that point, otherwise it is the linear extrapolation to zero radius.
func sectorConeApex(ni, nj sectorNode) math.Point3 {
	if ni.arc == nil {
		return ni.v0.Point()
	}
	if nj.arc == nil {
		return nj.v0.Point()
	}
	t := ni.r / (ni.r - nj.r)
	return ni.v0.Point().TranslateBy(ni.v0.Point().VectorTo(nj.v0.Point()).Scale(math.Scalar(t)))
}

// addSectorPlane adds the planar disk/annulus SECTOR for an axis-perpendicular edge: a single pie-slice
// loop (the two radial cap meridians close the inner and outer arcs into one boundary, so unlike the
// full annulus there is no separate inner hole). The outward normal is −axis when the edge runs
// outward (dr>0) and +axis when inward — the same rule as the full disk, and it agrees with the
// sector loop's right-hand winding, so a plain AddFace orients it correctly.
func addSectorPlane(bld *topo.Builder, ni, nj sectorNode, a math.UnitVector3, m0i, mAi *topo.Edge, feat string, i int) {
	outward := a.AsVector()
	if nj.r > ni.r { // edge i→j runs inward→outward (dr>0) ⇒ face points −axis
		outward = a.AsVector().Scale(-1)
	}
	plane, err := geom.NewPlane(ni.center, outward)
	if err != nil {
		return
	}
	bld.AddFace(plane, revLin(feat, "sector-disk", i), sectorLoop(ni, nj, m0i, mAi))
}

// addSectorCaps adds the two planar caps: the meridian in the start (angle 0) half-plane, facing −t0
// (away from the swept material), and in the end half-plane, facing +t (toward it). t0 = axis × r0 is
// the +angle sweep direction, so the start cap normal is −t0 and the end cap normal +t (t at the end
// angle). Each cap boundary is its ring of meridian edges; the end ring is reversed because its normal
// is opposite. On-axis seam edges (m0[i] == mA[i]) are used once by each cap.
func addSectorCaps(bld *topo.Builder, a, r0 math.UnitVector3, angle float64, axisOrigin math.Point3, m0, mA []*topo.Edge, feat string) {
	n := len(m0)
	t0 := a.AsVector().Cross(r0.AsVector()) // +angle sweep direction at the start
	startPlane, err := geom.NewPlane(axisOrigin, t0.Scale(-1))
	if err == nil {
		start := make([]topo.Use, n)
		for i := range n {
			start[i] = topo.Fwd(m0[i])
		}
		bld.AddFace(startPlane, revLin(feat, "sector-start-cap", 0), topo.OuterLoop(start...))
	}
	rA := math.Rotation4(angle, a, axisOrigin).TransformVector(r0.AsVector())
	tA := a.AsVector().Cross(rA) // +angle sweep direction at the end angle
	endPlane, err := geom.NewPlane(axisOrigin, tA)
	if err == nil {
		end := make([]topo.Use, n)
		for i := range n {
			end[i] = topo.Rev(mA[n-1-i])
		}
		bld.AddFace(endPlane, revLin(feat, "sector-end-cap", 0), topo.OuterLoop(end...))
	}
}
