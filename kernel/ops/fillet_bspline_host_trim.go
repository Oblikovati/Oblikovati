// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// End termination of the OPEN B-spline-host canal band: the band is built LONG (prolong
// stations past each edge end, fillet_bspline_host_band.go) and TRIMMED where it crosses
// the capping face's plane — OCCT's prolong-then-trim at a restriction
// (BRepBlend_SurfRstLineBuilder.cxx). Crossings are isolated per RAIL by bracketed
// bisection over the station columns (sign-safe; a window with zero or multiple sign
// changes REFUSES — never a guessed branch), the interior trim curve is marched column by
// column in the band's own parameter space, and every landing is SNAPPED onto the host's
// own boundary edge so the loop splice (segParam, weld-scaled) is exact by construction.

// bsplineHostCapPlane is one end's capping-plane data.
type bsplineHostCapPlane struct {
	face   *topo.Face
	plane  geom.Plane
	n      math.Vector3 // unit outward-agnostic plane normal (sign only needs consistency)
	origin math.Point3
}

// bsplineHostEndTrim is one solved end: the two rail-crossing v-params, the SNAPPED
// landing points on the host boundary edges, and the fitted crossing curve pA→pB.
type bsplineHostEndTrim struct {
	cap    bsplineHostCapPlane
	vA, vB float64
	pA, pB math.Point3
	trim   endSeg
}

// signedCapDist is the signed distance of a point to the capping plane.
func (c bsplineHostCapPlane) signedCapDist(p math.Point3) float64 {
	return float64(c.origin.VectorTo(p).Dot(c.n))
}

// newBsplineHostCapPlane resolves one end's capping face, requiring a PLANE capping (the
// B-spline-capped ends — I5/I7-class — are a later tier and decline honestly here).
func newBsplineHostCapPlane(v *topo.Vertex, ef edgeFillet) (bsplineHostCapPlane, string) {
	capping, ok, why := cappingFaceAtFarVertex(v, ef, map[uint64]bool{ef.edge.ID(): true})
	if !ok {
		return bsplineHostCapPlane{}, why
	}
	pl, isPlane := capping.Geometry().(geom.Plane)
	if !isPlane {
		return bsplineHostCapPlane{}, fmt.Sprintf("bspline-host runout: capping face %T at %v is not planar (B-spline-capped ends are not yet supported)", capping.Geometry(), v.Point())
	}
	n, err := math.UnitVector3FromVector(pl.Normal())
	if err != nil {
		return bsplineHostCapPlane{}, "bspline-host runout: degenerate capping plane normal"
	}
	return bsplineHostCapPlane{face: capping, plane: pl, n: n.AsVector(), origin: pl.Origin}, ""
}

// railCapCrossing isolates the crossing of one rail (u fixed at 0 or 1) with the capping
// plane inside the station-index window [i0, i1]: exactly one sign-change interval, then
// bisection to parameter convergence. Refuses zero (overrun too short / no crossing) and
// multiple (grazing cap) crossings, naming the count.
func railCapCrossing(surf geom.BSplineSurface, u float64, vp []float64, i0, i1 int, cap bsplineHostCapPlane) (float64, math.Point3, string) {
	f := func(v float64) float64 { return cap.signedCapDist(surf.PointAt(u, v)) }
	lo, hi, count := signChangeWindow(f, vp, i0, i1)
	if count != 1 {
		return 0, math.Point3{}, fmt.Sprintf("bspline-host runout: rail u=%g has %d cap-plane crossings in the end window (need exactly 1)", u, count)
	}
	v := bisectRoot(f, lo, hi)
	return v, surf.PointAt(u, v), ""
}

// signChangeWindow scans consecutive station columns for sign changes of f, returning the
// first bracketing interval and the total count found.
func signChangeWindow(f func(float64) float64, vp []float64, i0, i1 int) (lo, hi float64, count int) {
	prev := f(vp[i0])
	for i := i0 + 1; i <= i1; i++ {
		cur := f(vp[i])
		if (prev < 0) != (cur < 0) {
			if count == 0 {
				lo, hi = vp[i-1], vp[i]
			}
			count++
		}
		prev = cur
	}
	return lo, hi, count
}

// bisectRoot drives f to a root inside a bracketing interval (60 halvings: parameter
// resolution ~1e-18 of the interval — beyond float64, i.e. converged).
func bisectRoot(f func(float64) float64, lo, hi float64) float64 {
	flo := f(lo)
	for range 60 {
		mid := (lo + hi) / 2
		if fm := f(mid); (fm < 0) == (flo < 0) {
			lo, flo = mid, fm
		} else {
			hi = mid
		}
	}
	return (lo + hi) / 2
}

