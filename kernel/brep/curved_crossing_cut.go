// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Crossing-cylinder cut (M2 Phase 2, Oblikovati/Oblikovati#1335). Drilling a fat cylinder with a crossing
// rod: Boolean(Cut, fat, rod) removes the rod from the fat, leaving the fat with a clean cylindrical
// tunnel bored through its side. The result is assembled straight from the imprint loops, exact analytic
// surfaces preserved: the fat's two planar caps (untouched), the fat SIDE wall now carrying the two lens
// HOLES the rod broke through, and the rod-wall band as the tunnel wall (the rod surface flipped inward,
// since the kept material is OUTSIDE the rod). Scope: the in-scope thin-through-fat side breach where the
// tunnel passes through the wall (not the caps) and the loops lie strictly between the fat caps; anything
// else defers to the caller's fallback.

// CrossingCylinderCut returns target − tool for two crossing cylinders, or ok=false to defer (out-of-scope
// configurations are left to the caller's fallback). Both subtraction directions are built from the same
// imprint loops: when target is the fat cylinder the rod bores through, the result is the fat with a clean
// tunnel (drillFatWithRod); when target is the rod, the result is the two disconnected rod stubs sticking
// out either side of the fat (cutRodMinusFat).
//
// Example — a radius-3 cylinder drilled by a radius-1.5 rod:
//
//	fat, _  := brep.SolidCylinder(math.P3(0,0,-6), math.V3(0,0,1), 3, 12)
//	thin, _ := brep.SolidCylinder(math.P3(-6,0,0), math.V3(1,0,0), 1.5, 12)
//	res, ok := brep.CrossingCylinderCut(fat, thin) // fat with a tunnel
//	res, ok = brep.CrossingCylinderCut(thin, fat)  // the two rod stubs
func CrossingCylinderCut(target, tool *topo.Body) (*topo.Body, bool) {
	p, ok := crossingPartsOf(target, tool)
	if !ok {
		return nil, false
	}
	if !loopsBetweenCaps(p.fat, p.fatBase, p.fatHeight, p.lo, p.hi) {
		return nil, false // the breach reaches a fat cap (not a clean side breach): out of scope
	}
	tc, _, _, ok := cylinderSolidParams(facesOfAny(target))
	if !ok {
		return nil, false
	}
	if allLoopsEncircle([]geom.Polyline{p.lo, p.hi}, tc) {
		return cutRodMinusFat(p), true // target is the rod: two disconnected stubs
	}
	return drillFatWithRod(p), true // target is the fat: drill a tunnel
}

// crossingParts holds the resolved geometry of two crossing cylinders: the rod (both imprint loops encircle
// it) and the fat cylinder, each with its cap extent, plus the two imprint loops oriented CCW about the rod
// axis and assigned to the lower (lo) and upper (hi) end of the band (see assignRimLoops).
type crossingParts struct {
	rod, fat             geom.Cylinder
	rodBase, fatBase     math.Point3
	rodHeight, fatHeight float64
	lo, hi               geom.Polyline
}

// crossingPartsOf resolves two bodies into crossingParts, or ok=false when they are not two bare cylinders
// crossing in the thin-through-fat configuration (exactly two imprint loops, one cylinder the rod both
// encircle).
func crossingPartsOf(a, b *topo.Body) (*crossingParts, bool) {
	loops, ok := crossingCylinderImprint(a, b)
	if !ok || len(loops) != 2 {
		return nil, false
	}
	ca, baseA, hA, okA := cylinderSolidParams(facesOfAny(a))
	cb, baseB, hB, okB := cylinderSolidParams(facesOfAny(b))
	if !okA || !okB {
		return nil, false
	}
	rod, rodBase, rodH, fat, fatBase, fatH, ok := orderRodFat(loops, ca, baseA, hA, cb, baseB, hB)
	if !ok || !loopsSpanRod(rod, rodBase, rodH, loops) {
		return nil, false // a loop beyond a rod end is a partial penetration, not a full crossing
	}
	lo, hi, ok := assignRimLoops(rod, fat, loops)
	if !ok {
		return nil, false
	}
	return &crossingParts{rod, fat, rodBase, fatBase, rodH, fatH, lo, hi}, true
}

// orderRodFat picks which of the two cylinders is the rod (both loops encircle it) and which is the fat
// cylinder, carrying each one's cap base and height through.
func orderRodFat(loops []geom.Polyline, ca geom.Cylinder, baseA math.Point3, hA float64, cb geom.Cylinder, baseB math.Point3, hB float64) (rod geom.Cylinder, rodBase math.Point3, rodH float64, fat geom.Cylinder, fatBase math.Point3, fatH float64, ok bool) {
	switch {
	case allLoopsEncircle(loops, ca) && !allLoopsEncircle(loops, cb):
		return ca, baseA, hA, cb, baseB, hB, true
	case allLoopsEncircle(loops, cb) && !allLoopsEncircle(loops, ca):
		return cb, baseB, hB, ca, baseA, hA, true
	default:
		return geom.Cylinder{}, math.Point3{}, 0, geom.Cylinder{}, math.Point3{}, 0, false
	}
}

// loopsBetweenCaps reports whether both imprint loops lie strictly between the fat cylinder's two caps —
// the side-breach condition where the tunnel passes through the wall and leaves the caps whole.
func loopsBetweenCaps(fat geom.Cylinder, fatBase math.Point3, fatHeight float64, lo, hi geom.Polyline) bool {
	axis := fat.AxisDir.AsVector()
	vBot := float64(fat.Origin.VectorTo(fatBase).Dot(axis))
	for _, lp := range []geom.Polyline{lo, hi} {
		for _, p := range lp.Vertices {
			s := float64(fat.Origin.VectorTo(p).Dot(axis)) - vBot
			if s < 1e-9 || s > fatHeight-1e-9 {
				return false
			}
		}
	}
	return true
}

