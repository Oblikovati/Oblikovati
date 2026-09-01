// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Fit gates and the measured envelope error of the EllipticalCylinder∧Cone pinched canal — the
// do-no-harm floor that keeps a band from overrunning either host face (the cone-cap analogue of
// fillet_elliptic_rim_fit.go): every wall foot must stay between the rim and the wall's far
// boundary plane along its ruling, every cone foot between the rim and the cone's far boundary
// plane along its slant.

// ellipticConeFitGates carries the two hosts' far boundary planes, resolved once per build, and
// the band's geometric slack (ellipticRimEnvelopeCoef·weld): the pinch-region stations overshoot
// the rim boundary by the tangency slop (B8/C2's 1e-4 near-tangent defect — the very "tolerant
// blending" dimension of the fixtures), which is inside the band's own documented tolerance, so
// the gates widen by exactly that budget and no more.
type ellipticConeFitGates struct {
	wallFarN, coneFarN math.UnitVector3
	wallFarC, coneFarC float64
	slack              float64
}

// newEllipticConeFitGates resolves the far boundary planes of both hosts. ok=false (decline)
// when either host has no planar far boundary — a fillet running into a curved neighbour is a
// construction this vein does not model.
func newEllipticConeFitGates(e *topo.Edge, wallF, coneF *topo.Face, res tol.Resolution) (ellipticConeFitGates, bool) {
	wn, wc, ok := farBoundaryPlane(e, wallF)
	if !ok {
		return ellipticConeFitGates{}, false
	}
	cn, cc, ok := farBoundaryPlane(e, coneF)
	if !ok {
		return ellipticConeFitGates{}, false
	}
	return ellipticConeFitGates{wallFarN: wn, wallFarC: wc, coneFarN: cn, coneFarC: cc,
		slack: ellipticRimEnvelopeCoef * res.Weld()}, true
}

// farBoundaryPlane returns the plane bounding a host face at its FAR side: the plane of the face
// across any non-ruling boundary edge other than the rim itself. The generalization of
// ellipticWallFarPlane to OPEN boundary arcs (the B4/B8 half-solid's caps).
func farBoundaryPlane(e *topo.Edge, hostF *topo.Face) (math.UnitVector3, float64, bool) {
	for _, he := range hostF.Edges() {
		if he == e {
			continue
		}
		if _, isLine := he.Geometry().(geom.LineSegment); isLine {
			continue // seam/side rulings do not bound the far side
		}
		if n, c, ok := planeAcrossEdge(he, hostF); ok {
			return n, c, true
		}
	}
	return math.UnitVector3{}, 0, false
}

// planeAcrossEdge returns the plane of the face on the other side of he from hostF, if planar.
func planeAcrossEdge(he *topo.Edge, hostF *topo.Face) (math.UnitVector3, float64, bool) {
	for _, f := range he.Faces() {
		if f == hostF {
			continue
		}
		if p, isPlane := f.Geometry().(geom.Plane); isPlane {
			n := p.Normal().AsUnit()
			return n, float64(math.P3(0, 0, 0).VectorTo(p.Origin).Dot(n.AsVector())), true
		}
	}
	return math.UnitVector3{}, 0, false
}

// admit gates one station: the wall foot inside the wall's axial extent, the cone foot inside
// the cone face's slant extent.
func (g ellipticConeFitGates) admit(spine ellipticConeRimSpine, u float64, st ellipticConeStation) bool {
	return g.wallFootInside(spine, u, st) && g.coneFootInside(spine, st)
}

