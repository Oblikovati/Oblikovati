// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Analytic face constructors shared by the bespoke curved-boolean assemblers — the handlers that build
// their result directly instead of running the (u,v) arrangement (the boss, the coaxial ball stud).
// Each returns ONE curvedFace with its boundary already wound to the side that survives, because for
// these KINDs naming the surviving region IS the split (ADR-0045).

// cylinderBandFace builds a cylindrical wall between two coaxial circles as one seamed periodic loop —
// seam up, far circle reversed, seam down, near circle forward — the same loop SolidCylinder's side
// carries, so it welds to whatever bounds it at either rim. near and far must share a frame (see
// coaxialCircleAt) with near the lower one along cyl.AxisDir; reversed turns the wall inward, which is
// what a cut's bore needs.
func cylinderBandFace(cyl geom.Cylinder, near, far geom.Circle, reversed bool, lin topo.Lineage) curvedFace {
	seam := geom.NewLineSegment(near.PointAt(0), far.PointAt(0))
	loop := curvedLoop{edges: []loopEdge{
		{curve: seam, t0: 0, t1: 1},
		{curve: far, t0: 1, t1: 0},
		{curve: seam, t0: 1, t1: 0},
		{curve: near, t0: 0, t1: 1},
	}}
	return curvedFace{surface: cyl, reversed: reversed, loops: []curvedLoop{loop}, lineage: lin}
}

// discFace builds the planar disc that a circle bounds. forward is the direction the loop must walk
// the rim — the caller's neighbour on that same edge walks it the other way, which is the invariant
// ops.Validate checks — and the plane's normal is taken to MATCH that walk, so the loop stays
// counter-clockwise about its own surface normal. Which side holds material is then carried by the
// face sense alone: reversed when outward disagrees with that normal.
func discFace(rim geom.Circle, forward bool, outward math.Vector3, lin topo.Lineage) (curvedFace, bool) {
	normal := rim.Normal.AsVector()
	if !forward {
		normal = normal.Scale(-1)
	}
	pl, err := geom.NewPlane(rim.Center, normal)
	if err != nil {
		return curvedFace{}, false
	}
	return curvedFace{surface: pl, reversed: normal.Dot(outward) < 0, lineage: lin,
		loops: []curvedLoop{{edges: []loopEdge{orientedCircle(rim, normal)}}}}, true
}

// sphereCapFace builds the sphere face a circle on that sphere bounds, keeping the cap on the keep side
// of the circle's plane. A circle splits a sphere into exactly two caps and the WINDING is what names
// which one: kernel/ops reads the loop's direction against the outward radius to find the enclosed pole
// (capAxis in sphere_cap_mesh.go), so a forward loop keeps the cap the circle's normal points into.
// That makes this the one face in an assembly whose winding is NOT free — the neighbours have to take
// the opposite direction from it, not the other way round. reversed additionally turns the cap inward —
// a spherical dimple, where the material is outside the ball rather than inside it.
func sphereCapFace(sph geom.Sphere, rim geom.Circle, keep math.Vector3, reversed bool, lin topo.Lineage) curvedFace {
	return curvedFace{surface: sph, reversed: reversed, lineage: lin,
		loops: []curvedLoop{{edges: []loopEdge{orientedCircle(rim, keep)}}}}
}

// sphereBeltFace builds the sphere face BETWEEN two coaxial circles on it — the belt a rod passing right
// through a ball leaves. Unlike a cap it needs no winding to name its region: two distinct coaxial
// circles bound exactly one connected region, since the complement is two disjoint caps and those cannot
// be one face. So the directions here are fixed by convention (outer forwards, inner backwards) and it
// is the neighbours sharing each rim that adapt. inverted turns the belt inward.
func sphereBeltFace(sph geom.Sphere, outer, inner geom.Circle, inverted bool, lin topo.Lineage) curvedFace {
	return curvedFace{surface: sph, reversed: inverted, lineage: lin, loops: []curvedLoop{
		{edges: []loopEdge{{curve: outer, t0: 0, t1: 1}}},
		{edges: []loopEdge{{curve: inner, t0: 1, t1: 0}}},
	}}
}

// orientedCircle walks a circle forward when its normal agrees with want, backward otherwise.
func orientedCircle(c geom.Circle, want math.Vector3) loopEdge {
	if c.Normal.AsVector().Dot(want) < 0 {
		return loopEdge{curve: c, t0: 1, t1: 0}
	}
	return loopEdge{curve: c, t0: 0, t1: 1}
}
