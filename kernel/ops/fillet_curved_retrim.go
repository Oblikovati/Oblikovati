// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// M5 Slice A, Task 5.3 (m5-weld-setback-retrim-derivation.md §B): the curved-HOST retrim. Each
// original host face at the trihedral corner (the cylinder wall, the top/bottom caps, the radial
// planes) is re-clipped to a single outer loop whose corner bite is bounded by the arm CONTACT
// RAILS. Two of those rails are CIRCULAR arcs the ordinary transformFace (which only pulls a
// straight tangent vertex) cannot emit: the torus arm carves a circle of radius R in the wall
// (spine plane z=C_z) and a circle of radius R−r in the cap (§B.1/B.2), and a through-cylinder arm
// carves its cross-section circle on the cap it exits (§B.5 foot-bite). This file emits those exact
// arcs plus the straight rulings, and assembles the retrimmed host loops. It wires into nothing yet
// — T5.4 consumes retrimCurvedHost to build the nine result faces.

// retrimAxisParallelTol is the dimensionless floor for the "coaxial / perpendicular" host↔arm
// tests (a torus rail exists only on a wall coaxial with the torus axis, or a cap perpendicular to
// it). An angle carries no length, so ADR-0042's model-relative rule does not apply — a scale-free
// constant is correct. 1e-9 sits far inside the exact quadric geometry yet rejects a real
// misalignment (a 1° tilt is 0.017 in the parallel residual, eight orders larger).
const retrimAxisParallelTol = 1e-9

// torusArmStation returns the torus arm's setback major angle φ* from the solved corner — the
// azimuth span 0→φ* every torus contact rail sweeps (§B.1: wall arc 0°→−75.522°). ok=false when no
// torus arm meets at this corner (then there is no circular host rail to emit).
func torusArmStation(w cornerWeld) (float64, bool) {
	for _, a := range w.arms {
		if _, ok := a.arm.(geom.Torus); ok {
			return a.station, true
		}
	}
	return 0, false
}

// curvedHostArc returns the circular contact rail where the torus arm meets a host: on the WALL it
// is the circle radius R in the spine plane z=C_z (the torus outer equator); on the CAP it is the
// circle radius R−r in the cap plane (the tube-top circle). Both sweep the arm's azimuth 0→φ*, so
// PointAt(0) is the far (y=0 cut) end and PointAt(1) is the sphere-side host-tangent point. The host
// type selects which. It honest-rejects (ok=false) when the host is neither a coaxial wall nor a
// perpendicular tangent cap, so a spurious rail is never emitted. Example:
//
//	arc, ok := curvedHostArc(wall.Geometry(), torusArm, w, res)
//	if !ok { /* this host carries no torus rail */ }
func curvedHostArc(host geom.Surface, tor geom.Torus, w cornerWeld, res Resolution) (geom.Arc3d, bool) {
	phi, ok := torusArmStation(w)
	if !ok {
		return geom.Arc3d{}, false
	}
	center, radius, ok := torusContactCircle(host, tor, res)
	if !ok {
		return geom.Arc3d{}, false
	}
	arc, err := geom.NewArc3d(center, tor.AxisDir.AsVector(), tor.Ref.AsVector(), radius, 0, phi)
	if err != nil {
		return geom.Arc3d{}, false
	}
	return arc, true
}

// torusContactCircle returns the centre + radius of the circle where torus tor touches host: the
// outer-equator circle (radius R) in the spine plane on a coaxial cylinder wall, or the tube-top
// circle (radius ρ = MajorRadius) in a cap plane perpendicular to the torus axis. ok=false otherwise.
func torusContactCircle(host geom.Surface, tor geom.Torus, res Resolution) (math.Point3, float64, bool) {
	switch h := host.(type) {
	case geom.Cylinder:
		return wallContactCircle(h, tor, res)
	case geom.Plane:
		return capContactCircle(h, tor, res)
	default:
		return math.Point3{}, 0, false // only a cylinder wall or a cap plane carries a torus rail
	}
}