// wallFootInside checks the foot's ruling parameter v between the rim crossing and the far-plane
// crossing of the same ruling (the plane-cap ellipticRimWallSpanFits test, cone-cap edition).
func (g ellipticConeFitGates) wallFootInside(spine ellipticConeRimSpine, u float64, st ellipticConeStation) bool {
	den := float64(g.wallFarN.AsVector().Dot(spine.ec.AxisDir.AsVector()))
	if stdmath.Abs(den) < ellipticRimAxisTiltTol {
		return false
	}
	p0 := spine.ec.PointAt(u, 0)
	vRim := spine.vRimAt(u)
	vFar := (g.wallFarC - float64(math.P3(0, 0, 0).VectorTo(p0).Dot(g.wallFarN.AsVector()))) / den
	return betweenSlack(st.v, vRim, vFar, g.slack)
}

// coneFootInside checks the foot's axial position between the rim-plane and far-plane crossings
// of its own slant line.
func (g ellipticConeFitGates) coneFootInside(spine ellipticConeRimSpine, st ellipticConeStation) bool {
	c := spine.cone.AxisDir.AsVector()
	d := spine.cone.Apex.VectorTo(st.coneFoot)
	ax := d.Dot(c)
	rad, err := math.UnitVector3FromVector(d.Sub(c.Scale(ax)))
	if err != nil {
		return false
	}
	slant := c.Add(rad.AsVector().Scale(stdmath.Tan(spine.cone.HalfAngle)))
	sRim, ok1 := slantPlaneCrossing(spine.cone.Apex, slant, spine.nRim, spine.cRim)
	sFar, ok2 := slantPlaneCrossing(spine.cone.Apex, slant, g.coneFarN, g.coneFarC)
	return ok1 && ok2 && betweenSlack(ax, sRim, sFar, g.slack)
}

// slantPlaneCrossing solves n̂·(apex + s·slant) = c for the axial coordinate s.
func slantPlaneCrossing(apex math.Point3, slant math.Vector3, n math.UnitVector3, c float64) (float64, bool) {
	den := float64(n.AsVector().Dot(slant))
	if stdmath.Abs(den) < ellipticRimAxisTiltTol {
		return 0, false
	}
	return (c - float64(math.P3(0, 0, 0).VectorTo(apex).Dot(n.AsVector()))) / den, true
}

// betweenSlack reports x inside the interval spanned by a and b (either order), widened by the
// band's geometric slack — the PINCH station's foot sits ON (or, for the near-tangent B8/C2
// fixtures, up to their 1e-4 slop PAST) the rim boundary, and that overshoot is snapped into the
// pinch column within the same budget.
func betweenSlack(x, a, b, slack float64) bool {
	lo, hi := stdmath.Min(a, b), stdmath.Max(a, b)
	return x >= lo-slack && x <= hi+slack
}

// ellipticConeEnvelopeError is the measured between-station error the density loop bounds: how
// far the two rails stray off their hosts and the arc interior off ball radius, sampled at every
// interval midpoint (station columns are exact by construction).
func ellipticConeEnvelopeError(spine ellipticConeRimSpine, canal *ellipticConeCanal) float64 {
	vp := canal.loft.VParams
	worst := 0.0
	for j := 0; j+1 < len(vp); j++ {
		worst = stdmath.Max(worst, ellipticConeIntervalError(spine, canal.loft.Surf, 0.5*(vp[j]+vp[j+1])))
	}
	return worst
}

// ellipticConeIntervalError measures the three deviations at one v.
func ellipticConeIntervalError(spine ellipticConeRimSpine, surf geom.BSplineSurface, v float64) float64 {
	qWall := surf.PointAt(0, v)
	uStar, _, foot := geom.ClosestPointOnSurface(spine.ec, qWall)
	err := float64(foot.DistanceTo(qWall))
	err = stdmath.Max(err, spine.coneDistance(surf.PointAt(1, v)))
	st, ok := spine.solveStation(uStar)
	if !ok {
		return stdmath.Inf(1)
	}
	for k := 1; k < ellipticRimArcSamples; k++ {
		q := surf.PointAt(float64(k)/float64(ellipticRimArcSamples), v)
		err = stdmath.Max(err, stdmath.Abs(float64(q.DistanceTo(st.center))-spine.r))
	}
	return err
}
