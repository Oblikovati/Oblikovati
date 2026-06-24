// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Curved convex subtract — axial prism tunnel (M2 Phase 1, Oblikovati/Oblikovati#1334). A Cut is not a
// half-space composition: cylinder − box keeps the cylinder OUTSIDE the box plus the box walls INSIDE the
// cylinder, flipped to bound the cavity. This file handles the clean through-tunnel along the axis: a
// convex prism (the box) that passes through both caps and stays inside the cylinder radius. The side is
// untouched, each cap gains a polygonal hole (a planar face with an inner loop), and the prism's walls
// become the tube — no curved face gets a hole, so it stays within what tessellation already supports.

// SubtractAxialPrism returns the cylinder target with the convex prism through-hole whose cross-section,
// at the base cap, is polyBottom (ordered CCW about the axis, inside the disk). The caller guarantees the
// prism spans both caps and clears the side; ErrUnsupportedHalfSpace if target is not a bare cylinder or
// a corner falls outside the disk (a side-breaching tunnel — a later increment).
//
// Example — a 2×2 square hole down the axis of a radius-3 cylinder:
//
//	cyl, _ := brep.SolidCylinder(math.P3(0,0,0), math.V3(0,0,1), 3, 10)
//	sq := []math.Point3{ {1,1,0},{-1,1,0},{-1,-1,0},{1,-1,0} } // CCW about +z, at the base
//	holed, _ := brep.SubtractAxialPrism(cyl, sq)
func SubtractAxialPrism(target *topo.Body, polyBottom []math.Point3) (*topo.Body, error) {
	cyl, base, height, ok := cylinderSolidParams(facesOfAny(target))
	if !ok || len(polyBottom) < 3 {
		return nil, ErrUnsupportedHalfSpace
	}
	axis := cyl.AxisDir.AsVector()
	for _, c := range polyBottom {
		if float64(radialPart(base.VectorTo(c), axis).Length()) >= cyl.Radius-1e-9 {
			return nil, ErrUnsupportedHalfSpace // a corner reaches the side: not a clean axial tunnel
		}
	}
	polyTop := translateAll(polyBottom, axis.Scale(math.Scalar(height)))
	faces := prismHoledCaps(facesOfAny(target), base, polyBottom, polyTop)
	faces = append(faces, prismTubeWalls(polyBottom, polyTop)...)
	return curvedStitch(faces), nil
}

// prismHoledCaps keeps the cylinder side whole and gives each planar cap a polygonal hole loop (the prism
// cross-section at that cap's level), so the cap becomes an annulus-like face the hole tunnels through.
func prismHoledCaps(faces []curvedFace, base math.Point3, polyBottom, polyTop []math.Point3) []curvedFace {
	out := make([]curvedFace, 0, len(faces))
	for _, f := range faces {
		if _, isCyl := f.surface.(geom.Cylinder); isCyl {
			out = append(out, f) // the side never meets the axial tunnel: untouched
			continue
		}
		normal := capOutwardNormal(f)
		poly := polyBottom
		if float64(base.VectorTo(capOrigin(f)).Dot(normal)) > 0 {
			poly = polyTop // this cap's outward normal points away from the base → it is the top cap
		}
		f.loops = append(f.loops, holeLoop(poly, normal))
		out = append(out, f)
	}
	return out
}

// prismTubeWalls builds the cavity walls: one planar rectangle per cross-section edge, spanning base to
// top, with its outward normal pointing into the cylinder material (away from the prism interior) so it
// bounds the removed tunnel.
func prismTubeWalls(polyBottom, polyTop []math.Point3) []curvedFace {
	n := len(polyBottom)
	centroid := polygonCentroid(polyBottom)
	walls := make([]curvedFace, 0, n)
	for i := range polyBottom {
		j := (i + 1) % n
		bi, bj, tj, ti := polyBottom[i], polyBottom[j], polyTop[j], polyTop[i]
		outward := wallOutwardNormal(bi, bj, ti, centroid)
		plane, err := geom.NewPlane(bi, outward)
		if err != nil {
			continue
		}
		walls = append(walls, curvedFace{
			surface: plane,
			loops:   []curvedLoop{orientLoop([]math.Point3{bi, bj, tj, ti}, outward, +1)},
			lineage: topo.NewLineage(topo.Tok("subtract", "wall", i)),
		})
	}
	return walls
}