// wallContactCircle is the torus↔wall contact: the torus outer equator (radius ρ+r) in the spine
// plane, which on the exact geometry coincides with the wall (radius R, coaxial). Rejects a
// non-coaxial or non-tangent wall (torus centre off the axis, or ρ+r ≠ R beyond the model tolerance).
func wallContactCircle(cyl geom.Cylinder, tor geom.Torus, res Resolution) (math.Point3, float64, bool) {
	if !cyl.AxisDir.IsParallelTo(tor.AxisDir, retrimAxisParallelTol) {
		return math.Point3{}, 0, false // wall axis not parallel to the torus axis
	}
	axis := cyl.AxisDir.AsVector()
	foot := cyl.Origin.TranslateBy(axis.Scale(cyl.Origin.VectorTo(tor.Center).Dot(axis)))
	if float64(foot.DistanceTo(tor.Center)) > res.Weld()*cyl.Radius {
		return math.Point3{}, 0, false // torus centre off the wall axis — not coaxial
	}
	if stdmath.Abs((tor.MajorRadius+tor.MinorRadius)-cyl.Radius) > res.Weld()*cyl.Radius {
		return math.Point3{}, 0, false // outer equator ρ+r ≠ wall R — not tangent
	}
	return tor.Center, cyl.Radius, true
}

// capContactCircle is the torus↔cap contact: the tube-top circle (radius ρ) in the cap plane, which
// must be perpendicular to the torus axis and tangent to the tube (axial distance centre→plane =
// minor r). Rejects a plane not ⊥ the axis or not tangent — either would misplace the rail.
func capContactCircle(pl geom.Plane, tor geom.Torus, res Resolution) (math.Point3, float64, bool) {
	n, err := math.UnitVector3FromVector(pl.Normal())
	if err != nil {
		return math.Point3{}, 0, false
	}
	if !n.IsParallelTo(tor.AxisDir, retrimAxisParallelTol) {
		return math.Point3{}, 0, false // cap plane not perpendicular to the torus axis
	}
	axis := tor.AxisDir.AsVector()
	nv := n.AsVector()
	t := float64(tor.Center.VectorTo(pl.Origin).Dot(nv) / axis.Dot(nv)) // signed axial distance centre→plane
	if stdmath.Abs(stdmath.Abs(t)-tor.MinorRadius) > res.Weld()*(tor.MajorRadius+tor.MinorRadius) {
		return math.Point3{}, 0, false // cap not tangent to the tube (|axial dist| ≠ minor r)
	}
	return tor.Center.TranslateBy(axis.Scale(math.Scalar(t))), tor.MajorRadius, true
}

// retrimCurvedHost re-clips one host face at the trihedral corner to a single outer loop: the
// original outer edges, cut back where the arms/sphere contact them, plus the arm contact rails
// (circular arcs for the torus-adjacent hosts, straight rulings for the cylinder/planar-adjacent
// ones, and the through-arm foot arc on the cap it exits — §B.5). It honest-rejects (ok=false) when
// a contact rail is missing or a landing point does not lie on the original loop, never an open or
// self-crossing loop (a mis-closed retrim corrupts the mesh). Example:
//
//	ff, ok := retrimCurvedHost(wallFace, w, res)
//	if !ok { /* decline the weld — do-no-harm */ }
func retrimCurvedHost(host *topo.Face, w cornerWeld, res Resolution) (filletFace, bool) {
	segs := originalHostSegs(host)
	if len(segs) < 3 {
		return filletFace{}, false // a host loop needs ≥3 edges to bite a corner from
	}
	loop, ok := retrimHostSegs(host, segs, w, res)
	if !ok {
		return filletFace{}, false
	}
	return filletFace{surface: host.Geometry(), loops: []filletLoop{loopFromSegs(loop)}, parent: host.Lineage()}, true
}

// retrimHostSegs routes a host to its retrim mode: two corner arms meeting at a shared host-tangent
// point (wall/cap/radial), or a single through-arm foot-bite (the bottom cap the vertical arm exits).
// The on-edge / on-surface tolerance is corner-local (res.Weld·r, ADR-0042) — the corner is a local
// feature, so a body-diameter tolerance would mask a real crack.
func retrimHostSegs(host *topo.Face, segs []endSeg, w cornerWeld, res Resolution) ([]endSeg, bool) {
	tol := res.Weld() * w.radius
	v := bittenVertex(segs, w.center)
	arms, tHost, n := armsRollingOnHost(host, w, tol)
	if n == 2 {
		return retrimCornerHost(host, segs, v, arms, tHost, w, res, tol)
	}
	if n == 0 {
		return retrimFootHost(host, segs, v, w, tol)
	}
	return nil, false // a host touching exactly one corner arm is not part of this trihedral weld
}

