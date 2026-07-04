// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Partial cylindrical boss union — the curved-on-planar boolean KIND for a boss whose base circle CLIPS the
// seat face edge (a spigot straddling / overhanging a plate edge), #1591 / ADR-0049. Where JoinCylindricalBoss
// requires the base circle strictly inside one seat face, this handles the base circle crossing the seat
// boundary: the seat face keeps the plate minus the in-seat footprint (a planeUV trim), the overhang gets an
// exposed underside cap (the same base plane, inside the boss, outside the plate), the boss adds a full wall
// (its base split at the two crossings so it welds to seat + cap) and a top cap, and every OTHER target face
// that the crossings land on is split there so curvedStitch (which does not resolve T-junctions) welds clean.

// JoinPartialBoss returns target ∪ tool when tool is a cylinder seated flush and perpendicular on one planar
// face of target, protruding outward, with its base circle CLIPPING that face's boundary (a straddling boss),
// or ok=false to defer (a strictly-interior boss takes JoinCylindricalBoss; anything else keeps CSG).
func JoinPartialBoss(target, tool *topo.Body) (*topo.Body, bool) {
	cyl, base, height, ok := cylinderSolidParams(facesOfAny(tool))
	if !ok {
		return nil, false
	}
	ua := cyl.AxisDir.AsVector()
	c0, c1 := base, base.TranslateBy(ua.Scale(math.Scalar(height)))
	faces := facesOfAny(target)
	seat, near, far, ok := partialBossSeat(faces, c0, c1, ua, cyl.Radius)
	if !ok {
		return nil, false
	}
	body := assemblePartialBoss(faces, seat, near, far, cyl.Radius)
	if body == nil {
		return nil, false
	}
	return body, true
}

// partialBossSeat finds the planar face a boss cap sits flush on, protruding outward, whose base circle
// CLIPS the face boundary (pierced centre but not clean) — the straddling case. Returns the seat index and
// the seated (near) and protruding (far) cap centres, or ok=false.
func partialBossSeat(faces []curvedFace, c0, c1 math.Point3, ua math.Vector3, radius float64) (idx int, near, far math.Point3, ok bool) {
	for i, f := range faces {
		pl, isPlane := f.surface.(geom.Plane)
		if !isPlane || stdmath.Abs(float64(unit(pl.Normal()).Dot(ua))) < 1-1e-7 {
			continue
		}
		nOut := faceOutwardNormal(f, pl)
		flushTol := geom.ResolutionForSize(radius).Plane()
		for _, cap := range [2][2]math.Point3{{c0, c1}, {c1, c0}} {
			n, fr := cap[0], cap[1]
			if stdmath.Abs(pointPlaneDistance(n, pl)) > flushTol || float64(n.VectorTo(fr).Dot(nOut)) <= 0 {
				continue
			}
			if pierced, clean := circleVsCap(n, radius, f, pl); pierced && !clean {
				return i, n, fr, true
			}
		}
	}
	return 0, math.Point3{}, math.Point3{}, false
}

// assemblePartialBoss builds the union: the planeUV-trimmed seat, the overhang underside cap, the split-base
// boss wall and top cap, plus every other face split at the crossings so the T-junctions weld.
func assemblePartialBoss(faces []curvedFace, seatIdx int, near, far math.Point3, radius float64) *topo.Body {
	nearCirc, farCirc, axis, ok := bossBaseCircles(near, far, radius)
	if !ok {
		return nil
	}
	seatFace := faces[seatIdx]
	pl, _ := seatFace.surface.(geom.Plane)
	c := bossPlaneUV(seatFace, pl, near, axis, radius)
	crossings := c.planeCrossingsOf(nearCirc)
	if len(crossings) != 2 {
		return nil // Slice B scope: exactly one clipped edge → two crossings
	}
	seatOut, capOut, ok := trimSeatAndCap(c, seatFace, pl, nearCirc)
	if !ok {
		return nil
	}
	out := splitFacesAtCrossings(copyExcept(faces, seatIdx), crossings)
	out = append(out, seatOut...)
	out = append(out, capOut...)
	out = append(out, partialBossWall(near, axis, radius, nearCirc, farCirc, crossings), partialBossTopCap(far, axis, farCirc, bossSeamAngle(crossings)))
	return curvedStitch(out)
}