// drillFatWithRod builds the fat−rod solid: the fat's two caps, its side wall carrying the two lens holes,
// and the rod-wall tunnel band (the rod surface flipped inward). Every imprint-loop edge is shared by the
// holed wall and the tunnel band in opposite orientations, so the result is a closed manifold.
func drillFatWithRod(p *crossingParts) *topo.Body {
	bld := topo.NewBuilder(true, crossLin("body"))
	loPts, hiPts := p.lo.Vertices, p.hi.Vertices
	vLo := bld.AddVertex(loPts[0], crossLin("vlo"))
	vHi := bld.AddVertex(hiPts[0], crossLin("vhi"))
	eLo := bld.AddEdge(p.lo, vLo, vLo, crossLin("elo"))
	eHi := bld.AddEdge(p.hi, vHi, vHi, crossLin("ehi"))
	rSeam := bld.AddEdge(geom.NewLineSegment(loPts[0], hiPts[0]), vLo, vHi, crossLin("rseam"))
	// The tunnel wall is the rod band flipped inward (kept material is outside the rod).
	bld.AddReversedFace(p.rod, crossLin("tunnel"),
		topo.OuterLoop(topo.Fwd(rSeam), topo.Rev(eHi), topo.Rev(rSeam), topo.Fwd(eLo)))
	addFatCapsAndHoledWall(bld, p.fat, p.fatBase, p.fatHeight, clearSeamParam(p.fat, p.lo, p.hi),
		topo.Rev(eLo), topo.Fwd(eHi))
	return bld.Build()
}

// addFatCapsAndHoledWall adds the fat cylinder's two planar caps and its side wall carrying the given hole
// loops (one per wall breach). seam is an angular parameter placed clear of every hole (the unroll-and-CDT
// mesher needs a hole-free seam); each hole use is the lens-loop edge in the orientation OPPOSITE the band
// that fills it from the other side, so every edge stays used exactly twice.
func addFatCapsAndHoledWall(bld *topo.Builder, fat geom.Cylinder, fatBase math.Point3, fatHeight, seam float64, holes ...topo.Use) {
	axis := fat.AxisDir.AsVector()
	vBot := float64(fat.Origin.VectorTo(fatBase).Dot(axis))
	topCenter := fatBase.TranslateBy(axis.Scale(math.Scalar(fatHeight)))
	seamBot, seamTop := fat.PointAt(seam, vBot), fat.PointAt(seam, vBot+fatHeight)
	botC := seamedCircle(fatBase, fat.AxisDir, seamBot, fat.Radius)
	topC := seamedCircle(topCenter, fat.AxisDir, seamTop, fat.Radius)

	vb := bld.AddVertex(seamBot, crossLin("fvb"))
	vt := bld.AddVertex(seamTop, crossLin("fvt"))
	eBot := bld.AddEdge(botC, vb, vb, crossLin("febot"))
	eTop := bld.AddEdge(topC, vt, vt, crossLin("fetop"))
	eSeam := bld.AddEdge(geom.NewLineSegment(seamBot, seamTop), vb, vt, crossLin("fseam"))

	capBot, _ := geom.NewPlane(fatBase, axis.Scale(-1))
	capTop, _ := geom.NewPlane(topCenter, axis)
	bld.AddFace(capBot, crossLin("fcapbot"), topo.OuterLoop(topo.Rev(eBot)))
	bld.AddFace(capTop, crossLin("fcaptop"), topo.OuterLoop(topo.Fwd(eTop)))
	wallLoops := []topo.LoopSpec{topo.OuterLoop(topo.Fwd(eSeam), topo.Rev(eTop), topo.Rev(eSeam), topo.Fwd(eBot))}
	for _, h := range holes {
		wallLoops = append(wallLoops, topo.InnerLoop(h))
	}
	bld.AddFace(fat, crossLin("fwall"), wallLoops...)
}

// clearSeamParam returns an angular parameter on the fat cylinder midway through the larger gap between the
// two lens centres, so a seam placed there crosses neither hole (the holed-wall mesher needs a hole-free
// seam).
func clearSeamParam(fat geom.Cylinder, lo, hi geom.Polyline) float64 {
	u1 := axisAngleOf(loopCentroid(lo), fat)
	u2 := axisAngleOf(loopCentroid(hi), fat)
	a, b := stdmath.Min(u1, u2), stdmath.Max(u1, u2)
	if b-a >= 2*stdmath.Pi-(b-a) {
		return (a + b) / 2 // the gap from a to b is the larger one
	}
	return (a+b)/2 + stdmath.Pi // the gap wrapping past 0 is larger
}

// clearSeamForLens returns an angular parameter on the fat cylinder diametrically opposite a single lens
// hole, so a seam placed there crosses the hole-free side of the wall (the holed-wall mesher needs a
// hole-free seam).
func clearSeamForLens(fat geom.Cylinder, lens geom.Polyline) float64 {
	return axisAngleOf(loopCentroid(lens), fat) + stdmath.Pi
}

// seamedCircle builds a cap circle whose angle-zero seam vertex is at seamPt (radius and centre on the
// cap), so a cylinder wall sharing it can seam there.
func seamedCircle(center math.Point3, axis math.UnitVector3, seamPt math.Point3, radius float64) geom.Circle {
	ref, err := math.UnitVector3FromVector(center.VectorTo(seamPt))
	if err != nil {
		return geom.Circle{Center: center, Normal: axis, RefDir: axis, Radius: radius}
	}
	return geom.Circle{Center: center, Normal: axis, RefDir: ref, Radius: radius}
}