// armsRollingOnHost returns the corner arms with a rail endpoint (a host-tangent point) lying on this
// host, plus that shared tangent point. Two arms → a corner host; none → a foot-bite host.
func armsRollingOnHost(host *topo.Face, w cornerWeld, tol float64) ([]armSetback, math.Point3, int) {
	surf := host.Geometry()
	var arms []armSetback
	var tHost math.Point3
	for _, a := range w.arms {
		for _, d := range [2]math.UnitVector3{a.railDir0, a.railDir1} {
			ep := endpointOf(w.center, w.radius, d)
			if onHostSurface(surf, ep, tol) {
				arms = append(arms, a)
				tHost = ep
				break
			}
		}
	}
	return arms, tHost, len(arms)
}

// onHostSurface reports whether p lies on the host surface within tol (signed distance to a plane, or
// radial distance to a cylinder wall).
func onHostSurface(surf geom.Surface, p math.Point3, tol float64) bool {
	switch h := surf.(type) {
	case geom.Plane:
		n, err := math.UnitVector3FromVector(h.Normal())
		if err != nil {
			return false
		}
		return stdmath.Abs(float64(h.Origin.VectorTo(p).Dot(n.AsVector()))) <= tol
	case geom.Cylinder:
		axis := h.AxisDir.AsVector()
		w := h.Origin.VectorTo(p)
		return stdmath.Abs(float64(w.Sub(axis.Scale(w.Dot(axis))).Length())-h.Radius) <= tol
	}
	return false
}

// retrimCornerHost assembles a two-arm host loop: rail A (outer→tHost), rail B (tHost→outer), then
// the surviving far path (outerB→outerA) that avoids the bitten trihedral vertex.
func retrimCornerHost(host *topo.Face, segs []endSeg, v math.Point3, arms []armSetback, tHost math.Point3, w cornerWeld, res Resolution, tol float64) ([]endSeg, bool) {
	railA, outerA, okA := armContactRail(host, arms[0], tHost, v, segs, w, res, tol)
	railB, outerB, okB := armContactRail(host, arms[1], tHost, v, segs, w, res, tol)
	if !okA || !okB {
		return nil, false
	}
	far, ok := farPathSegs(segs, outerB, outerA, v, tol)
	if !ok {
		return nil, false
	}
	out := append([]endSeg{railA}, reverseEndSegs([]endSeg{railB})...) // outerA→tHost→outerB
	return append(out, far...), true                                   // …→outerA (closed)
}

// armContactRail builds one arm's contact rail on a corner host as an endSeg oriented outer→tHost,
// plus the outer landing point. Torus arms carve a circular arc (curvedHostArc); cylinder arms carve
// a straight ruling from the far runout to tHost.
func armContactRail(host *topo.Face, arm armSetback, tHost, v math.Point3, segs []endSeg, w cornerWeld, res Resolution, tol float64) (endSeg, math.Point3, bool) {
	switch s := arm.arm.(type) {
	case geom.Torus:
		arc, ok := curvedHostArc(host.Geometry(), s, w, res)
		if !ok || float64(arc.PointAt(1).DistanceTo(tHost)) > tol {
			return endSeg{}, math.Point3{}, false // no torus rail here, or it misses the tangent point
		}
		outer := arc.PointAt(0)
		return endSeg{from: outer, to: tHost, curve: arc, mid: arc.PointAt(0.5), arc: true}, outer, true
	case geom.Cylinder:
		outer, ok := rulingOuterEnd(host, s, tHost, v, segs, tol)
		if !ok {
			return endSeg{}, math.Point3{}, false
		}
		return endSeg{from: outer, to: tHost}, outer, true
	}
	return endSeg{}, math.Point3{}, false
}

// rulingOuterEnd is the far end of a cylinder arm's straight ruling on a host: on a cylinder wall the
// ruling runs along the shared axis to the far rim (the axial extreme away from the bitten vertex);
// on a planar host it runs along the arm axis to where it exits the original loop.
func rulingOuterEnd(host *topo.Face, cylArm geom.Cylinder, tHost, v math.Point3, segs []endSeg, tol float64) (math.Point3, bool) {
	axis := cylArm.AxisDir.AsVector()
	switch h := host.Geometry().(type) {
	case geom.Cylinder:
		return axialExtremeEnd(h, segs, tHost, v, axis), true
	case geom.Plane:
		return planeRayLoopExit(h, segs, tHost, awayFrom(axis, tHost, v), tol)
	}
	return math.Point3{}, false
}

