// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	opstol "oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cone-host corner WELD — the cone-host trihedral-corner campaign, Slice CN4b-2 (the payoff;
// cone-host-corner-derivation.md §3 "corner-face trim" + §4 "machinery-reuse map"). The single-ball weld
// path (armRailBundle → curvedWeldFaces → curvedHostFaces) already carries the torus (CN1) and cylinder
// arms of a cone-host rim corner; this file adds the CANAL (Cone∧Plane ruling) arm's bundle so C2/C6/D1
// close watertight. The canal arm differs from a torus/cylinder arm in exactly two ways: its two host
// contact rails are the exact plane-foot / cone-foot SPRINGS (coneCanalSpring, not an arc or a ruling),
// and its far boundary is CN4b-1's oblique cap (⊥-axis ψ-sweep) or the D1 snout — never a perpendicular
// cross-section. Everything else — the great-circle setback rails, the spherical-triangle corner face
// (chainSetbackArcs), and the host retrims (retrimCornerHost consuming the bundle's hostA/hostB) — is the
// hardened single-ball machinery, reached verbatim. The corner's cone-host rail pinches cleanly at the
// single tangency point T (both cone-side arms touch the cone only there), so no degenerate edge is built.

// canalArmRailBundle assembles the CANAL arm's four boundary rails at the shared corner centre. Unlike the
// torus/cylinder path it does NOT build a perpendicular runout first and re-terminate: the canal far cap is
// always oblique (or the snout), so it fixes the two far feet closed-form ON the springs, builds the far
// cap trim (arm ∩ capping), and trims each host spring from its foot to the setback tangent point in one
// shot. t0 is the tangent point on ef.a, t1 on ef.b (endpointOf(C, r, railDir), the setback-rail ends).
// Returns the non-empty obstruction on any decline — do-no-harm, never a snapped rail.
func canalArmRailBundle(set armSetback, ef edgeFillet, w cornerWeld, filletedEdges map[uint64]bool, res opstol.Resolution) (armRails, string) {
	t0 := endpointOf(w.center, w.radius, set.railDir0) // tangent point on ef.a
	t1 := endpointOf(w.center, w.radius, set.railDir1) // tangent point on ef.b
	setback, ok := setbackEndSeg(w, set, t0, t1)
	if !ok {
		return armRails{}, "canal weld: setback great-circle rail could not be built"
	}
	capping, ok, reason := canalCappingFace(ef, w, filletedEdges, res)
	if !ok {
		return armRails{}, reason
	}
	run, h0, h1, reason := canalFarAndRails(*set.canalSpine, ef, capping, set, t0, t1, w, res)
	if reason != "" {
		return armRails{}, reason
	}
	rails := closeArmRails(h0, h1, setback, run)
	rails.armSurface = reloftCanalOverCorner(*set.canalSpine, ef.edge, set.station, res) // re-loft over [x_f,far, x_f,C]
	return rails, ""
}

// reloftCanalOverCorner re-lofts the canal arm over the CORNER span [x_f,far, x_f,C] instead of the edge
// span, so the setback boundary at x_f,C = stationOf(C) is the surface's exact v-edge (not a curve interior
// to it, which meshes the sliver past the corner — D1 +119). The far end keeps the edge's far station
// (xfSpanLoose's endpoint farther from x_f,C); the near end is x_f,C exactly. Returns nil on any loft
// decline (the arm face then falls back to ef.armSurface — do-no-harm, the pre-CN4b-2 mesh).
func reloftCanalOverCorner(sp coneCanalSpine, e *topo.Edge, xfC float64, res opstol.Resolution) geom.Surface {
	lo, hi := sp.xfSpanLoose(e)
	far := lo
	if stdmath.Abs(hi-xfC) > stdmath.Abs(lo-xfC) {
		far = hi // the edge endpoint farther from the corner station is the far end
	}
	a, b := sortedSpan(far, xfC)
	_, surf, reason := sp.resolveStations(a, b, res)
	if reason != coneArmBuilt {
		return nil
	}
	return surf
}

