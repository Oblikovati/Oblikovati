// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The CONCAVE closed-rim band solve + spill gate (concave-sphere-cone-arm-derivation.md §3, §5). It
// mirrors solveClosedRimBand (the convex J1 path) with the concave contact-circle branch (external
// tangency ρ = R+r / −r) and the derivation's §5 honest-reject: the plane-contact rail radius equals the
// arm major, so when it exceeds the cap face's own boundary the cove spills off the plate onto the box
// side walls and no clean closed band is watertight — this slice declines with the offending major vs
// the plate extent rather than shipping a body whose grown plate hole pokes past the plate outline.

// solveConcaveClosedRimBand resolves the concave closed-band rim fillet: the curved host + cap faces, the
// arm torus re-framed onto the rim-vertex azimuth, the concave (external-tangency) contact circles, and
// the spill gate. Declines (naming the offending value) when the hosts are not one curved host + one cap
// plane, the rim vertex lies on the torus axis, the reframe declines, or the plane-contact rail spills.
func solveConcaveClosedRimBand(ef edgeFillet, res Resolution) (*rimFillet, string) {
	arm := ef.armSurface.(geom.Torus)
	e := ef.edge
	devF, capF, ok := rimBandHosts(e)
	if !ok {
		return nil, fmt.Sprintf("concave closed-rim band: edge %d must border one curved host-of-revolution and one cap plane", e.ID())
	}
	rimV := e.StartVertex()
	ref, err := math.UnitVector3FromVector(perpComponent(arm.Center.VectorTo(rimV.Point()), arm.AxisDir))
	if err != nil {
		return nil, fmt.Sprintf("concave closed-rim band: rim vertex %v is on the torus axis — degenerate seam frame", rimV.Point())
	}
	tor, err := geom.NewTorusWithRef(arm.Center, arm.AxisDir.AsVector(), ref.AsVector(), arm.MajorRadius, arm.MinorRadius)
	if err != nil {
		return nil, fmt.Sprintf("concave closed-rim band: torus reframe declined: %v", err)
	}
	return assembleConcaveRimBand(ef, devF, capF, tor, ref, rimV, res)
}

// assembleConcaveRimBand builds the concave rimFillet, gating on the SPILL boundary (§5). The two closed
// contact circles come from the concave (external-tangency) branch; the plane-side rail radius is the arm
// major, and when it exceeds the cap face's boundary the cove spills off the plate — an honest reject
// carrying the offending major and the plate extent, since this slice does not build the cove-onto-wall
// multi-face interaction (a follow-on slice does). Otherwise it returns the concave band for the rebuild.
func assembleConcaveRimBand(ef edgeFillet, devF, capF *topo.Face, tor geom.Torus, ref math.UnitVector3, rimV *topo.Vertex, res Resolution) (*rimFillet, string) {
	capCenter, capR, ok1 := concaveTorusContactCircle(capF.Geometry(), tor, res)
	devCenter, devR, ok2 := concaveTorusContactCircle(devF.Geometry(), tor, res)
	if !ok1 || !ok2 {
		return nil, fmt.Sprintf("concave closed-rim band: contact circle unresolved (cap ok=%v, host %T ok=%v)", ok1, devF.Geometry(), ok2)
	}
	if fits, extent := contactCircleFitsFace(capF, capCenter, capR); !fits {
		return nil, fmt.Sprintf("concave closed-rim band: plane-contact radius %.6g exceeds the cap face half-extent %.6g "+
			"— the cove spills off the plate onto the adjacent walls (a multi-face concave interaction is a follow-on slice)", capR, extent)
	}
	capTan := geom.Circle{Center: capCenter, Normal: tor.AxisDir, RefDir: ref, Radius: capR}
	devTan := geom.Circle{Center: devCenter, Normal: tor.AxisDir, RefDir: ref, Radius: devR}
	seamEdge, bottomV := wallSeam(devF, ef.edge, rimV)
	if seamEdge == nil {
		return nil, fmt.Sprintf("concave closed-rim band: host %T has no seam edge at the rim vertex to recede", devF.Geometry())
	}
	return &rimFillet{
		cyl: devF, cap: capF, rimEdge: ef.edge, seamEdge: seamEdge, rimV: rimV, bottomV: bottomV,
		cylTan: devTan, capTan: capTan, band: tor, r: tor.MinorRadius, seamMid: rimBandSeamMid(tor, devTan, capTan),
	}, ""
}

// concaveTorusContactCircle returns the centre + radius of the circle where the CONCAVE (external-
// tangency) arm torus touches host: the tube-top circle (radius major) in a perpendicular cap plane
// (capContactCircle, sign-agnostic), or the EXTERNAL-tangency circle on a host sphere/cone. It is the
// concave dual of torusContactCircle — a SEPARATE entry so the convex closed rim (J1) and every corner
// weld keep calling torusContactCircle byte-identically (the convex internal-tangency asserts are
// untouched). ok=false when the host is not a cap plane / coaxial host sphere / coaxial host cone.
func concaveTorusContactCircle(host geom.Surface, tor geom.Torus, res Resolution) (math.Point3, float64, bool) {
	switch h := host.(type) {
	case geom.Plane:
		return capContactCircle(h, tor, res)
	case geom.Sphere:
		return concaveSphereContactCircle(h, tor, res)
	case geom.Cone:
		return concaveConeContactCircle(h, tor, res)
	default:
		return math.Point3{}, 0, false
	}
}

