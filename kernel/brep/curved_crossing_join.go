// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Crossing-cylinder join and rod−fat cut (M2 Phase 2, Oblikovati/Oblikovati#1335). The two boolean
// outcomes that keep the rod material OUTSIDE the fat cylinder, assembled from the same imprint loops as the
// drill (curved_crossing_cut.go) and sharing its rod stub builder:
//
//   - Join (fat ∪ rod): one connected solid — the fat's two caps and its holed side wall (reused from the
//     drill), plus a rod-wall STUB protruding from each lens hole out to the rod's own end cap. Each lens
//     edge is shared by the wall (as a hole) and its stub in opposite orientation, so the union is watertight.
//   - Cut (rod − fat): two DISCONNECTED rod stubs (the rod sticking out either side of the fat), each a
//     closed lump — a stub band, the rod end cap, and the fat-wall lens closing its inner end (the fat
//     surface reversed, since the kept material is outside the fat). The two lumps are merged into one
//     multi-shell body.

// CrossingCylinderJoin builds fat ∪ rod for two crossing cylinders (a fat cylinder side-breached by a rod
// passing right through it), or ok=false to defer (a non-cylinder operand, equal radii, or a breach that
// reaches a fat cap) so kernel/ops keeps its fallback.
//
// Example — a radius-3 cylinder joined with a crossing radius-1.5 rod gives the fat with two rod stubs:
//
//	fat, _  := brep.SolidCylinder(math.P3(0,0,-6), math.V3(0,0,1), 3, 12)
//	thin, _ := brep.SolidCylinder(math.P3(-6,0,0), math.V3(1,0,0), 1.5, 12)
//	res, ok := brep.CrossingCylinderJoin(fat, thin)
func CrossingCylinderJoin(a, b *topo.Body) (*topo.Body, bool) {
	p, ok := crossingPartsOf(a, b)
	if !ok {
		return nil, false
	}
	if !loopsBetweenCaps(p.fat, p.fatBase, p.fatHeight, p.lo, p.hi) {
		return nil, false // a stub would breach a fat cap, not the side wall: out of scope
	}
	return joinFatAndRod(p), true
}

// joinFatAndRod welds the union: the fat's two caps and its holed side wall (the two lens holes), plus a rod
// stub out of each hole to the rod's end cap. Each lens edge is used by the wall (as a hole) and its stub in
// opposite orientation, so every edge is used exactly twice — a closed manifold solid.
func joinFatAndRod(p *crossingParts) *topo.Body {
	bld := topo.NewBuilder(true, crossLin("body"))
	vLo := bld.AddVertex(p.lo.Vertices[0], crossLin("vlo"))
	vHi := bld.AddVertex(p.hi.Vertices[0], crossLin("vhi"))
	eLo := bld.AddEdge(p.lo, vLo, vLo, crossLin("elo"))
	eHi := bld.AddEdge(p.hi, vHi, vHi, crossLin("ehi"))
	addFatCapsAndHoledWall(bld, p.fat, p.fatBase, p.fatHeight, clearSeamParam(p.fat, p.lo, p.hi),
		topo.Rev(eLo), topo.Fwd(eHi))
	axis := p.rod.axisVec()
	rodFar := p.rodBase.TranslateBy(axis.Scale(math.Scalar(p.rodHeight)))
	// The lo lens (fat-outward along −rodaxis) connects to the rod's base end; the hi lens to the far end.
	// The wall holes are InnerLoop(Rev(eLo)) and InnerLoop(Fwd(eHi)), so the stubs take the opposite half.
	addRodStub(bld, p.rod, p.rodBase, axis.Scale(-1), eLo, vLo, p.lo.Vertices[0], true, false, "slo")
	addRodStub(bld, p.rod, rodFar, axis, eHi, vHi, p.hi.Vertices[0], false, false, "shi")
	return bld.Build()
}

