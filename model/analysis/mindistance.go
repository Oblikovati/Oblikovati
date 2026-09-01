// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Minimum distance (M18-F01 PBI-164, #428; analytic since M48/C3 #3458): the closest approach between
// two model entities of any kind, computed against the EXACT trimmed B-rep — never a face's triangle
// mesh, whose chord sagitta biased convex-curved clearances inward. Each entity becomes a
// brep.DistSupport (a vertex → a point, an edge → its trimmed curve, a face → its trimmed surface) and
// the kernel's OCCT-style distance recursion measures the pair. Result is millimetres; zero when the
// entities touch or intersect.

// probeOnTol is the boundary band for the probe's interior-collision test: a probe point within this
// gap of a wall still reads as inside, matching OCC distToShape's overlap=0 verdict.
const probeOnTol = 1e-6

// MeasureEntity is a resolved model entity (exactly one of Vertex/Edge/Face) for distance queries.
type MeasureEntity struct {
	Vertex *topo.Vertex
	Edge   *topo.Edge
	Face   *topo.Face
}

// MinDistanceMm returns the minimum distance in millimetres between two entities. The Quality argument
// is retained for call-site compatibility but no longer used now the measure is analytic.
func MinDistanceMm(a, b MeasureEntity, _ ops.Quality) float64 {
	return brep.EntityDistance(a.support(), b.support()) * cmToMM
}

// support maps a resolved model entity onto the kernel's distance operand.
func (e MeasureEntity) support() brep.DistSupport {
	switch {
	case e.Vertex != nil:
		return brep.PointSupport(e.Vertex.Point())
	case e.Edge != nil:
		return brep.CurveSupport(e.Edge.Geometry())
	default:
		return brep.FaceSupport(e.Face)
	}
}

// MinDistanceProbeToBody returns the minimum distance, in database units (cm), between a transient
// probe polyline and a body — the closest approach of an arbitrary travel path (e.g. a CAM linking
// move) to the part, measured against the exact trimmed faces. The probe reduces to its segments (a
// lone point becomes a zero-length segment), each measured against every face by the analytic
// curve→face distance, and a probe point strictly inside the material is a collision (distance 0). It
// is +Inf when the probe or the body has no geometry, and 0 when they touch or intersect. This is the
// transient-operand counterpart of MinDistanceMm — the out-of-process projection of
// MeasureTools.GetMinimumDistance (Oblikovati/Oblikovati#630).
func MinDistanceProbeToBody(probe []math.Point3, body *topo.Body, q ops.Quality) float64 {
	if len(probe) == 0 {
		return stdmath.Inf(1)
	}
	// Distance to a *solid*, not just its skin: a probe point inside the material is a collision even
	// when no segment crosses a boundary face — a fully interior travel move would otherwise report the
	// gap to the nearest wall. A segment that merely passes through is caught below (it grazes a face).
	for _, p := range probe {
		if query.BodyContainment(body, p, q, probeOnTol) == query.ContainInside {
			return 0
		}
	}
	return probeToFacesDistance(probe, body)
}

// probeToFacesDistance is the smallest analytic curve→face distance between the probe's segments and
// the body's faces.
func probeToFacesDistance(probe []math.Point3, body *topo.Body) float64 {
	best := stdmath.Inf(1)
	for _, seg := range probeSegments(probe) {
		curve := brep.CurveSupport(seg)
		for _, f := range body.Faces() {
			best = stdmath.Min(best, brep.EntityDistance(curve, brep.FaceSupport(f)))
		}
	}
	return best
}

// probeSegments turns a probe polyline into geom line segments; a single point becomes one zero-length
// segment so it still measures point-to-body.
func probeSegments(probe []math.Point3) []geom.Curve3 {
	if len(probe) == 1 {
		return []geom.Curve3{geom.NewLineSegment(probe[0], probe[0])}
	}
	segs := make([]geom.Curve3, 0, len(probe)-1)
	for i := 1; i < len(probe); i++ {
		segs = append(segs, geom.NewLineSegment(probe[i-1], probe[i]))
	}
	return segs
}