// axialExtremeEnd slides tHost along the wall axis to the loop's extreme axial coordinate on the side
// away from the bitten vertex — the far rim the vertical ruling reaches.
func axialExtremeEnd(cyl geom.Cylinder, segs []endSeg, tHost, v math.Point3, axis math.Vector3) math.Point3 {
	base := float64(cyl.Origin.VectorTo(tHost).Dot(axis))
	up := base >= float64(cyl.Origin.VectorTo(v).Dot(axis)) // away from v = away from its axial coord
	ext := base
	for _, s := range segs {
		a := float64(cyl.Origin.VectorTo(s.from).Dot(axis))
		if (up && a > ext) || (!up && a < ext) {
			ext = a
		}
	}
	return tHost.TranslateBy(axis.Scale(math.Scalar(ext - base)))
}

// awayFrom returns axis oriented to point away from the bitten vertex v (the side the ruling runs to).
func awayFrom(axis math.Vector3, tHost, v math.Point3) math.Vector3 {
	if float64(v.VectorTo(tHost).Dot(axis)) >= 0 {
		return axis
	}
	return axis.Scale(-1)
}

// retrimFootHost assembles the bottom-cap foot-bite: the through-arm's cross-section arc replaces the
// corner near the bitten vertex, closed by the surviving far path (§B.5 — the reason the bottom cap
// is 1932.47, not the naive quarter-disk 1963.50).
func retrimFootHost(host *topo.Face, segs []endSeg, v math.Point3, w cornerWeld, tol float64) ([]endSeg, bool) {
	pl, ok := host.Geometry().(geom.Plane)
	if !ok {
		return nil, false
	}
	center, radius, ok := footContact(pl, w)
	if !ok {
		return nil, false
	}
	hits := footCircleLoopHits(pl, segs, center, radius, tol)
	if len(hits) != 2 {
		return nil, false // the foot arc must cut the original loop in exactly two points
	}
	rail := footArcSeg(center, radius, pl, hits[0], hits[1], v)
	far, ok := farPathSegs(segs, hits[1], hits[0], v, tol)
	if !ok {
		return nil, false
	}
	return append([]endSeg{rail}, far...), true
}

// footContact finds the through-cylinder arm perpendicular to the cap and returns its cross-section
// circle there (centre = axis∩plane, radius = tube r) — the foot-bite the arm carves on the cap.
func footContact(pl geom.Plane, w cornerWeld) (math.Point3, float64, bool) {
	n, err := math.UnitVector3FromVector(pl.Normal())
	if err != nil {
		return math.Point3{}, 0, false
	}
	for _, a := range w.arms {
		cyl, ok := a.arm.(geom.Cylinder)
		if !ok || !cyl.AxisDir.IsParallelTo(n, retrimAxisParallelTol) {
			continue
		}
		axis, nv := cyl.AxisDir.AsVector(), n.AsVector()
		t := float64(cyl.Origin.VectorTo(pl.Origin).Dot(nv) / axis.Dot(nv))
		return cyl.Origin.TranslateBy(axis.Scale(math.Scalar(t))), cyl.Radius, true
	}
	return math.Point3{}, 0, false
}

// footArcSeg builds the foot-bite arc from h0 to h1 on the circle (centre, radius) in the cap plane,
// picking the sub-arc whose midpoint is NEARER the bitten vertex — the near-corner arc that bounds
// the removed lens, and so also bounds the KEPT region on its interior side (the far/interior-bulging
// sub-arc would carve a much larger region than the fillet removes).
func footArcSeg(center math.Point3, radius float64, pl geom.Plane, h0, h1, v math.Point3) endSeg {
	n, _ := math.UnitVector3FromVector(pl.Normal())
	midDir, err := math.UnitVector3FromVector(center.VectorTo(h0.Midpoint(h1)))
	if err != nil {
		midDir = n // h0,h1 diametric: any in-plane direction works (not reached in Slice A)
	}
	toward := center.TranslateBy(midDir.AsVector().Scale(math.Scalar(radius)))
	away := center.TranslateBy(midDir.AsVector().Scale(math.Scalar(-radius)))
	mid := toward
	if v.DistanceTo(away) < v.DistanceTo(toward) {
		mid = away
	}
	arc, _ := geom.Arc3dByThreePoints(h0, mid, h1)
	return endSeg{from: h0, to: h1, curve: arc, mid: mid, arc: true}
}