// addRodStub adds one rod stub to a join/cut body: the rod-wall band from the lens loop out to the rod's
// end cap, plus that planar end cap. The shared lens edge eLens is used in the orientation lensFwd so the
// stub welds opposite to the face that owns the other side of it (the holed wall, or the fat lens cap). The
// band keeps the rod's natural outward normal (the kept material is inside the rod) when reversed is false;
// when reversed is true it is flipped inward as a tunnel wall (the kept material is outside the rod, e.g. a
// blind hole) and the end cap faces back into the void. The end cap faces along capNormal (the outward
// ±rod-axis), negated when reversed. The end circle is re-seamed to the lens loop's start angle so the band
// loft stitches a near-constant-angle seam.
func addRodStub(bld *topo.Builder, rod crossRod, endCenter math.Point3, capNormal math.Vector3, eLens *topo.Edge, vLens *topo.Vertex, lensStart math.Point3, lensFwd, reversed bool, tag string) {
	seamPt := stubSeamPoint(rod, endCenter, lensStart)
	endC := seamedCircle(endCenter, rod.axisUnit(), seamPt, rod.endRadius(endCenter))
	vEnd := bld.AddVertex(seamPt, crossLin(tag+"ve"))
	eEnd := bld.AddEdge(endC, vEnd, vEnd, crossLin(tag+"ee"))
	eSeam := bld.AddEdge(geom.NewLineSegment(seamPt, lensStart), vEnd, vLens, crossLin(tag+"es"))
	lensHalf := topo.Fwd(eLens)
	if !lensFwd {
		lensHalf = topo.Rev(eLens)
	}
	band := topo.OuterLoop(lensHalf, topo.Rev(eSeam), topo.Fwd(eEnd), topo.Fwd(eSeam))
	capNorm := capNormal
	if reversed {
		bld.AddReversedFace(rod.surface(), crossLin(tag+"band"), band)
		capNorm = capNormal.Scale(-1)
	} else {
		bld.AddFace(rod.surface(), crossLin(tag+"band"), band)
	}
	cap, _ := geom.NewPlane(endCenter, capNorm)
	bld.AddFace(cap, crossLin(tag+"cap"), topo.OuterLoop(topo.Rev(eEnd)))
}

// stubSeamPoint returns the point on the rod's end circle at the same axial angle as the lens loop's start,
// so the seam connecting them is a near-constant-angle ruling of the rod (lying on the surface). It rebuilds
// the point from the rod's axis frame (ref·cos u + binormal·sin u, the convention both geom.Cylinder and
// geom.Cone use) at the end's radius, so it works for a cone end too.
func stubSeamPoint(rod crossRod, endCenter, lensStart math.Point3) math.Point3 {
	ax := rod.axisOf()
	u := axisAngleOf(lensStart, ax)
	radial := ax.ref.Scale(stdmath.Cos(u)).Add(ax.dir.Cross(ax.ref).Scale(stdmath.Sin(u)))
	return endCenter.TranslateBy(radial.Scale(rod.endRadius(endCenter)))
}

// cutRodMinusFat builds rod − fat: the two rod stubs sticking out either side of the fat, each a separate
// closed lump, merged into one multi-shell body (the boolean result is two disconnected pieces).
func cutRodMinusFat(p *crossingParts) *topo.Body {
	axis := p.rod.axisVec()
	rodFar := p.rodBase.TranslateBy(axis.Scale(math.Scalar(p.rodHeight)))
	lo := rodStubLump(p.rod, p.fat, p.rodBase, axis.Scale(-1), p.lo, "rlo")
	hi := rodStubLump(p.rod, p.fat, rodFar, axis, p.hi, "rhi")
	return topo.MergeBodies(crossLin("body"), true, lo, hi)
}

// rodStubLump builds one closed rod stub: its wall band, the rod end cap, and the fat-wall lens closing the
// inner end. The lens cap is the fat surface REVERSED — the kept material is outside the fat, so the lens
// normal points back into the fat (away from the stub). Its edge is shared with the stub band opposite.
func rodStubLump(rod crossRod, fat geom.Cylinder, endCenter math.Point3, capNormal math.Vector3, lens geom.Polyline, tag string) *topo.Body {
	bld := topo.NewBuilder(true, crossLin(tag))
	vLens := bld.AddVertex(lens.Vertices[0], crossLin(tag+"vl"))
	eLens := bld.AddEdge(lens, vLens, vLens, crossLin(tag+"el"))
	bld.AddReversedFace(fat, crossLin(tag+"lenscap"), topo.OuterLoop(topo.Rev(eLens)))
	addRodStub(bld, rod, endCenter, capNormal, eLens, vLens, lens.Vertices[0], true, false, tag)
	return bld.Build()
}
