// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The EllipticalCylinder∧Cone rim canal (OCCT blend/tolblend_simple B4/B7/B8/C2/C3): the
// cone-cap sibling of the plane-cap elliptic rim canal. The spine is the closed form in
// fillet_elliptic_cone_spine.go; the band is the PINCHED canal loft (the section collapses at
// the host-tangency azimuth — geom.LoftPinchedCanalStations). Two topologies ship:
//   - CLOSED rim (B7/C2/C3): the band is one teardrop face whose rails both close through the
//     pinch vertex; the host seams are re-aimed onto the rails (fillet_elliptic_cone_rebuild.go).
//   - OPEN arc rim (B4/B8): the band runs from a planar side-face runout (the section arc lies
//     IN that face's plane and imprints the added lens on it) to the pinch, which coincides with
//     the arc's far END VERTEX (fillet_elliptic_cone_runout.go).
// Everything else — an unpinched cone-cap rim, a pinch away from an open arc's end, a runout not
// landing on a plane — declines honestly to the byte-identical flat refusal (do-no-harm).

// ellipticConeCanal is the built cone-cap canal payload, carried inside ellipticRimCanal
// (armEllipticRim) so the existing single-pick weld dispatch routes it with zero new plumbing.
type ellipticConeCanal struct {
	spine        ellipticConeRimSpine
	loft         geom.PinchedCanalLoft
	stations     []ellipticConeStation
	closed       bool
	pinch        math.Point3
	wallF, coneF *topo.Face
}

// ellipticConeRimArmEdge classifies an EllipticalCylinder∧Cone rim (closed circle or open arc)
// and builds its pinched canal band. handled=true ONLY when the band was built; every decline
// falls through to the byte-identical curvedAdjacentError refusal.
func ellipticConeRimArmEdge(body *topo.Body, e *topo.Edge, p filletPick) (edgeFillet, bool) {
	ec, cone, wallF, coneF, ok := ellipticalCylinderConeEdge(e)
	if !ok {
		return edgeFillet{}, false
	}
	spine, ok := newEllipticConeRimSpine(body, e, ec, cone, wallF, coneF, p.r0)
	if !ok {
		return edgeFillet{}, false
	}
	canal, ok := buildEllipticConeCanal(body, e, spine, wallF, coneF)
	if !ok {
		return edgeFillet{}, false
	}
	faces := e.Faces()
	rim := &ellipticRimCanal{surf: canal.loft.Surf, concave: true, r: spine.r, coneCap: canal}
	return edgeFillet{a: faces[0], b: faces[1], edge: e, armSurface: canal.loft.Surf, armEllipticRim: rim}, true
}

// ellipticalCylinderConeEdge reports an edge bounded by one EllipticalCylinder face and one Cone
// face — the cone-cap sibling of ellipticalCylinderPlaneEdge.
func ellipticalCylinderConeEdge(e *topo.Edge) (geom.EllipticalCylinder, geom.Cone, *topo.Face, *topo.Face, bool) {
	faces := e.Faces()
	if len(faces) != 2 {
		return geom.EllipticalCylinder{}, geom.Cone{}, nil, nil, false
	}
	for i := 0; i < 2; i++ {
		ec, okE := faces[i].Geometry().(geom.EllipticalCylinder)
		co, okC := faces[1-i].Geometry().(geom.Cone)
		if okE && okC {
			return ec, co, faces[i], faces[1-i], true
		}
	}
	return geom.EllipticalCylinder{}, geom.Cone{}, nil, nil, false
}

// buildEllipticConeCanal finds the pinch, resolves the station density against the measured
// envelope error, and packages the band. ok=false on every geometric decline.
func buildEllipticConeCanal(body *topo.Body, e *topo.Edge, spine ellipticConeRimSpine, wallF, coneF *topo.Face) (*ellipticConeCanal, bool) {
	res := ResolutionForBody(body)
	span, ok := resolveEllipticConeRimSpan(e, spine, res)
	if !ok {
		return nil, false
	}
	for n := ellipticConeStationsMin; n <= ellipticRimStationsMax; n *= 2 {
		canal, ok := resolveEllipticConeCanal(e, spine, span, wallF, coneF, n, res)
		if !ok {
			return nil, false
		}
		if canal != nil && ellipticConeEnvelopeError(spine, canal) <= ellipticRimEnvelopeCoef*res.Weld() {
			return canal, true
		}
	}
	return nil, false
}

// ellipticConeStationsMin floors the adaptive station count. The pinched band's section varies
// quadratically near the tangency azimuth, so it starts denser than the plane-cap canal's 32.
const ellipticConeStationsMin = 64

