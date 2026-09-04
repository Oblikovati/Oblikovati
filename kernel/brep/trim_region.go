// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A face's region in its surface's (u, v) domain, as the flux quadrature and the orientation probe both
// have to read it.
//
// It is the face's CHART: closed contours in the covering space, material on the left (face_chart.go,
// ADR-0063). A closed contour needs no rule about which side of it the face is, because it has an
// inside — which is the whole point of carrying it. What used to live here instead was four rules that
// tried to recover that from open polylines: an azimuth-wrapping rim read by an upward ray, a
// tube-wrapping ring read by the nearest rim above and its material side, a choice between the two
// windows two rim levels bound on a periodic axis, and the outerless complement flag. Each was correct
// for the case it was written for and wrong for the next one (Oblikovati#3453, #3429, #3506, ADR-0062).
type trimRegion struct {
	contours [][]math.Point2
	// decided is false when the face carries no chart and its loops do not determine one. The readers
	// then decline (fluxDomain returns ok=false, so a shell is left uncertified) rather than picking a
	// side, which is what the deleted rules did (ADR-0063).
	decided bool
	// uPeriodic/vPeriodic say whether the surface's own parameter wraps, which is what decides whether
	// a query has to be taken into the chart's branch before it is tested.
	uPeriodic, vPeriodic bool
	// index buckets the contours' segments by v so a query scans only those a +u ray can cross. nil for
	// a chart short enough to scan whole (face_chart_index.go).
	index *chartIndex
}

// contains is the region's own point membership in (u, v). A boundaryless face (a whole sphere/torus)
// owns its entire domain.
func (r trimRegion) contains(q math.Point2) bool {
	if len(r.contours) == 0 {
		return true
	}
	return chartContainsIndexed(r.contours, r.index, q, r.uPeriodic, r.vPeriodic)
}

// faceTrimRegion takes the face's chart once — deriving it costs a ParamAt per loop sample, so this
// must not run per query.
func faceTrimRegion(f curvedFace) trimRegion {
	uPer, vPer := surfacePeriodic(f.surface)
	contours, ok := faceChart(f, uPer, vPer)
	return trimRegion{contours: contours, decided: ok, uPeriodic: uPer, vPeriodic: vPer,
		index: newChartIndex(contours)}
}

// trimPolys projects every loop of the face into one continuous (u, v) polyline (reusing loopToUV's seam
// unwrapping). A boundaryless face returns nil — its whole finite domain is integrated.
func trimPolys(f curvedFace, uPer, vPer bool) [][]math.Point2 {
	if len(f.loops) == 0 {
		return nil
	}
	polys := make([][]math.Point2, 0, len(f.loops))
	for _, loop := range f.loops {
		if poly := loopToUV(f.surface, loop, uPer, vPer); len(poly) >= 3 {
			polys = append(polys, poly)
		}
	}
	return polys
}

// fluxDomain is the (u, v) rectangle the quadrature covers: the chart's own bounding box, and the
// surface's finite domain for a boundaryless face. It fails (ok=false) only for a face whose domain is
// unbounded and which carries no chart, which a closed body never has.
//
// The chart's box is the right window because a chart BOUNDS the material: there is no longer a second
// window on the far side of a periodic axis to choose between, which is what periodicWindowHoldingMaterial
// existed to do — a torus half whose material lay through the seam used to sample its whole quadrature
// over the other half (ADR-0062).
func fluxDomain(f curvedFace, r trimRegion) (u0, u1, v0, v1 float64, ok bool) {
	if !r.decided {
		return 0, 0, 0, 0, false
	}
	if len(r.contours) > 0 {
		if u0, u1, v0, v1, ok = polyBounds(r.contours); ok {
			return u0, u1, v0, v1, true
		}
	}
	return surfaceDomainRect(f.surface, r)
}

// surfaceDomainRect is the surface's whole finite domain, with each PERIODIC axis re-centred on the
// chart's own branch: a contour lives on whichever turn the arrangement recorded it, which the canonical
// [0, 2π] window need not contain, and the quadrature's containment test reads only that branch.
func surfaceDomainRect(s geom.Surface, r trimRegion) (u0, u1, v0, v1 float64, ok bool) {
	centre := chartBranchCentre(r)
	uPer, vPer := surfacePeriodic(s)
	ul, uh := s.UDomain()
	vl, vh := s.VDomain()
	u0, u1 = axisWindow(ul, uh, uPer, centre.X)
	v0, v1 = axisWindow(vl, vh, vPer, centre.Y)
	return u0, u1, v0, v1, isFiniteRect(u0, u1, v0, v1)
}

// axisWindow is one axis of that rectangle: a full turn centred on the chart's branch when the axis is
// periodic, else the surface's own domain.
func axisWindow(lo, hi float64, periodic bool, centre float64) (float64, float64) {
	if !periodic {
		return lo, hi
	}
	return centre - stdmath.Pi, centre + stdmath.Pi
}

// chartBranchCentre is the (u, v) centroid of the first contour — the branch the periodic axes are
// centred on. A boundaryless face has none to place, and every full turn is the same window there, so
// the origin serves.
func chartBranchCentre(r trimRegion) math.Point2 {
	if len(r.contours) == 0 {
		return math.P2(0, 0)
	}
	return loopCentroid(r.contours[0])
}

// isFiniteRect reports whether the rectangle is bounded and non-degenerate, the precondition of the
// quadrature.
func isFiniteRect(u0, u1, v0, v1 float64) bool {
	if stdmath.IsInf(u0, 0) || stdmath.IsInf(u1, 0) || stdmath.IsInf(v0, 0) || stdmath.IsInf(v1, 0) {
		return false
	}
	return u1 > u0 && v1 > v0
}

// boundaryDistance is the (u,v) distance from q to the nearest contour segment — how far inside its trim
// a point sits, for choosing a probe point clear of the boundary.
func (r trimRegion) boundaryDistance(q math.Point2) float64 {
	best := stdmath.Inf(1)
	for _, contour := range r.contours {
		for i := range contour {
			best = stdmath.Min(best, pointSegmentDistance2D(q, contour[i], contour[(i+1)%len(contour)]))
		}
	}
	return best
}

// pointSegmentDistance2D is the distance from q to the segment a→b.
func pointSegmentDistance2D(q, a, b math.Point2) float64 {
	ab := a.VectorTo(b)
	l2 := float64(ab.Dot(ab))
	t := 0.0
	if l2 > 0 {
		t = stdmath.Max(0, stdmath.Min(1, float64(a.VectorTo(q).Dot(ab))/l2))
	}
	return float64(q.DistanceTo(a.TranslateBy(ab.Scale(math.Scalar(t)))))
}
