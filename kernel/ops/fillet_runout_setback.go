// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/math"
)

// railParallelFloor is the smallest |n·û| (= sine of the angle between a flank-rail generator and the
// far plane) at which the rail is taken to pierce the plane. Both n and û are unit vectors, so this is
// a DIMENSIONLESS, scale-invariant cutoff (unlike a length tolerance it needs no model-scale factor):
// below it the generator runs parallel to the plane and no finite pierce exists.
const railParallelFloor = 1e-9

// applyRunoutSetback slides each flank tangent rail of a RUNOUT pick edge down its own axis-parallel
// generator to where it PIERCES the adjacent far plane — OCCT's rail termination — instead of running
// out to the pick-edge end vertex's axial projection (today's corner.ta/tb, which overshoots whenever
// the end vertex sits at d>r outside the fillet tube). It rewrites corner.ta/tb (and, at a trihedral
// end, the arc mid) IN PLACE, so the flank-triangle apex (filletMaps), the cylinder rail endpoint
// (cylinderFace), and the fan cap's first/last piece (buildEndCornerFan → boundaryPoint) all derive
// from the SAME corrected corner — the single source that keeps the cylinder loop closed and welded
// twice. Only constant fillets with >=1 fan (>3-valent) end are touched, so a fillet with no runout
// end stays byte-identical (scope guard for the trihedral corpus). Returns an error when a fan flank
// rail is parallel to its far plane (no pierce) — the n-valent analogue of the #1800 reject.
//
// Example: on the occtparity V5 fixture the valence-6 end's rails set back from station 110.38 to the
// OCCT runout vertices RV_3/RV_12 (a 3.21/5.36 axial slide), matching OCCT's blend area.
func applyRunoutSetback(fils []edgeFillet) error {
	for i := range fils {
		if fils[i].varying || !edgeHasFanEnd(fils[i]) {
			continue // scope guard: only constant fillets with a runout end set back
		}
		if err := setbackCorner(&fils[i], &fils[i].c0); err != nil {
			return err
		}
		if err := setbackCorner(&fils[i], &fils[i].c1); err != nil {
			return err
		}
	}
	return nil
}

// edgeHasFanEnd reports whether either end of ef terminates at a >3-valent all-planar runout vertex
// (an endCornerFan) — the scope gate that keeps non-runout fillets out of the setback entirely.
func edgeHasFanEnd(ef edgeFillet) bool {
	_, ok0 := fanForEndCorner(ef, ef.c0)
	_, ok1 := fanForEndCorner(ef, ef.c1)
	return ok0 || ok1
}

// setbackCorner terminates corner c's two flank rails at their far-plane pierce: against the fan's
// A/B-flank far faces at a fan end, or against the single opposite far face at a trihedral end of the
// same runout edge. A non-simple end (blend/miter/runout, or a non-planar far face) is left unmoved.
func setbackCorner(ef *edgeFillet, c *corner) error {
	if fan, ok := fanForEndCorner(*ef, *c); ok {
		return setbackFanCorner(c, fan)
	}
	return setbackTrihedralCorner(ef, c)
}

// setbackFanCorner pierces the A rail against fan[0]'s plane and the B rail against fan[last]'s plane
// (the flank-adjacent far faces the runout chain opens/closes on), through the apex vertex. A parallel
// rail is honest-rejected — the fan must close, so it cannot silently keep the overshoot.
func setbackFanCorner(c *corner, fan endCornerFan) error {
	nA := fan.fan[0].normal
	nB := fan.fan[len(fan.fan)-1].normal
	ta, okA := railPierce(c.ta, fan.axis, fan.apex, nA)
	tb, okB := railPierce(c.tb, fan.axis, fan.apex, nB)
	if !okA || !okB {
		return filletRunoutError(fan, "a flank rail is parallel to its adjacent far plane (no pierce)", fan.filletEdge)
	}
	c.ta, c.tb = ta, tb
	return nil
}

// setbackTrihedralCorner pierces both flank rails of a trihedral end against its single opposite far
// plane (through the corner vertex), then re-seats the arc mid on the cylinder at the pierced
// endpoints' bisector so the end arc stays a cylinder∩plane section. It fires only for a simple
// planar-far end (e.g. V1's (0,0,100) start, at d>r); a non-simple/non-planar end or a rail parallel
// to the plane leaves the corner at today's already-valid termination rather than rejecting a case
// that ships today (unlike the fan end, closure never depends on this move).
func setbackTrihedralCorner(ef *edgeFillet, c *corner) error {
	n, q, ok := opposFarPlane(c)
	if !ok {
		return nil
	}
	axis := ef.cyl.AxisDir.AsVector()
	ta, okA := railPierce(c.ta, axis, q, n)
	tb, okB := railPierce(c.tb, axis, q, n)
	if !okA || !okB {
		return nil
	}
	c.ta, c.tb = ta, tb
	if mid, okM := axisBisectorPoint(c.cen, axis, ef.cyl.Radius, ta, tb); okM {
		c.mid = mid
	}
	return nil
}

// opposFarPlane returns the outward normal and a through-point (the corner vertex, which lies on the
// far plane) of a trihedral end's single opposite far face — the end cap the fillet rounds against.
// ok=false unless the corner is a simple end (not blend/miter/runout) whose end face is planar.
func opposFarPlane(c *corner) (math.Vector3, math.Point3, bool) {
	if c.blend || c.miter || c.runout || c.endFace == nil {
		return math.Vector3{}, math.Point3{}, false
	}
	pl, ok := c.endFace.Geometry().(geom.Plane)
	if !ok {
		return math.Vector3{}, math.Point3{}, false
	}
	return outwardPlaneNormal(c.endFace, pl), c.vertex.Point(), true
}

// railPierce slides rail point a along the fillet axis û to where the generator through a pierces the
// far plane through q with outward normal n: a + t·û, t = n·(q−a)/(n·û). This is OCCT's flank-rail
// termination — the runout vertex lies on every far plane incident to it. ok=false when the generator
// is parallel to the plane (|n·û| < railParallelFloor), i.e. no finite pierce.
func railPierce(a math.Point3, axis math.Vector3, q math.Point3, n math.Vector3) (math.Point3, bool) {
	uhat := probe.Unit(axis)
	denom := n.Dot(uhat)
	if stdmath.Abs(denom) < railParallelFloor {
		return math.Point3{}, false
	}
	t := n.Dot(a.VectorTo(q)) / denom
	return a.TranslateBy(uhat.Scale(t)), true
}