// canalFarAndRails fixes the two oblique runout feet on the arm's host springs, builds the far cap trim
// (arm ∩ capping, oriented feet[0]→feet[1] on ef.a/ef.b), and trims each host spring from its foot to the
// setback tangent point. h0 is on ef.a (oriented foot→t0), h1 on ef.b (foot→t1); the returned armRunout
// carries the far trim + feet + capping so the far-runout host bite routes by capping identity (FR3).
func canalFarAndRails(sp coneCanalSpine, ef edgeFillet, capping *topo.Face, set armSetback, t0, t1 math.Point3, w cornerWeld, res opstol.Resolution) (armRunout, endSeg, endSeg, string) {
	far := farEndVertex(ef.edge, w.center).Point()
	footA, xfA, okA, rA := coneCanalSpring{spine: sp, onCone: hostIsCone(ef.a)}.canalCapFootStation(capping.Geometry(), far, res)
	footB, xfB, okB, rB := coneCanalSpring{spine: sp, onCone: hostIsCone(ef.b)}.canalCapFootStation(capping.Geometry(), far, res)
	if !okA || !okB {
		return armRunout{}, endSeg{}, endSeg{}, firstReason(rA, rB, "canal weld: a host spring does not cross the far cap")
	}
	feet := [2]math.Point3{footA, footB}
	section, ok := intersectArmCapping(ef, capping.Geometry(), feet, w.radius, res)
	if !ok {
		return armRunout{}, endSeg{}, endSeg{}, capTrimDeclineReason(ef, capping.Geometry(), feet, w.radius, res)
	}
	h0 := canalSpringRail(sp, hostIsCone(ef.a), footA, xfA, t0, set.station)
	h1 := canalSpringRail(sp, hostIsCone(ef.b), footB, xfB, t1, set.station)
	run := armRunout{trim: endSeg{from: footA, to: footB, curve: section, mid: section.PointAt(0.5)}, feet: feet, capping: capping, regime: runoutOblique}
	return run, h0, h1, ""
}

// canalSpringRail is one canal host contact rail as a trimmed coneCanalSpring endSeg oriented outer→tHost:
// PointAt(0) is the far foot (station xfFoot), PointAt(1) is the setback tangent point tHost (station
// xfC = stationOf(C)). The endpoints are pinned to the exact foot/tHost so the neighbour far-cap trim and
// setback rail weld point-for-point. onCone selects the cone-foot vs plane-foot locus.
func canalSpringRail(sp coneCanalSpine, onCone bool, foot math.Point3, xfFoot float64, tHost math.Point3, xfC float64) endSeg {
	spring := coneCanalSpring{spine: sp, lo: xfFoot, hi: xfC, onCone: onCone}
	return endSeg{from: foot, to: tHost, curve: spring}
}

// hostIsCone reports whether host face f is the geom.Cone of a cone-host corner.
func hostIsCone(f *topo.Face) bool {
	_, ok := f.Geometry().(geom.Cone)
	return ok
}

// canalCappingFace finds the face closing the canal arm's far end: the D1 SNOUT cap (a radial plane ⊥ the
// hyperbola-vertex spine tangent) when the arm's far span reaches the spine vertex (x_f→0), else the
// ordinary transverse capping face at the far vertex (cappingFaceAtFarVertex — C2/C6's ⊥-axis cap). The
// snout branch is span-based because the snout cap is ∥ the ruling tangent and the transversality finder is
// structurally blind to it (CN4b-1 review finding #2); it fires ONLY on the canal path (do-no-harm).
func canalCappingFace(ef edgeFillet, w cornerWeld, filletedEdges map[uint64]bool, res opstol.Resolution) (*topo.Face, bool, string) {
	far := farEndVertex(ef.edge, w.center)
	if ef.armCanalSpine != nil && armReachesSpineVertex(*ef.armCanalSpine, far) {
		if cap, ok := snoutCapFace(*ef.armCanalSpine, far, res); ok {
			return cap, true, ""
		}
	}
	return cappingFaceAtFarVertex(far, ef, filletedEdges)
}

// armReachesSpineVertex reports whether the ruling's far endpoint lies past the hyperbola vertex (the ball
// no longer fits: xfAtEndpoint declines) — the D1 snout signature.
func armReachesSpineVertex(sp coneCanalSpine, far *topo.Vertex) bool {
	_, fits := sp.xfAtEndpoint(far.Point())
	return !fits
}

// snoutCapFace returns the radial plane at the far vertex that carries the D1 snout cap: a geom.Plane whose
// normal is ⊥ the axis and aligned with the vertex spine tangent ê (|n̂·ê|≈1) and that contains the vertex
// ball centre m(0). That is exactly canalSnoutTrim's two-condition guard, so the trim it later builds is
// certified. Declines (ok=false) when no such face sits at the far vertex.
func snoutCapFace(sp coneCanalSpine, far *topo.Vertex, res opstol.Resolution) (*topo.Face, bool) {
	m := sp.center(0)
	tol := res.Weld() * sp.radius
	for _, f := range facesAround(far) {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok {
			continue
		}
		n := pl.Normal()
		if stdmath.Abs(float64(n.Dot(sp.axis))) > sinFloor {
			continue // not a radial (axis-parallel) plane
		}
		if 1-stdmath.Abs(float64(n.Dot(sp.ePerp))) > sinFloor {
			continue // not ⊥ the vertex spine tangent ê
		}
		if stdmath.Abs(float64(pl.Origin.VectorTo(m).Dot(n))) <= tol {
			return f, true
		}
	}
	return nil, false
}
