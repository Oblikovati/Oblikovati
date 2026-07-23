// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// setbackSection constructs one of the interior seam rails of the 4 coons4 corner-blend panels: the
// non-analytic cross-section where two neighbouring dual-host panels meet at axis-station z (derivation
// §2.3, method A; the CRUX rail of the U4 multi-rail build, #2007 Group C). Chicken-and-egg: the seam
// rail IS a section of the setback surface being built, so it cannot be read off an existing analytic
// face — it is constructed from the two bosses' rim points at the station.
//
// The load-bearing geometric fact (validated against the DRAWEXE oracle at z=±6.240 and z=0, and an
// interior station z=3): OCCT's real section curve is, to its own BSpline-vs-circle sampling floor
// (~6e-7), an EXACT circular arc of the fillet's rolling-ball radius ef.cyl.Radius (=5) — the rolling
// ball's radius is preserved in every axis-perpendicular section, only its centre migrates as the ball
// setbacks against the two bosses. So method A is not an arbitrary 3-point arc through a heuristic
// bulge; it is the radius-r arc through the two boss-rim endpoints, apex on the outward side
// (radiusArcApex). Reproducing OCCT this way costs ~4e-7 (the oracle's own floor) vs ~4e-3 for the
// naive fillet-45°-point bulge — four orders of magnitude, the whole reason this rail is faithful.
//
// pA is host A's contribution at z (its dip-rim point where A is active, else the fillet cylinder's
// A-tangent point on the still-unmodified sliver end); pB is host B's dip-rim point at z. Endpoints are
// the exact rim points at the station, so a neighbour panel that ends its A-rim / B-rim rail at the
// same station welds to this rail bit-for-bit (the corner-weld invariant, ADR-0042). ok=false when a
// rim carries no crossing at the station (a station outside the boss's own active band — the caller
// must not ask for a section a boss does not reach).
//
// Example: setbackSection(0, dets, ef, res) returns the z=0 core mid-panel seam — a radius-5 arc from
// host A's rim point (8,-20,0) through the outward apex to host B's rim point (10,-17,0).
func setbackSection(zStation float64, dets []obstacleDetection, ef edgeFillet, res Resolution) (geom.Curve3, bool) {
	detA, detB, ok := hostDetections(dets)
	if !ok {
		return nil, false
	}
	pA := sectionEndA(detA, ef, zStation)
	pB, okB := dipRimPointAtStation(detB, ef, zStation)
	if !okB {
		return nil, false
	}
	if pA.DistanceTo(pB) < res.Weld() { // the two rim points coincide: a pinched, zero-width section
		return nil, false
	}
	return radiusArcRail(pA, pB, ef, zStation), true
}

// hostDetections splits a dual-host detection pair into (host A, host B), keyed on hostIsA. ok=false
// unless exactly one A and one B are present — setbackSection is only defined for the qualifying==2
// dual-host case (derivation §3.1), never a lone or triplicated host.
func hostDetections(dets []obstacleDetection) (detA, detB obstacleDetection, ok bool) {
	var haveA, haveB bool
	for _, d := range dets {
		if d.hostIsA && !haveA {
			detA, haveA = d, true
		} else if !d.hostIsA && !haveB {
			detB, haveB = d, true
		}
	}
	return detA, detB, haveA && haveB
}

// sectionEndA returns host A's rail endpoint at station z: its dip-rim point where host A dips PAST the
// A-tangent boundary (its footprint sets the fillet back — the core span), else — over a B-only sliver
// span — the fillet cylinder's A-tangent point at z, since there the A-side is still the unmodified
// rolling-ball wall (derivation §1.3). Whether the rim is active is decided geometrically, not by the
// approximate sampled-node interval: the exact rim point is on the fillet side only when it lies past
// the A-tangent in the into-band direction (the c0 wall radial), so the classification is exact at the
// A-node station itself — where the sampled node interval is a sagitta-width off and would misjudge it.
// The A-tangent point is the axis point at z pushed out by c0's constant host-A radial (cornerRadials),
// the same radial the wing sections use. It always yields a point (the A-tangent is the guaranteed
// fallback), so there is no failure mode to report.
func sectionEndA(detA obstacleDetection, ef edgeFillet, z float64) math.Point3 {
	hostRadial, wallRadial, _ := cornerRadials(ef, true)
	aTangent := filletAxisAt(ef, z).TranslateBy(hostRadial)
	if rim, ok := dipRimPointAtStation(detA, ef, z); ok && aTangent.VectorTo(rim).Dot(wallRadial) > 0 {
		return rim
	}
	return aTangent
}

// radiusArcRail builds the section rail as the radius-ef.cyl.Radius circular arc through pA and pB,
// bulging away from the fillet axis (radiusArcApex → Arc3dByThreePoints). When the apex collapses onto
// the pA–pB chord (a near-collinear, near-degenerate section — a vanishingly thin sliver end, §2.4),
// Arc3dByThreePoints reports collinear points and the rail falls back to a straight segment.
func radiusArcRail(pA, pB math.Point3, ef edgeFillet, z float64) geom.Curve3 {
	pm := radiusArcApex(pA, pB, ef.cyl.Radius, filletAxisAt(ef, z), ef.cyl.AxisDir.AsVector())
	arc, err := geom.Arc3dByThreePoints(pA, pm, pB)
	if err != nil {
		return geom.NewLineSegment(pA, pB)
	}
	return arc
}

