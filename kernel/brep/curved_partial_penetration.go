// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Partial-penetration crossing-cylinder intersection (M2 Phase 2, Oblikovati/Oblikovati#1335). A thin rod
// that does NOT fully cross the fatter cylinder — it breaches one wall and ENDS inside, its blind end cap
// sitting within the fat solid. The rod surface then meets the fat surface in just ONE imprint loop (the
// single entry breach), not the two of a full crossing, so this case falls outside CrossingCylinderIntersect
// (which needs two loops) and is assembled here instead.
//
// The intersection (rod ∩ fat) is the rod "plug" inside the fat: three analytic faces — the fat-wall LENS
// cap (the one breach), the rod-wall stub BAND from that lens out to the rod's blind end, and the rod's
// blind END cap (the flat disc, wholly inside the fat). The lens edge is shared by the lens cap and the
// stub band, and the end circle by the stub band and the end cap, both in opposite orientation, so the plug
// is a closed manifold solid. The stub band and end cap reuse the rod stub builder shared with the join and
// drill (curved_crossing_join.go).

// partLin tags the assembled plug body's topology (one entity per role, so the index is always 0).
func partLin(role string) topo.Lineage { return topo.NewLineage(topo.Tok("partpen", role, 0)) }

// PartialPenetrationIntersect builds the exact intersection of a thin rod that ends inside a fatter
// cylinder (the rod plug), or ok=false when the configuration is outside that case (not exactly one imprint
// loop, neither cylinder is the rod the loop encircles, or no rod end lies cleanly inside the other) so
// kernel/ops keeps its CSG fallback.
//
// Example — a radius-1.5 rod on x ending at the centre of a radius-3 cylinder on z gives a three-face plug:
//
//	fat, _  := brep.SolidCylinder(math.P3(0,0,-6), math.V3(0,0,1), 3, 12)
//	stub, _ := brep.SolidCylinder(math.P3(-6,0,0), math.V3(1,0,0), 1.5, 6) // ends at x=0, inside the fat
//	res, ok := brep.PartialPenetrationIntersect(fat, stub)
func PartialPenetrationIntersect(a, b *topo.Body) (*topo.Body, bool) {
	p, ok := partialPlugPartsOf(a, b)
	if !ok {
		return nil, false
	}
	return partialPlug(p), true
}

// partialPlugParts holds the resolved geometry of a partial penetration: the rod (the single imprint loop
// encircles it) and the fat cylinder; the rod's blind end cap (inside the fat) and its entry end cap
// (outside the fat), each as a centre and outward axial normal; the single entry imprint loop oriented CCW
// about the rod axis; and the fat cylinder's cap base and height (for the wall the Cut and Join carry).
type partialPlugParts struct {
	rod, fat                 geom.Cylinder
	blindCenter, entryCenter math.Point3
	blindNormal, entryNormal math.Vector3
	lens                     geom.Polyline
	fatBase                  math.Point3
	fatHeight                float64
}

// partialPlugPartsOf resolves two bodies into a partial penetration, or ok=false when they are not a thin
// rod ending inside a fatter cylinder. The imprint is traced on the FIRST body's surface within that body's
// axial window, so it must be traced with the ROD first: the rod's short extent then clips the would-be
// exit loop of the two infinite surfaces, leaving the single entry loop (tracing fat-first would window the
// full crossing and return both loops). Which body is the rod is unknown up front, so both orderings are
// tried; the rod-first ordering is the one that yields exactly one loop encircling that body and exactly one
// of that body's ends inside the other.
func partialPlugPartsOf(a, b *topo.Body) (*partialPlugParts, bool) {
	if p, ok := tryPartialPlug(a, b); ok {
		return p, true
	}
	return tryPartialPlug(b, a)
}

