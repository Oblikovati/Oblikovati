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
	return withClosedSurfaceComplement(curvedFace{surface: sph, reversed: reversed, lineage: lin,
		loops: []curvedLoop{{edges: []loopEdge{orientedCircle(rim, keep)}}}})
}

// withClosedSurfaceComplement records, on a face of a CLOSED surface, whether its loops bound a HOLE
// rather than the face itself — the [curvedFace.outerless] case, which curvedStitch then mints as an
// inner loop and every later reader recovers from topo.Loop.IsOuter.
//
// It reads the loops' handedness in (u, v), the same rule curved_halfspace_torus_uv.go applies to the
// torus complement. Reading it HERE is sound and reading it at classification time is not: this builder
// wound these loops itself, one line above, whereas a classifier handed an arbitrary body is reading a
// datum that orients a shell only up to one global sign (orient_consistent.go fixes that sign from the
// body's signed volume). Recording it at the source is what lets the trim classifier stay orientation-
// free (Oblikovati/Oblikovati#3477).
func withClosedSurfaceComplement(cf curvedFace) curvedFace {
	cf.outerless = closedSurfaceRingsAreHoles(cf)
	return cf
}

// closedSurfaceRingsAreHoles reports whether the face's freshly wound loops enclose everything BUT the
// face, which only a closed parameter domain (a sphere, a torus — no axis reaching an exterior) admits.
func closedSurfaceRingsAreHoles(cf curvedFace) bool {
	uPer, vPer := surfacePeriodic(cf.surface)
	if _, open := castAxis(cf.surface, uPer, vPer); open || len(cf.loops) == 0 {
		return false
	}
	area := 0.0
	for _, loop := range cf.loops {
		area += signedArea2D(loopToUV(cf.surface, loop, uPer, vPer))
	}
	return area < 0
}

// annulusFace builds the planar ring between two coaxial circles in one plane — what is left of a rod's
// end cap when the ball's own surface crosses it. outerForward is the direction the loop must walk the
// outer rim (its neighbour on that rim walks the other way); the plane's normal follows that walk, the
// hole is wound against it as a hole must be, and the face sense places the material.
func annulusFace(outer, inner geom.Circle, outerForward bool, outward math.Vector3, lin topo.Lineage) (curvedFace, bool) {
	f, ok := discFace(outer, outerForward, outward, lin)
	if !ok {
		return curvedFace{}, false
	}
	hole := orientedCircle(inner, outer.Normal.AsVector().Scale(-1))
	if !outerForward {
		hole = orientedCircle(inner, outer.Normal.AsVector())
	}
	f.loops = append(f.loops, curvedLoop{edges: []loopEdge{hole}})
	return f, true
}

// sphereBeltFace builds the sphere face BETWEEN two coaxial circles on it — the belt a rod passing right
// through a ball leaves, the shoulder ring one stopping part way through it does. Both rims carry the
// same normal (coaxialRod.circleAt builds every rim with the axis normal), and `lo` is the rim the belt
// lies on the +normal side OF.
//
// Its winding NAMES its region exactly as a cap's does, so the directions here are NOT free. A loop
// walked forward — counter-clockwise about the circle's own normal — encloses the +normal side, so the
// belt is named by walking `lo` forward (the belt is above it) and `hi` backward, as a hole. Walked the
// other way round, the same two circles name the COMPLEMENT: on a sphere that is the two disjoint caps,
// and every reader of the trim — brep's geodesic winding, ops' analytic region integral, the
// tessellator — then measures the wrong region (Oblikovati/Oblikovati#3447). inverted turns the belt's
// material side inward; it moves no loop, because a loop is wound about the SURFACE normal, not the
// face sense.
func sphereBeltFace(sph geom.Sphere, lo, hi geom.Circle, inverted bool, lin topo.Lineage) curvedFace {
	return withClosedSurfaceComplement(curvedFace{surface: sph, reversed: inverted, lineage: lin,
		loops: []curvedLoop{
			{edges: []loopEdge{{curve: lo, t0: 0, t1: 1}}},
			{edges: []loopEdge{{curve: hi, t0: 1, t1: 0}}},
		}})
}

// orientedCircle walks a circle forward when its normal agrees with want, backward otherwise.
func orientedCircle(c geom.Circle, want math.Vector3) loopEdge {
	if c.Normal.AsVector().Dot(want) < 0 {
		return loopEdge{curve: c, t0: 1, t1: 0}
	}
	return loopEdge{curve: c, t0: 0, t1: 1}
}