// radiusArcApex returns the apex (arc midpoint) of the radius-r circular arc through pA and pB that
// lies in the station plane (normal axisDir) and bulges outward, away from axisPt (the fillet axis
// point at the station). The centre sits at chord-distance h = √(r²−(|chord|/2)²) on the concave side,
// so the apex is mid + nOut·(r−h). When the half-chord already reaches or exceeds r (no radius-r circle
// spans the endpoints) the apex degenerates to the chord midpoint, which makes Arc3dByThreePoints see
// three collinear points and drives radiusArcRail's straight-segment fallback (§2.4).
func radiusArcApex(pA, pB math.Point3, r float64, axisPt math.Point3, axisDir math.Vector3) math.Point3 {
	mid := pA.Lerp(pB, 0.5)
	chord := pA.VectorTo(pB)
	half := chord.Length() / 2
	if half >= r {
		return mid
	}
	nOut, err := math.UnitVector3FromVector(axisDir.Cross(chord))
	if err != nil {
		return mid
	}
	out := nOut.AsVector()
	if out.Dot(axisPt.VectorTo(mid)) < 0 {
		out = out.Scale(-1)
	}
	return mid.TranslateBy(out.Scale(r - stdmath.Sqrt(r*r-half*half)))
}

// filletAxisAt returns the point on the fillet cylinder axis at axis-station z (axisParam == z): the
// cylinder origin slid along its axis by the station offset. It is the outward reference radiusArcApex
// uses to orient the section bulge and the anchor sectionEndA pushes the A-tangent point out from.
func filletAxisAt(ef edgeFillet, z float64) math.Point3 {
	origin := ef.cyl.Origin
	return origin.TranslateBy(ef.cyl.AxisDir.AsVector().Scale(z - axisParam(ef, origin)))
}

// dipRimPointAtStation returns the point on a boss's DIP-side rim at axis-station z: the rim is
// intersected with the station plane (rimPlaneCrossings) and the crossing lying on the dip arc is kept
// (nearestDipCrossing). Working on the analytic rim curve (a circle for host A, a b-spline ellipse for
// host B) rather than the 64-chord sampled polyline keeps the endpoint at the oracle floor (~1e-6), not
// the polyline's ~1e-2 sagitta. ok=false when the rim does not reach the station (no crossing).
func dipRimPointAtStation(d obstacleDetection, ef edgeFillet, z float64) (math.Point3, bool) {
	crossings := rimPlaneCrossings(d.holeEdge.Geometry(), ef, z)
	if len(crossings) == 0 {
		return math.Point3{}, false
	}
	return nearestDipCrossing(crossings, d), true
}

// rimPlaneCrossings returns every point where the closed rim curve crosses the station plane
// (axisParam == z), found by scanning the rim for sign changes of axisParam−z and bisecting each strict
// bracket on the analytic curve. A sample that sits EXACTLY on the plane (f0==0) is a crossing at the
// sample itself — this is not an edge case but the norm for host A, whose rim circle seam (parameter 0)
// coincides with its widest, dip-side point at z==0; bisecting that degenerate zero-endpoint bracket
// would walk to the far sample (the z=0.196 misread this guards against). rimScanSamples is dense
// enough to isolate the (at most two) crossings of a convex rim; bisection refines each to ~1e-12.
func rimPlaneCrossings(rim geom.Curve3, ef edgeFillet, z float64) []math.Point3 {
	f := func(t float64) float64 { return axisParam(ef, rim.PointAt(t)) - z }
	var out []math.Point3
	for i := 0; i < rimScanSamples; i++ {
		t0, t1 := float64(i)/float64(rimScanSamples), float64(i+1)/float64(rimScanSamples)
		f0, f1 := f(t0), f(t1)
		if f0 == 0 {
			out = append(out, rim.PointAt(t0)) // seam / exact-on-plane sample (t1==0 handled at i=0)
		} else if f1 != 0 && (f0 < 0) != (f1 < 0) {
			out = append(out, rim.PointAt(bisectRimParam(f, t0, t1)))
		}
	}
	return out
}

// rimScanSamples is the coarse scan density for isolating a rim's station-plane crossings before
// bisection. 256 keeps each bracket well inside one monotone half of a convex rim (a circle or the
// oblique ellipse), so no crossing pair is missed and none is double-counted.
const rimScanSamples = 256

// bisectRimParam refines a sign-change bracket [lo,hi] of f to the parameter where f==0. 60 iterations
// drive the interval below float64's resolution — this is a parametric root solve, not a model-length
// weld, so it converges on the parameter itself (tol:parametric).
func bisectRimParam(f func(float64) float64, lo, hi float64) float64 {
	flo := f(lo)
	for i := 0; i < 60; i++ {
		mid := (lo + hi) / 2
		fmid := f(mid)
		if (flo < 0) == (fmid < 0) {
			lo, flo = mid, fmid
		} else {
			hi = mid
		}
	}
	return (lo + hi) / 2
}

// nearestDipCrossing selects, among a rim's station-plane crossings, the one on the DIP arc — the arc
// that pokes onto the fillet side, sampled by dipRimSamples. A convex rim crosses the station plane
// twice (dip side and far side); the dip point is the crossing nearest the dip polyline. This is
// type-generic (no per-surface centre math) and robust for both the circular and elliptical rims.
func nearestDipCrossing(crossings []math.Point3, d obstacleDetection) math.Point3 {
	dip := dipRimSamples(d)
	best, bestDist := crossings[0], stdmath.Inf(1)
	for _, c := range crossings {
		if dist := distToPolyline(c, dip); dist < bestDist {
			best, bestDist = c, dist
		}
	}
	return best
}

// distToPolyline returns the minimum distance from p to the open polyline pts (its segments).
func distToPolyline(p math.Point3, pts []math.Point3) float64 {
	best := stdmath.Inf(1)
	for i := 0; i+1 < len(pts); i++ {
		if dist := geom.DistancePointToSegment(geom.NewLineSegment(pts[i], pts[i+1]), p); dist < best {
			best = dist
		}
	}
	return best
}
