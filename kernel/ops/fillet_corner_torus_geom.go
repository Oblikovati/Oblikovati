// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// mixedGeom is the fully-solved geometry of one mixed-sense torus corner: the torus (center, axis,
// major R, minor r), the shared host plane and its outward normal, and — per PAIRED band — the wall
// it shares with the pivot, its spine foot (F_i, the rolling-ball centre at the retracted
// station) and its two contact points (on the host plane and on the wall). Every downstream point
// (patch arcs, band cross-sections, host detour) is derived from these, so the four sub-arcs weld by
// construction. Indices [0]/[1] follow the two paired bands pair[0]/pair[1].
type mixedGeom struct {
	r, R      float64
	center    math.Point3 // torus centre C (on the convex fillet axis, at the concave-spine level)
	centerTop math.Point3 // centre of the top-contact circle (C dropped onto the host plane along the axis)
	axis      math.Vector3
	top       *topo.Face
	nTop      math.Vector3
	wall      [2]*topo.Face
	spine     [2]math.Point3
	pTop      [2]math.Point3
	pWall     [2]math.Point3
	pairBand  [2]cornerBand // the two paired bands' fils slots (for writing the retraction back)
}

// mixedCornerGeom solves the torus frame and every contact point. It finds the torus centre C as the
// point on the PIVOT band's fillet axis nearest the first paired spine (the common-perpendicular foot),
// verifies the pivot radius is the derived 2r, then fills each band's spine and tangent points. ok=false
// on a non-planar host or a pivot radius off 2r (a non-orthogonal / curved mixed corner, out of scope).
func mixedCornerGeom(pivot cornerBand, pair []cornerBand) (mixedGeom, bool) {
	top, ok := sharedBandFace(pair[0], pair[1])
	if !ok {
		return mixedGeom{}, false
	}
	nTop, ok := planeNormal(top)
	if !ok {
		return mixedGeom{}, false
	}
	axis := pivot.cyl.AxisDir.AsVector()
	center, foot := closestPointsBetweenLines(pivot.cyl.Origin, axis, pair[0].cyl.Origin, pair[0].cyl.AxisDir.AsVector())
	g := mixedGeom{r: pivot.cyl.Radius, R: center.DistanceTo(foot), center: center, axis: axis, top: top, nTop: nTop}
	if stdmath.Abs(g.R-2*g.r) > mixedTorusRadiusTol*g.r {
		return mixedGeom{}, false // not the 2r rolling pivot — a non-orthogonal / curved mixed corner
	}
	g.centerTop = dropOntoPlane(center, top, nTop)
	g.pairBand[0], g.pairBand[1] = pair[0], pair[1]
	if !fillMixedTangents(&g, pair) {
		return mixedGeom{}, false
	}
	return g, true
}

// fillMixedTangents fills each paired band's spine foot and its two contact points. The spine F_i is
// the rolling-ball centre on the band's axis at the retracted station (the axis point nearest C); the
// contact point on any face is F_i PROJECTED onto that face's plane — the foot of the perpendicular,
// which is the tangency point whichever SIDE the ball sits on. (The projection replaced a
// `F_i − r·n_outward` translate: that hard-codes the ball being on the face's outward side, which holds
// only for the 2-concave signature. For the 1-concave/2-convex corner the paired bands are CONVEX and
// their spines sit r on the INWARD side, so the translate landed the contact 2r away — the 12.36
// off-its-own-cylinder rail on B5/C4/D7. |F_i − face| is r by construction either way, so the
// projection is the same point on the old signature and the correct one on the new.)
// ok=false when a band's wall is non-planar.
func fillMixedTangents(g *mixedGeom, pair []cornerBand) bool {
	for i := 0; i < 2; i++ {
		g.spine[i] = footOnLine(g.center, pair[i].cyl.Origin, pair[i].cyl.AxisDir.AsVector())
		wall := otherBandFace(pair[i], g.top)
		nW, ok := planeNormal(wall)
		if !ok {
			return false
		}
		g.wall[i] = wall
		g.pTop[i] = dropOntoPlane(g.spine[i], g.top, g.nTop)
		g.pWall[i] = dropOntoPlane(g.spine[i], wall, nW)
	}
	return true
}