// capOutwardNormal returns a planar cap's outward normal (its plane normal, flipped when the face is a
// reversed use of that plane).
func capOutwardNormal(f curvedFace) math.Vector3 {
	n := f.surface.(geom.Plane).NormalAt(0, 0)
	if f.reversed {
		return n.Scale(-1)
	}
	return n
}

// capOrigin returns a planar cap's plane origin.
func capOrigin(f curvedFace) math.Point3 {
	return f.surface.(geom.Plane).Origin
}

// holeLoop builds the polygon as an inner (hole) loop of a cap: wound CW about the cap's outward normal,
// the opposite sense to the outer circle, so it removes material.
func holeLoop(poly []math.Point3, normal math.Vector3) curvedLoop {
	return orientLoop(poly, normal, -1)
}

// orientLoop builds a closed polygon loop (line segments between consecutive corners) wound so its signed
// area about normal has the requested sign (+1 CCW for an outer/wall loop, −1 CW for a hole).
func orientLoop(poly []math.Point3, normal math.Vector3, want float64) curvedLoop {
	pts := append([]math.Point3(nil), poly...)
	if signedLoopArea(pts, normal)*want < 0 {
		reverse3(pts)
	}
	edges := make([]loopEdge, len(pts))
	for i := range pts {
		j := (i + 1) % len(pts)
		edges[i] = loopEdge{curve: geom.NewLineSegment(pts[i], pts[j]), t0: 0, t1: 1}
	}
	return curvedLoop{edges: edges}
}

// wallOutwardNormal returns the unit normal of a tube wall (the plane through edge bi→bj and the axial
// direction ti−bi) pointing INTO the prism interior (toward the centroid). The solid is the cylinder
// material OUTSIDE the prism, so the wall's outward face normal points away from that material — into the
// removed tunnel — which is toward the cross-section centroid.
func wallOutwardNormal(bi, bj, ti, centroid math.Point3) math.Vector3 {
	n := bi.VectorTo(bj).Cross(bi.VectorTo(ti))
	if float64(centroid.VectorTo(bi).Dot(n)) > 0 {
		n = n.Scale(-1) // pointing away from the centroid → flip to point into the prism
	}
	return n
}

// signedLoopArea is twice the polygon area projected on normal, signed +CCW about it (Newell's method).
func signedLoopArea(pts []math.Point3, normal math.Vector3) float64 {
	var sum math.Vector3
	for i := range pts {
		j := (i + 1) % len(pts)
		sum = sum.Add(pts[i].AsVector().Cross(pts[j].AsVector()))
	}
	return float64(sum.Dot(normal))
}

// polygonCentroid returns the average of the polygon corners.
func polygonCentroid(pts []math.Point3) math.Point3 {
	var v math.Vector3
	for _, p := range pts {
		v = v.Add(p.AsVector())
	}
	return math.P3(0, 0, 0).TranslateBy(v.Scale(math.Scalar(1 / float64(len(pts)))))
}

// radialPart returns the component of v perpendicular to axis (its distance from the axis line).
func radialPart(v, axis math.Vector3) math.Vector3 {
	return v.Sub(axis.Scale(v.Dot(axis)))
}

// translateAll offsets every point by delta.
func translateAll(pts []math.Point3, delta math.Vector3) []math.Point3 {
	out := make([]math.Point3, len(pts))
	for i, p := range pts {
		out[i] = p.TranslateBy(delta)
	}
	return out
}

// reverse3 reverses a point slice in place.
func reverse3(pts []math.Point3) {
	for i, j := 0, len(pts)-1; i < j; i, j = i+1, j-1 {
		pts[i], pts[j] = pts[j], pts[i]
	}
}