// tryPartialPlug resolves a partial plug treating rodBody as the penetrating rod (traced first), or
// ok=false when that does not hold: not exactly one imprint loop, the loop does not encircle rodBody, or
// not exactly one of rodBody's ends lies inside fatBody.
func tryPartialPlug(rodBody, fatBody *topo.Body) (*partialPlugParts, bool) {
	loops, ok := crossingCylinderImprint(rodBody, fatBody)
	if !ok || len(loops) != 1 {
		return nil, false
	}
	rod, rodBase, rodH, okR := cylinderSolidParams(facesOfAny(rodBody))
	fat, fatBase, fatH, okF := cylinderSolidParams(facesOfAny(fatBody))
	if !okR || !okF || !allLoopsEncircle(loops, rod) {
		return nil, false
	}
	blindC, blindN, entryC, entryN, ok := rodEnds(rod, rodBase, rodH, fat, fatBase, fatH)
	if !ok {
		return nil, false
	}
	return &partialPlugParts{
		rod: rod, fat: fat,
		blindCenter: blindC, blindNormal: blindN,
		entryCenter: entryC, entryNormal: entryN,
		lens:    orientLoopCCW(loops[0], rod),
		fatBase: fatBase, fatHeight: fatH,
	}, true
}

// rodEnds returns the rod's blind end cap (inside the fat — the tip of a partial penetration) and its entry
// end cap (outside the fat — where the rod sticks out), each as a centre and the outward axial normal
// (pointing away from the rod material). ok=false unless exactly one of the rod's two ends is inside the fat
// (both inside is a rod fully contained, neither is a full crossing — both handled elsewhere).
func rodEnds(rod geom.Cylinder, rodBase math.Point3, rodH float64, fat geom.Cylinder, fatBase math.Point3, fatH float64) (blindCenter math.Point3, blindNormal math.Vector3, entryCenter math.Point3, entryNormal math.Vector3, ok bool) {
	axis := rod.AxisDir.AsVector()
	e0 := rodBase
	e1 := rodBase.TranslateBy(axis.Scale(math.Scalar(rodH)))
	in0 := rodEndInsideFat(rod, e0, fat, fatBase, fatH)
	in1 := rodEndInsideFat(rod, e1, fat, fatBase, fatH)
	switch {
	case in1 && !in0: // blind end at the +axis extreme (e1), entry end at the base (e0)
		return e1, axis, e0, axis.Scale(-1), true
	case in0 && !in1: // blind end at the base (e0), entry end at the +axis extreme (e1)
		return e0, axis.Scale(-1), e1, axis, true
	default:
		return math.Point3{}, math.Vector3{}, math.Point3{}, math.Vector3{}, false
	}
}

// rodEndInsideFat reports whether the whole rod end circle (radius = rod radius, centre = end) lies strictly
// inside the fat solid — sampled around the circle so a tilted disc is judged by its actual extent, not just
// its centre. This is the condition for the blind end cap to be a clean disc wholly within the fat.
func rodEndInsideFat(rod geom.Cylinder, center math.Point3, fat geom.Cylinder, fatBase math.Point3, fatH float64) bool {
	const samples = 24
	for k := 0; k < samples; k++ {
		ang := 2 * stdmath.Pi * float64(k) / samples
		p := center.TranslateBy(rod.NormalAt(ang, 0).Scale(math.Scalar(rod.Radius)))
		if !pointInsideCylinderSolid(fat, fatBase, fatH, p) {
			return false
		}
	}
	return true
}

// pointInsideCylinderSolid reports whether p is strictly inside the finite cylinder solid — within the
// radius and between the caps, by a small margin so a point on the surface counts as outside.
func pointInsideCylinderSolid(c geom.Cylinder, base math.Point3, height float64, p math.Point3) bool {
	const margin = 1e-7
	if float64(radialOf(p, c).Length()) > c.Radius-margin {
		return false
	}
	axis := c.AxisDir.AsVector()
	vBase := float64(c.Origin.VectorTo(base).Dot(axis))
	v := float64(c.Origin.VectorTo(p).Dot(axis))
	return v >= vBase+margin && v <= vBase+height-margin
}

// partialPlug welds the three faces of the rod plug: the fat-wall lens cap (the fat surface, natural
// outward normal — the plug is inside the fat), the rod stub band out to the blind end, and the rod's blind
// end cap. The lens edge is shared by the lens cap (reversed) and the stub band (forward), so the plug is a
// closed manifold solid.
func partialPlug(p *partialPlugParts) *topo.Body {
	bld := topo.NewBuilder(true, partLin("body"))
	lensStart := p.lens.Vertices[0]
	vLens := bld.AddVertex(lensStart, partLin("vlens"))
	eLens := bld.AddEdge(p.lens, vLens, vLens, partLin("elens"))
	bld.AddFace(p.fat, partLin("lenscap"), topo.OuterLoop(topo.Rev(eLens)))
	addRodStub(bld, p.rod, p.blindCenter, p.blindNormal, eLens, vLens, lensStart, true, false, "plug")
	return bld.Build()
}