// snapLandingToHostEdge projects a rail-crossing point onto the host face's own boundary
// edge nearest it (excluding the picked edge): the landing the loop splice consumes must
// lie ON the host loop exactly, and the crossing is only within the band's envelope bound
// of it. snapTol bounds the snap distance — a snap past it means the crossing landed on
// the wrong feature and refuses.
func snapLandingToHostEdge(host *topo.Face, picked *topo.Edge, q math.Point3, snapTol float64) (math.Point3, string) {
	best := math.Point3{}
	bestD := stdmath.Inf(1)
	for _, e := range host.Edges() {
		if e == picked {
			continue
		}
		if foot, d, ok := footOnEdgeCurve(e, q); ok && d < bestD {
			best, bestD = foot, d
		}
	}
	if bestD > snapTol {
		return math.Point3{}, fmt.Sprintf("bspline-host runout: crossing %v is %g off the host boundary (snap tolerance %g)", q, bestD, snapTol)
	}
	return best, ""
}

// footOnEdgeCurve is the closest point on one edge's curve to q (domain-clamped).
func footOnEdgeCurve(e *topo.Edge, q math.Point3) (math.Point3, float64, bool) {
	c := e.Geometry()
	if c == nil {
		return chordFoot(e, q)
	}
	t, _ := geom.CurveParamAtPoint3(c, q)
	lo, hi := c.Domain()
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
		return chordFoot(e, q)
	}
	t = stdmath.Min(hi, stdmath.Max(lo, t))
	foot := c.PointAt(t)
	return foot, float64(foot.DistanceTo(q)), true
}

// chordFoot projects q onto the edge's vertex chord — the fallback for a nil/unbounded
// edge curve (a straight edge carries no curve object).
func chordFoot(e *topo.Edge, q math.Point3) (math.Point3, float64, bool) {
	a, b := e.StartVertex().Point(), e.EndVertex().Point()
	ab := a.VectorTo(b)
	den := float64(ab.LengthSquared())
	if den == 0 {
		return a, float64(a.DistanceTo(q)), true
	}
	t := stdmath.Min(1, stdmath.Max(0, float64(a.VectorTo(q).Dot(ab))/den))
	foot := a.TranslateBy(ab.Scale(math.Scalar(t)))
	return foot, float64(foot.DistanceTo(q)), true
}

// bsplineHostTrimSamplesMin/Max bound the trim-curve sampling refinement.
const (
	bsplineHostTrimSamplesMin = 32
	bsplineHostTrimSamplesMax = 128
)

// bsplineHostTrimCurve marches the band∩cap-plane curve across the band (one v-root per
// u-column, bracketed inside the end window), snaps its endpoints to the two landings and
// fits the interpolating curve, refining the sampling until the fit is inside the envelope
// bound. Orientation: pA (host-A side) → pB.
func bsplineHostTrimCurve(surf geom.BSplineSurface, end bsplineHostEndTrim, vp []float64, i0, i1 int, bound float64) (endSeg, string) {
	for k := bsplineHostTrimSamplesMin; k <= bsplineHostTrimSamplesMax; k *= 2 {
		pts, reason := trimCurveSamples(surf, end, vp, i0, i1, k)
		if reason != "" {
			return endSeg{}, reason
		}
		fitted, err := geom.NewFittedBSplineCurve(pts)
		if err != nil {
			return endSeg{}, fmt.Sprintf("bspline-host runout: trim fit failed: %v", err)
		}
		if trimFitError(fitted, surf, end, bound) <= bound {
			return endSeg{from: end.pA, to: end.pB, curve: fitted}, ""
		}
	}
	return endSeg{}, "bspline-host runout: trim curve fit error over bound at the sampling cap"
}

// trimCurveSamples solves the per-column v-roots and packages the sample run pA→…→pB.
func trimCurveSamples(surf geom.BSplineSurface, end bsplineHostEndTrim, vp []float64, i0, i1, k int) ([]math.Point3, string) {
	pts := make([]math.Point3, 0, k+1)
	pts = append(pts, end.pA)
	for i := 1; i < k; i++ {
		u := float64(i) / float64(k)
		f := func(v float64) float64 { return end.cap.signedCapDist(surf.PointAt(u, v)) }
		lo, hi, count := signChangeWindow(f, vp, i0, i1)
		if count != 1 {
			return nil, fmt.Sprintf("bspline-host runout: trim column u=%g has %d cap crossings (need exactly 1)", u, count)
		}
		pts = append(pts, surf.PointAt(u, bisectRoot(f, lo, hi)))
	}
	return append(pts, end.pB), ""
}

// trimFitError measures the fitted trim between its samples: mid-parameter points must sit
// on the cap plane and on the band within the envelope bound.
func trimFitError(fitted geom.BSplineCurve, surf geom.BSplineSurface, end bsplineHostEndTrim, bound float64) float64 {
	lo, hi := fitted.Domain()
	worst := 0.0
	for i := range 16 {
		q := fitted.PointAt(lo + (hi-lo)*(float64(i)+0.5)/16)
		worst = stdmath.Max(worst, stdmath.Abs(end.cap.signedCapDist(q)))
		worst = stdmath.Max(worst, distToSurface(surf, q))
		if worst > bound {
			return worst // early out: one violation already decides
		}
	}
	return worst
}