// concaveSphereContactCircle is the torus↔host-sphere contact for the CONCAVE (external-tangency) arm:
// the projection formulas are IDENTICAL to sphereContactCircle (k = R/ρ; centre O + k·(O′−O); radius
// k·major), only the tangency certificate flips from ρ = R−r to ρ = R+r. For S5: ρ = √(h²+major²) = 16 =
// R+r, k = 13/16, centre (0,0,2.4375), radius 12.769 (on the dome, above the rim). Rejects a torus not
// externally tangent to the host sphere (|ρ − (R+r)| > tol) or one whose spine passes the sphere centre.
func concaveSphereContactCircle(sph geom.Sphere, tor geom.Torus, res Resolution) (math.Point3, float64, bool) {
	n := tor.AxisDir.AsVector()
	h := float64(tor.Center.VectorTo(sph.Center).Dot(n)) // n̂·(O−O′) = h
	rho := stdmath.Sqrt(h*h + tor.MajorRadius*tor.MajorRadius)
	if rho < res.Weld()*sph.Radius {
		return math.Point3{}, 0, false // spine through the sphere centre — degenerate
	}
	if stdmath.Abs(rho-(sph.Radius+tor.MinorRadius)) > res.Weld()*sph.Radius {
		return math.Point3{}, 0, false // torus not EXTERNALLY tangent to the host sphere (ρ ≠ R+r)
	}
	k := sph.Radius / rho
	center := sph.Center.TranslateBy(sph.Center.VectorTo(tor.Center).Scale(math.Scalar(k)))
	return center, k * tor.MajorRadius, true
}

// concaveConeContactCircle is the torus↔host-cone contact for the CONCAVE (external-tangency) arm: the
// projection (star = h·cosα + R_s·sinα; centre A + star·cosα·â; radius star·sinα) is IDENTICAL to
// coneContactCircle, only the tangency certificate flips from +r to −r (h·sinα − R_s·cosα = −r). For S2:
// −8 = −r, star = 34.985, centre (0,0,6.06), radius 8.485. Rejects a torus not coaxial with the cone or
// not externally tangent to it.
func concaveConeContactCircle(co geom.Cone, tor geom.Torus, res Resolution) (math.Point3, float64, bool) {
	if !co.AxisDir.IsParallelTo(tor.AxisDir, retrimAxisParallelTol) {
		return math.Point3{}, 0, false
	}
	a := co.AxisDir.AsVector()
	h := float64(co.Apex.VectorTo(tor.Center).Dot(a))
	band := res.Weld() * (tor.MajorRadius + tor.MinorRadius)
	if float64(co.Apex.TranslateBy(a.Scale(h)).DistanceTo(tor.Center)) > band {
		return math.Point3{}, 0, false // torus centre off the cone axis — not coaxial
	}
	sinA, cosA := stdmath.Sincos(co.HalfAngle)
	if stdmath.Abs(h*sinA-tor.MajorRadius*cosA+tor.MinorRadius) > band {
		return math.Point3{}, 0, false // tube not EXTERNALLY tangent to the cone (h·sinα − R_s·cosα ≠ −r)
	}
	star := h*cosA + tor.MajorRadius*sinA
	return co.Apex.TranslateBy(a.Scale(star * cosA)), star * sinA, true
}

// contactCircleFitsFace reports whether a circle of the given centre + radius on face f's plane lies
// wholly within f's outer loop (and clear of its holes), and returns the face half-extent (the minimum
// in-plane distance from the centre to the outer loop boundary) for the spill diagnostic. A concave
// plane-contact rail wider than that half-extent spills off the plate (§5), so the caller honest-rejects.
func contactCircleFitsFace(f *topo.Face, center math.Point3, radius float64) (bool, float64) {
	pl, ok := f.Geometry().(geom.Plane)
	if !ok {
		return false, 0
	}
	uv := func(p math.Point3) math.Point2 {
		d := pl.Origin.VectorTo(p)
		return math.P2(d.Dot(pl.UAxis.AsVector()), d.Dot(pl.VAxis.AsVector()))
	}
	c2, extent := uv(center), stdmath.Inf(1)
	for _, l := range f.Loops() {
		if l.IsOuter() {
			extent = minDistToPolygon(c2, loopUVPolygon(l, uv))
		}
	}
	return radius <= extent, extent
}

// minDistToPolygon is the minimum distance from point q to the closed polygon poly (its edges).
func minDistToPolygon(q math.Point2, poly []math.Point2) float64 {
	if len(poly) < 2 {
		return stdmath.Inf(1)
	}
	best := stdmath.Inf(1)
	for i := range poly {
		d := distPointSegment(q, poly[i], poly[(i+1)%len(poly)])
		if d < best {
			best = d
		}
	}
	return best
}

// distPointSegment is the distance from q to the segment ab (clamped projection).
func distPointSegment(q, a, b math.Point2) float64 {
	ab := a.VectorTo(b)
	l2 := float64(ab.Dot(ab))
	if l2 == 0 {
		return float64(q.DistanceTo(a))
	}
	t := stdmath.Max(0, stdmath.Min(1, float64(a.VectorTo(q).Dot(ab))/l2))
	return float64(q.DistanceTo(a.TranslateBy(ab.Scale(math.Scalar(t)))))
}