// PartialPenetrationCut builds target − tool when tool is a thin rod that ends inside the fatter target (a
// BLIND HOLE), or ok=false to defer. The result is the fat with a blind cylindrical pocket: its two caps,
// its side wall carrying the single lens hole, the rod-wall tunnel flipped inward, and the rod's blind end
// cap as the pocket bottom.
//
// Example — a radius-3 cylinder blind-drilled by a radius-1.5 rod ending at its centre:
//
//	fat, _  := brep.SolidCylinder(math.P3(0,0,-6), math.V3(0,0,1), 3, 12)
//	stub, _ := brep.SolidCylinder(math.P3(-6,0,0), math.V3(1,0,0), 1.5, 6)
//	res, ok := brep.PartialPenetrationCut(fat, stub) // fat with a blind pocket
func PartialPenetrationCut(target, tool *topo.Body) (*topo.Body, bool) {
	p, ok := partialPlugPartsOf(target, tool)
	if !ok || !targetIsFat(target, p) {
		return nil, false // only fat − rod is the blind hole; rod − fat (a stub lump) defers for now
	}
	return blindHole(p), true
}

// PartialPenetrationJoin builds target ∪ tool for a thin rod ending inside the fatter cylinder, or ok=false
// to defer. The result is the fat with a single rod STUB sticking out the entry side: the fat's two caps,
// its holed side wall, the rod-wall stub band from the lens out to the rod's entry end, and the rod's entry
// end cap.
func PartialPenetrationJoin(a, b *topo.Body) (*topo.Body, bool) {
	p, ok := partialPlugPartsOf(a, b)
	if !ok {
		return nil, false
	}
	return partialJoinStub(p), true
}

// targetIsFat reports whether the Cut target is the fat cylinder of the resolved partial penetration (so the
// subtraction is the blind hole), rather than the rod.
func targetIsFat(target *topo.Body, p *partialPlugParts) bool {
	tc, _, _, ok := cylinderSolidParams(facesOfAny(target))
	return ok && nearEqual(tc.Radius, p.fat.Radius)
}

// blindHole welds the fat with a blind pocket: the fat caps and holed side wall (one lens hole), the rod
// tunnel flipped inward (kept material is outside the rod), and the rod's blind end cap as the pocket
// bottom. The lens edge is shared by the wall hole and the tunnel band in opposite orientation, so the
// result is a closed manifold solid.
func blindHole(p *partialPlugParts) *topo.Body {
	bld := topo.NewBuilder(true, partLin("body"))
	lensStart := p.lens.Vertices[0]
	vLens := bld.AddVertex(lensStart, partLin("vlens"))
	eLens := bld.AddEdge(p.lens, vLens, vLens, partLin("elens"))
	addFatCapsAndHoledWall(bld, p.fat, p.fatBase, p.fatHeight, clearSeamForLens(p.fat, p.lens), topo.Rev(eLens))
	addRodStub(bld, p.rod, p.blindCenter, p.blindNormal, eLens, vLens, lensStart, true, true, "blind")
	return bld.Build()
}

// partialJoinStub welds the union of the fat and a partially-penetrating rod: the fat caps and holed side
// wall (one lens hole), the rod stub band from the lens out to the rod's entry end, and the rod's entry end
// cap. The lens edge is shared by the wall hole and the stub band in opposite orientation, so the result is
// a closed manifold solid.
func partialJoinStub(p *partialPlugParts) *topo.Body {
	bld := topo.NewBuilder(true, partLin("body"))
	lensStart := p.lens.Vertices[0]
	vLens := bld.AddVertex(lensStart, partLin("vlens"))
	eLens := bld.AddEdge(p.lens, vLens, vLens, partLin("elens"))
	addFatCapsAndHoledWall(bld, p.fat, p.fatBase, p.fatHeight, clearSeamForLens(p.fat, p.lens), topo.Rev(eLens))
	addRodStub(bld, p.rod, p.entryCenter, p.entryNormal, eLens, vLens, lensStart, true, false, "jstub")
	return bld.Build()
}
