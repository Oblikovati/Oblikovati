// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"errors"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Curved half-space cut (M2 Phase 1, Oblikovati/Oblikovati#1334). The first operation of the
// general curved boolean: trim an analytic curved solid by ONE plane, keeping the part on the
// plane's negative side and closing it with a planar lid — an EXACT curved B-rep (the kept curved
// faces preserve their analytic surface, not a tessellated soup). It is the building block the box
// boolean composes (a convex planar tool is an intersection of half-spaces) and the natural place
// the imprint→split→classify→stitch pipeline starts, restricted to a single planar cutter so
// classification is just "which side of the plane" (no point-in-solid needed yet).

// ErrUnsupportedHalfSpace reports a body the half-space cut does not yet handle (a surface type or
// trim configuration beyond the analytic primitives wired so far) — the caller keeps the BSP-CSG
// fallback, so there is no regression.
var ErrUnsupportedHalfSpace = errors.New("brep: half-space cut handles only the wired analytic curved solids")

// HalfSpaceCut returns the part of body on the NEGATIVE side of plane — the half-space
// {x : normal·(x − origin) ≤ 0} — capped by a planar lid whose outward normal is +plane.Normal.
// The result is an exact curved B-rep. A plane that misses the body returns the whole body (all
// negative) or an empty body (all positive). ErrUnsupportedHalfSpace for an as-yet-unwired solid.
//
// Example — a sphere cut by z=0 keeps the lower hemisphere (a true spherical cap + a disk lid):
//
//	sphere, _ := brep.SolidSphere(math.P3(0,0,0), 5, "s")
//	plane, _ := geom.NewPlane(math.P3(0,0,0), math.V3(0,0,1))
//	cap, _ := brep.HalfSpaceCut(sphere, plane) // lower hemisphere
func HalfSpaceCut(body *topo.Body, plane geom.Plane) (*topo.Body, error) {
	faces := facesOfAny(body)
	if len(faces) == 1 {
		if sph, ok := faces[0].surface.(geom.Sphere); ok && len(faces[0].loops) == 0 {
			return sphereHalfSpace(body, sph, plane, faces[0].lineage)
		}
	}
	if cyl, base, height, ok := cylinderSolidParams(faces); ok {
		return cylinderHalfSpace(body, cyl, base, height, plane)
	}
	return nil, ErrUnsupportedHalfSpace
}

// sphereHalfSpace cuts a bare sphere by a plane: the part on the plane's negative side is a
// spherical cap closed by a circular lid. The cut circle is the analytic imprint (curvedImprint);
// when the plane clears the sphere the cap is the whole sphere (centre below) or empty (above).
func sphereHalfSpace(body *topo.Body, sph geom.Sphere, plane geom.Plane, lineage topo.Lineage) (*topo.Body, error) {
	n := unit(plane.Normal())
	curves, _ := curvedImprint(curvedFace{surface: sph}, curvedFace{surface: plane})
	if len(curves) == 0 {
		if float64(plane.Origin.VectorTo(sph.Center).Dot(n)) <= 0 {
			return body, nil // centre on the negative side → the whole sphere is kept
		}
		return topo.MergeBodies(topo.NewLineage(topo.Tok("halfspace", "empty", 0)), true), nil
	}
	circle, ok := curves[0].(geom.Circle)
	if !ok {
		return nil, ErrUnsupportedHalfSpace
	}
	return assembleSphereCap(sph, circle, n, lineage)
}

// assembleSphereCap builds the spherical cap (kept side) and its planar disk lid, sharing the cut
// circle as one edge. The cap face carries the sphere's lineage (its reference key survives); the
// sphere face's loop is the circle reversed so the kept −n cap is enclosed, and the lid's outward
// normal is +n (it faces away from the kept material on the −n side).
func assembleSphereCap(sph geom.Sphere, circle geom.Circle, n math.Vector3, lineage topo.Lineage) (*topo.Body, error) {
	lid, err := geom.NewPlane(circle.Center, n)
	if err != nil {
		return nil, err
	}
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("halfspace", "body", 0)))
	v := bld.AddVertex(circle.PointAt(0), topo.NewLineage(topo.Tok("halfspace", "seam", 0)))
	edge := bld.AddEdge(circle, v, v, topo.NewLineage(topo.Tok("halfspace", "cut", 0)))
	bld.AddFace(lid, topo.NewLineage(topo.Tok("halfspace", "lid", 0)), topo.OuterLoop(topo.Fwd(edge)))
	bld.AddFace(sph, lineage, topo.OuterLoop(topo.Rev(edge))) // cap keeps the sphere's key (K1a/K1b)
	return bld.Build(), nil
}