// mixedTorusPatch builds the torus corner face: the analytic geom.Torus and its four-arc boundary loop
// (the e1 tube arc, the inner-equator arc shared with the convex band, the e2 tube arc, and the top-
// contact arc shared with the host plane). ok=false on a degenerate torus/arc.
func mixedTorusPatch(g mixedGeom) (filletFace, bool) {
	tor, err := geom.NewTorus(g.center, g.axis, g.R, g.r)
	if err != nil {
		return filletFace{}, false
	}
	loop, ok := mixedPatchLoop(g)
	if !ok {
		return filletFace{}, false
	}
	return filletFace{surface: tor, loops: []filletLoop{loop}}, true
}

// mixedPatchLoop chains the torus patch's four bounding arcs into a closed loop. Each arc is a 90°
// quarter through its bisector midpoint: pair0's tube (radius r about spine0), the inner equator (radius
// R−r about the axis, shared with the PIVOT band), pair1's tube, and the top contact (radius R about the
// axis, shared with the host plane). orientFilletShell unifies the winding at assembly.
func mixedPatchLoop(g mixedGeom) (filletLoop, bool) {
	arcs := []cornerArcSpec{
		{g.pTop[0], g.pWall[0], g.spine[0], g.r},
		{g.pWall[0], g.pWall[1], g.center, g.R - g.r},
		{g.pWall[1], g.pTop[1], g.spine[1], g.r},
		{g.pTop[1], g.pTop[0], g.centerTop, g.R},
	}
	var fl filletLoop
	for _, a := range arcs {
		arc, err := geom.Arc3dByThreePoints(a.from, arcMidOnCircle(a.center, a.from, a.to, a.radius), a.to)
		if err != nil {
			return filletLoop{}, false
		}
		fl.pts = append(fl.pts, a.from)
		fl.curves = append(fl.curves, arc)
	}
	return fl, true
}

// cornerArcSpec is one 90° corner sub-arc from→to on the circle (center, radius); its midpoint is the
// on-circle bisector point.
type cornerArcSpec struct {
	from, to, center math.Point3
	radius           float64
}

// mixedTopCorner is the synthetic simple END corner injected into the host plane's re-trim so the
// shared plane's receded vertex expands into the torus top-contact arc (radius R about the axis). Its
// two face-tangent points are the paired bands' host-plane contacts, keyed so addEndCorner's tOf picks
// the right one for either loop-traversal direction: ta on pair0's wall, tb on pair1's wall.
func mixedTopCorner(g mixedGeom, v *topo.Vertex) corner {
	return corner{
		a: g.wall[0], b: g.wall[1],
		ta: g.pTop[0], tb: g.pTop[1],
		mid:     arcMidOnCircle(g.centerTop, g.pTop[0], g.pTop[1], g.R),
		endFace: g.top, vertex: v,
	}
}

// mixedBandRetracts builds the three bands' rewritten cross-sections: each PAIRED band retracts to its
// tube arc (host + wall contacts), the PIVOT band retracts to the inner-equator arc (its two wall
// contacts) — the SAME four contact arcs the torus patch is bounded by, so band and patch weld.
func mixedBandRetracts(g mixedGeom, pivot cornerBand) [3]bandRetract {
	return [3]bandRetract{
		bandRetractFor(g.pairBand[0], pointByFace(g.top, g.pTop[0], g.wall[0], g.pWall[0]), arcMidOnCircle(g.spine[0], g.pTop[0], g.pWall[0], g.r)),
		bandRetractFor(g.pairBand[1], pointByFace(g.top, g.pTop[1], g.wall[1], g.pWall[1]), arcMidOnCircle(g.spine[1], g.pTop[1], g.pWall[1], g.r)),
		bandRetractFor(pivot, pointByFace(g.wall[0], g.pWall[0], g.wall[1], g.pWall[1]), arcMidOnCircle(g.center, g.pWall[0], g.pWall[1], g.R-g.r)),
	}
}

// pointByFace keys two contact points by their face id — the per-face lookup a band retract applies to
// whichever of a corner's (a,b) faces matches.
func pointByFace(fa *topo.Face, pa math.Point3, fb *topo.Face, pb math.Point3) map[uint64]math.Point3 {
	return map[uint64]math.Point3{fa.ID(): pa, fb.ID(): pb}
}

// bandRetractFor packages a band's rewritten cross-section (its fils slot + the per-face contact points
// + the section-arc midpoint).
func bandRetractFor(b cornerBand, pt map[uint64]math.Point3, mid math.Point3) bandRetract {
	return bandRetract{fi: b.fi, atC1: b.atC1, pt: pt, mid: mid}
}
