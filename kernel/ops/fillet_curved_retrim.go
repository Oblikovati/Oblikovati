// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// M5 Slice A, Task 5.3 (m5-weld-setback-retrim-derivation.md §B): the curved-HOST retrim. Each
// CORNER host face at the trihedral corner (the two hosts each arm rolls on — the cylinder wall, the
// top cap, the radial plane) is re-clipped to a single outer loop whose corner bite is bounded by the
// arm CONTACT RAILS. Two of those rails are CIRCULAR arcs the ordinary transformFace (which only pulls
// a straight tangent vertex) cannot emit: the torus arm carves a circle of radius R in the wall (spine
// plane z=C_z) and a circle of radius R−r in the cap (§B.1/B.2). This file emits those exact arcs plus
// the straight rulings, and assembles the retrimmed corner-host loops. The FAR-runout hosts (the y=0
// cut two arms run out to, and the bottom cap the through-arm exits — §B.5) are NOT corner hosts; their
// cross-section bite is spliced by fillet_curved_farrunout.go, not here. T5.4 consumes retrimCurvedHost.

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

// retrimCurvedHost re-clips one CORNER host face at the trihedral corner: the BITTEN loop L* — the
// loop the corner-sphere centre actually lands nearest (bittenLoop), which is the OUTER rim on every
// B3 host but can be an INNER notch window on a boolean-cut wall like N7's — has its original edges
// cut back where the two arms/sphere contact it, plus the arm contact rails spliced in (circular arcs
// for the torus-adjacent hosts, straight rulings for the cylinder/planar-adjacent ones). Every OTHER
// loop on the host (incl. the outer rim, when L* is inner) is untouched by the corner and is carried
// through verbatim (loopsExcept) — generalizes the T5.3 "always retrim the outer loop" assumption
// (C0 / derivation R.0). It honest-rejects (ok=false) when L* is ambiguous (bittenLoop tie), a contact
// rail is missing, or a landing point does not lie on L*'s original edges — never an open or
// self-crossing loop (a mis-closed retrim corrupts the mesh). Example:
//
//	ff, ok := retrimCurvedHost(wallFace, w, res)
//	if !ok { /* decline the weld — do-no-harm */ }
func retrimCurvedHost(host *topo.Face, w cornerWeld, res Resolution) (filletFace, bool) {
	tol := res.Weld() * w.radius
	star, ok := bittenLoop(host, w.center, tol)
	if !ok {
		return filletFace{}, false // no unambiguous bitten loop — do-no-harm
	}
	segs := segsFromLoop(star)
	if len(segs) < 3 {
		return filletFace{}, false // a host loop needs ≥3 edges to bite a corner from
	}
	loop, ok := retrimHostSegs(host, segs, w, res)
	if !ok {
		return filletFace{}, false
	}
	// the bitten loop, retrimmed, first; every other loop (incl. the outer rim on the wall) verbatim.
	loops := append([]filletLoop{loopFromSegs(loop)}, loopsExcept(host, star)...)
	return filletFace{surface: host.Geometry(), loops: loops, parent: host.Lineage()}, true
}

// retrimHostSegs re-clips a CORNER host: the two arms rolling on it meet at a shared host-tangent
// point (wall/cap/radial), and their contact rails bound the bite. The on-edge / on-surface tolerance
// is corner-local (res.Weld·r, ADR-0042) — the corner is a local feature, so a body-diameter tolerance
// would mask a real crack. A host reached by any other count declines: exactly-one arm is not part of
// this trihedral weld, and the far-runout hosts (e.g. the bottom cap the through-arm exits) are NOT
// corner hosts — their cross-section bite is spliced by spliceCornerBite (fillet_curved_farrunout.go),
// not here, so retrimCurvedHost is only ever called on the two-arm corner hosts.
func retrimHostSegs(host *topo.Face, segs []endSeg, w cornerWeld, res Resolution) ([]endSeg, bool) {
	tol := res.Weld() * w.radius
	arms, tHost, n := armsRollingOnHost(host, w, tol)
	if n != 2 {
		return nil, false
	}
	v := bittenVertex(segs, w.center)
	return retrimCornerHost(host, segs, v, arms, tHost, w, res, tol)
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
		outer, ok := armRulingEnd(host, s, arm, tHost, v, segs, tol)
		if !ok {
			return endSeg{}, math.Point3{}, false
		}
		return endSeg{from: outer, to: tHost}, outer, true
	}
	return endSeg{}, math.Point3{}, false
}

