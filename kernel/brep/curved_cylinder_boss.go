// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cylindrical boss union (M2 Phase 3, Oblikovati/Oblikovati#1336 — the coplanar curved+planar overlap
// case). A cylinder sitting flush on a planar face of a solid (a boss: think a spigot on a plate) shares
// that face's plane: its base disk is COINCIDENT and counter-oriented with the disk it covers, the
// canonical coplanar overlap. The general boolean faceted this through CSG. The union is exact — the seat
// face loses the boss-footprint disk (gains a circular hole), the boss adds one cylinder wall and a top
// cap — so build it directly through the curvedFace model and keep the analytic surface. The boss is the
// join analogue of CutCylindricalHole: a hole in the seat face plus an OUTWARD-facing wall (a solid boss,
// material toward the axis) rather than a reversed one. Anything else (the cylinder not seated flush and
// perpendicular on a single face, the circle clipping the face) returns ok=false so the caller keeps CSG.

// JoinCylindricalBoss returns target ∪ tool when tool is a cylinder seated flush and perpendicular on one
// planar face of target, protruding outward (a boss), or ok=false to defer.
//
// Example:
//
//	plate, _ := brep.SolidBlock(math.P3(-5,-5,0), math.P3(5,5,2), "plate")
//	boss, _ := brep.SolidCylinder(math.P3(0,0,2), math.V3(0,0,1), 1.5, 3) // sits on the z=2 face
//	u, ok := brep.JoinCylindricalBoss(plate, boss)                        // ok: plate + boss
func JoinCylindricalBoss(target, tool *topo.Body) (*topo.Body, bool) {
	cyl, base, height, ok := cylinderSolidParams(facesOfAny(tool))
	if !ok {
		return nil, false
	}
	ua := cyl.AxisDir.AsVector()
	c0, c1 := base, base.TranslateBy(ua.Scale(math.Scalar(height)))
	faces := facesOfAny(target)
	near, far, seat, ok := bossSeatFace(faces, c0, c1, ua, cyl.Radius)
	if !ok {
		return nil, false
	}
	body := assembleBoss(faces, seat, near, far, cyl.Radius)
	if body == nil {
		return nil, false
	}
	return body, true
}

// bossSeatFace finds the planar face one cap sits flush on (perpendicular to the axis, the circle clear
// inside it) with the boss protruding to that face's OUTWARD side. It returns the seated (near) and
// protruding (far) cap centres and the face index, or ok=false when no face seats the boss.
func bossSeatFace(faces []curvedFace, c0, c1 math.Point3, ua math.Vector3, radius float64) (near, far math.Point3, idx int, ok bool) {
	for i, f := range faces {
		pl, isPlane := f.surface.(geom.Plane)
		if !isPlane || stdmath.Abs(float64(unit(pl.Normal()).Dot(ua))) < 1-1e-7 {
			continue // not a face perpendicular to the boss axis
		}
		nOut := faceOutwardNormal(f, pl)
		for _, cap := range [2][2]math.Point3{{c0, c1}, {c1, c0}} {
			n, fr := cap[0], cap[1]
			if stdmath.Abs(pointPlaneDistance(n, pl)) > axialTouchTol {
				continue // this cap is not flush on the face plane
			}
			if float64(n.VectorTo(fr).Dot(nOut)) <= 0 {
				continue // boss would protrude into the solid, not out of the seat face
			}
			if _, clean := circleVsCap(n, radius, f, pl); clean {
				return n, fr, i, true
			}
		}
	}
	return math.Point3{}, math.Point3{}, 0, false
}

// assembleBoss rebuilds the union: every target face unchanged except the seat face, which gains the
// boss-footprint hole, plus the outward boss wall and its top cap, re-welded through curvedStitch.
func assembleBoss(faces []curvedFace, seat int, near, far math.Point3, radius float64) *topo.Body {
	axis := unit(near.VectorTo(far)) // near → far, the boss's outward direction
	nearCirc, err := geom.NewCircle(near, axis, radius)
	if err != nil {
		return nil
	}
	farCirc := geom.Circle{Center: far, Normal: nearCirc.Normal, RefDir: nearCirc.RefDir, Radius: radius}

	out := make([]curvedFace, 0, len(faces)+2)
	for i, f := range faces {
		if i == seat {
			out = append(out, capWithHoleReversed(f, nearCirc)) // the disk the boss covers becomes a hole
			continue
		}
		out = append(out, f)
	}
	out = append(out, bossWallFace(near, axis, radius, nearCirc, farCirc), bossTopCapFace(far, axis, farCirc))
	return curvedStitch(out)
}

// capWithHoleReversed appends the circle to a face as an inner loop wound REVERSED — opposite to
// capWithHole — so its single use is counter to the boss wall's forward use of the same circle (the wall
// reuses the bottom circle the way SolidCylinder's side does, Fwd(eb)).
func capWithHoleReversed(f curvedFace, circ geom.Circle) curvedFace {
	hole := curvedLoop{edges: []loopEdge{{curve: circ, t0: 1, t1: 0}}}
	f.loops = append(append([]curvedLoop{}, f.loops...), hole)
	return f
}

// bossWallFace builds the boss side as an OUTWARD cylinder face (material toward the axis, a solid boss —
// AddFace, not the reversed hole wall) with the same seamed loop SolidCylinder's side uses (seam up, far
// circle reversed, seam down, near circle forward), so it welds to the seat-face hole and the top cap
// along the shared circle edges.
func bossWallFace(baseCenter math.Point3, axis math.Vector3, radius float64, nearCirc, farCirc geom.Circle) curvedFace {
	cyl, _ := geom.NewCylinder(baseCenter, axis, radius)
	seam := geom.NewLineSegment(nearCirc.PointAt(0), farCirc.PointAt(0))
	loop := curvedLoop{edges: []loopEdge{
		{curve: seam, t0: 0, t1: 1},
		{curve: farCirc, t0: 1, t1: 0},
		{curve: seam, t0: 1, t1: 0},
		{curve: nearCirc, t0: 0, t1: 1},
	}}
	return curvedFace{surface: cyl, reversed: false, loops: []curvedLoop{loop}, lineage: topo.NewLineage(topo.Tok("brep", "bosswall", 0))}
}

// bossTopCapFace builds the boss's outer cap disk (outward normal along the axis, away from the seat).
func bossTopCapFace(center math.Point3, axis math.Vector3, farCirc geom.Circle) curvedFace {
	pl, _ := geom.NewPlane(center, axis)
	loop := curvedLoop{edges: []loopEdge{{curve: farCirc, t0: 0, t1: 1}}}
	return curvedFace{surface: pl, reversed: false, loops: []curvedLoop{loop}, lineage: topo.NewLineage(topo.Tok("brep", "bosscap", 0))}
}

// faceOutwardNormal returns a face's true outward normal: the plane normal, flipped when the face was
// added reversed (its material side is the plane's positive side).
func faceOutwardNormal(f curvedFace, pl geom.Plane) math.Vector3 {
	n := unit(pl.Normal())
	if f.reversed {
		return n.Scale(-1)
	}
	return n
}

// pointPlaneDistance returns the signed distance of p from the plane (positive on the normal side).
func pointPlaneDistance(p math.Point3, pl geom.Plane) float64 {
	return float64(pl.Origin.VectorTo(p).Dot(unit(pl.Normal())))
}