// bossBaseCircles builds the boss's near (seated) and far (protruding) base circles and axis, sharing one
// frame (Normal/RefDir) so their parameter t agrees — the wall's seam and split arcs anchor on that shared t.
func bossBaseCircles(near, far math.Point3, radius float64) (nearC, farC geom.Circle, axis math.Vector3, ok bool) {
	axis = unit(near.VectorTo(far))
	nearC, err := geom.NewCircle(near, axis, radius)
	if err != nil {
		return geom.Circle{}, geom.Circle{}, math.Vector3{}, false
	}
	farC = geom.Circle{Center: far, Normal: nearC.Normal, RefDir: nearC.RefDir, Radius: radius}
	return nearC, farC, axis, true
}

// trimSeatAndCap trims the seat plane to keep the plate minus the in-boss footprint (planeMaterial) and the
// coincident overhang underside to keep the in-boss-but-off-plate cantilever cap (capMaterial). ok=false if
// either trim yields nothing (a degenerate contact outside Slice B's straddling scope).
func trimSeatAndCap(c *planeUV, seatFace curvedFace, pl geom.Plane, nearCirc geom.Circle) (seat, cap []curvedFace, ok bool) {
	seat, _, err := trimByImprint(c, seatFace, pl, []geom.Curve3{nearCirc}, planeMaterial(c))
	if err != nil || len(seat) == 0 {
		return nil, nil, false
	}
	capIn := curvedFace{surface: pl, reversed: !seatFace.reversed, lineage: topo.NewLineage(topo.Tok("brep", "bosscapunder", 0))}
	cap, _, err = trimByImprint(c, capIn, pl, []geom.Curve3{nearCirc}, capMaterial(c))
	if err != nil || len(cap) == 0 {
		return nil, nil, false
	}
	return seat, cap, true
}

// bossSeamAngle returns the smaller of the two crossing parameters — the conic t where the wall seam and the
// top cap anchor, so the full top circle is never bisected by the seam (it starts and ends at a crossing).
func bossSeamAngle(crossings []planeCrossing) float64 {
	ta := crossings[0].tConic
	if crossings[1].tConic < ta {
		return crossings[1].tConic
	}
	return ta
}

// bossPlaneUV builds the planeUV operand for the seat plane: the seat polygon as the frame, and inside-the-
// boss (radial distance below the base circle radius) as the tool membership.
func bossPlaneUV(seatFace curvedFace, pl geom.Plane, near math.Point3, axis math.Vector3, radius float64) *planeUV {
	seat3D := make([][]math.Point3, len(seatFace.loops))
	for i, lp := range seatFace.loops {
		seat3D[i] = sampleCurvedLoop(lp)
	}
	seatUV := make([][]math.Point2, len(seat3D))
	for i, ring := range seat3D {
		uv := make([]math.Point2, len(ring))
		for j, p := range ring {
			uv[j] = to2D(pl, p)
		}
		seatUV[i] = uv
	}
	inTool := func(p math.Point3) bool {
		return offAxisDistance(near, axis, p) < radius-geom.ResolutionForSize(radius).Weld()
	}
	return &planeUV{plane: pl, seatUV: seatUV, seat3D: seat3D, inTool: inTool, res: geom.ResolutionForSize(2 * radius)}
}

// copyExcept returns all faces but the one at skip (a shallow copy; loops are shared, never mutated).
func copyExcept(faces []curvedFace, skip int) []curvedFace {
	out := make([]curvedFace, 0, len(faces))
	for i, f := range faces {
		if i != skip {
			out = append(out, f)
		}
	}
	return out
}