// ellipticConeRimSpan is the resolved azimuth walk: from the runout/seam end to the pinch (open)
// or the full period from the pinch back to itself (closed).
type ellipticConeRimSpan struct {
	u0, u1     float64 // walk from u0 to u1 (u1 is always a pinched end)
	pinchStart bool    // closed rims pinch at BOTH ends (u0 = u1 − 2π at the same point)
	closed     bool
	pinch      math.Point3
}

// resolveEllipticConeRimSpan locates the host-tangency pinch on the rim and fixes the walk.
// Declines when the rim does not pinch (min section chord above the band's geometric slack),
// when an open arc's pinch is not at an arc end, or when a closed rim's pinch is off the host
// seam (closedEllipticConeSpan — a measured decline, not a missing check).
func resolveEllipticConeRimSpan(e *topo.Edge, spine ellipticConeRimSpine, res Resolution) (ellipticConeRimSpan, bool) {
	closed := e.StartVertex() == e.EndVertex()
	lo, hi := ellipticConeRimURange(e, spine, closed)
	uStar, chord, ok := ellipticConePinchAzimuth(spine, lo, hi)
	if !ok || chord > ellipticRimEnvelopeCoef*res.Weld() {
		return ellipticConeRimSpan{}, false // no pinch: the unpinched cone-cap rim is a different construction
	}
	st, ok := spine.solveStation(uStar)
	if !ok {
		return ellipticConeRimSpan{}, false
	}
	if closed {
		return closedEllipticConeSpan(e, spine, st.wallFoot, res)
	}
	return openEllipticConeSpan(e, spine, lo, hi, res)
}

// ellipticConeRimURange is the rim's wall-azimuth range: the full period for a closed rim, the
// arc between the endpoint azimuths (walked through the edge midpoint) for an open one.
func ellipticConeRimURange(e *topo.Edge, spine ellipticConeRimSpine, closed bool) (lo, hi float64) {
	if closed {
		u0, _ := spine.ec.ParamAt(e.StartVertex().Point())
		return u0, u0 + 2*stdmath.Pi
	}
	ua, _ := spine.ec.ParamAt(e.StartVertex().Point())
	ub, _ := spine.ec.ParamAt(e.EndVertex().Point())
	um, _ := spine.ec.ParamAt(edgeMidpoint(e))
	return orientArcRange(ua, ub, um)
}

// orientArcRange returns the arc's [lo,hi] azimuth walk from ua to ub that passes through the
// on-arc midpoint azimuth um (the two-candidate wrap disambiguation).
func orientArcRange(ua, ub, um float64) (float64, float64) {
	fwd := wrap2piPositive(ub - ua) // walking +u from ua
	mid := wrap2piPositive(um - ua) // midpoint's offset along that walk
	if mid <= fwd {
		return ua, ua + fwd
	}
	return ua - (2*stdmath.Pi - fwd), ua // walking −u from ua lands on ub too
}

// wrap2piPositive wraps an angle difference into [0, 2π).
func wrap2piPositive(d float64) float64 {
	w := stdmath.Mod(d, 2*stdmath.Pi)
	if w < 0 {
		w += 2 * stdmath.Pi
	}
	return w
}

// ellipticConePinchAzimuth scans the rim for the minimum section chord and ternary-refines it:
// the host-tangency azimuth. Returns the azimuth, the minimal chord (2r·sin half-angle), ok.
func ellipticConePinchAzimuth(spine ellipticConeRimSpine, lo, hi float64) (float64, float64, bool) {
	const scan = 720
	bestU, bestA, ok := 0.0, stdmath.Inf(1), false
	for k := 0; k <= scan; k++ {
		u := lo + (hi-lo)*float64(k)/scan
		if st, sok := spine.solveStation(u); sok {
			if a := spine.sectionHalfAngle(st); a < bestA {
				bestU, bestA, ok = u, a, true
			}
		}
	}
	if !ok {
		return 0, 0, false
	}
	u := ternaryMinHalfAngle(spine, bestU-(hi-lo)/scan, bestU+(hi-lo)/scan)
	u = stdmath.Max(lo, stdmath.Min(hi, u))
	st, sok := spine.solveStation(u)
	if !sok {
		return 0, 0, false
	}
	return u, 2 * spine.r * stdmath.Sin(spine.sectionHalfAngle(st)), true
}

// ternaryMinHalfAngle refines the half-angle minimum on [a,b] by ternary search (the objective
// is smooth and locally quadratic at the tangency).
func ternaryMinHalfAngle(spine ellipticConeRimSpine, a, b float64) float64 {
	f := func(u float64) float64 {
		st, ok := spine.solveStation(u)
		if !ok {
			return stdmath.Inf(1)
		}
		return spine.sectionHalfAngle(st)
	}
	for i := 0; i < 200; i++ {
		m1, m2 := a+(b-a)/3, b-(b-a)/3
		if f(m1) <= f(m2) {
			b = m2
		} else {
			a = m1
		}
	}
	return 0.5 * (a + b)
}