// sinFloor is the dimensionless (angular, scale-free) floor for a DEGENERATE ruling: a chart direction
// whose magnitude falls below it means the arm axis is (near-)perpendicular to a planar host — the
// ruling projects to a point and casting it would divide by ~0. Like retrimAxisParallelTol, an angle
// carries no length, so ADR-0042's model-relative rule does not apply; 1e-6 sits far inside the exact
// geometry yet rejects any real misalignment. Never triggers on the corpus (every arm axis lies in / is
// parallel to its host), so it is a defensive decline, not a hot path.
const sinFloor = 1e-6

// armRulingEnd is the far end of a cylinder arm's straight ruling on a host: the FIRST forward crossing
// of the ruling with the (bitten) loop, computed in the host's intrinsic chart (θ,z on a wall; u,v on a
// plane), then cross-checked against the filleted edge's far vertex (the runout authority, R.1a).
// Replaces axialExtremeEnd, which slid to the loop's GLOBAL axial extreme and overshot any intermediate
// trim edge (N7 s_4: global rim z=130, true runout z=80). It honest-rejects (false) when the crossing's
// runout disagrees with the far vertex's beyond tol (the edge ends at an interior weld — out of scope —
// or a wrong edge was hit), or when the chart ruling is degenerate (arm axis ⊥ a planar host).
func armRulingEnd(host *topo.Face, cylArm geom.Cylinder, arm armSetback, tHost, v math.Point3, segs []endSeg, tol float64) (math.Point3, bool) {
	ch, o2, d2, ok := rulingChartRay(host, cylArm, tHost, v)
	if !ok {
		return math.Point3{}, false // no host chart, or the ruling projects to a point (grazing) — decline
	}
	end, ok := chartRulingExit(ch, segs, o2, d2, tol)
	if !ok {
		return math.Point3{}, false // the ruling meets no forward loop edge
	}
	if arm.runoutKnown && !runoutAgrees(ch, o2, d2, end, arm.farVertex, tol) {
		return math.Point3{}, false // crossing runout ≠ edge far-vertex runout: interior weld or wrong edge
	}
	return end, true
}

// rulingChartRay develops the host and returns the arm ruling as a chart ray: origin ch.to2(tHost),
// direction the arm axis oriented away from the bitten vertex, mapped into the chart (θ fixed → vertical
// on a wall; projected axis on a plane). ok=false when the host has no chart or the ruling is degenerate
// (the axis projects to a ~0 chart direction — sinFloor guards the divide chartRulingExit would do).
func rulingChartRay(host *topo.Face, cylArm geom.Cylinder, tHost, v math.Point3) (hostChart, math.Point2, math.Vector2, bool) {
	ch, ok := hostChartFor(host.Geometry())
	if !ok {
		return nil, math.Point2{}, math.Vector2{}, false // host is neither plane nor cylinder
	}
	dir := awayFrom(cylArm.AxisDir.AsVector(), tHost, v)
	o2 := ch.to2(tHost)
	d2 := o2.VectorTo(ch.to2(tHost.TranslateBy(dir)))
	if float64(d2.Length()) < sinFloor { // arm axis ⊥ a planar host: ruling has no chart extent
		return nil, math.Point2{}, math.Vector2{}, false
	}
	return ch, o2, d2, true
}

// runoutAgrees reports whether the crossing end and the far vertex share the same RUNOUT (their extent
// along the ruling direction d2), within tol. It compares only the along-ruling coordinate — NOT the
// full 3D distance — because a fillet's contact ruling is laterally offset from its sharp edge by ~r, so
// the ruling end and the edge far vertex share the runout coordinate (both reach the same rim) but sit
// at different lateral positions; a 3D distance would false-reject by that offset.
func runoutAgrees(ch hostChart, o2 math.Point2, d2 math.Vector2, end, farVertex math.Point3, tol float64) bool {
	unit := d2.Scale(math.Scalar(1 / float64(d2.Length())))
	rEnd := float64(o2.VectorTo(ch.to2(end)).Dot(unit))
	rFar := float64(o2.VectorTo(ch.to2(farVertex)).Dot(unit))
	return stdmath.Abs(rEnd-rFar) <= tol
}

// awayFrom returns axis oriented to point away from the bitten vertex v (the side the ruling runs to).
func awayFrom(axis math.Vector3, tHost, v math.Point3) math.Vector3 {
	if float64(v.VectorTo(tHost).Dot(axis)) >= 0 {
		return axis
	}
	return axis.Scale(-1)
}
