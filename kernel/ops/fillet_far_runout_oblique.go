// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// FR3 — the oblique far termination's authoritative feet + host-rail re-termination (ADR-4).
//
// Under an oblique capping face the two host rails' OUTER ends move (N7's W4 lesson): the perpendicular
// cross-section end is off the oblique cap. The engine fixes the feet closed-form — each is the crossing
// of an arm↔host SPRING (contact) curve with the capping face (armSprings + springCapFoot, FR2, pinned to
// DRAWEXE) — and re-terminates each host rail on its foot. The trim then runs foot→foot (the same feet),
// so trim.endpoints == feet == rail-outer-ends by construction (the shared-edge identity). Everything here
// is a closed form of the arm + host + capping geometry; a foot that does not land on its rail declines.

// armRunoutFeet returns the two oblique runout feet ORDERED to the arm's hosts (feet[0] on ef.a, feet[1]
// on ef.b): armSprings yields the two host-contact springs, springCapFoot crosses each with the capping
// face (near0/near1 — the incoming ⊥ rail ends — select the material-side root), and the pair is mapped
// back to (ef.a, ef.b) so it lines up with the rails h0/h1. Declines when the hosts are not a recognized
// fillet pairing or a spring does not cross the capping.
func armRunoutFeet(ef edgeFillet, capping geom.Surface, near0, near1 math.Point3, r float64, res Resolution) ([2]math.Point3, bool, string) {
	springs, ok := armSprings(ef, ef.a.Geometry(), ef.b.Geometry(), r)
	if !ok {
		return [2]math.Point3{}, false, "oblique runout: armSprings declined the arm's host pairing (not a recognized sphere/plane fillet)"
	}
	springA, springB := springsForHosts(ef, springs)
	footA, okA := springCapFoot(springA, capping, near0, res)
	footB, okB := springCapFoot(springB, capping, near1, res)
	if !okA || !okB {
		return [2]math.Point3{}, false, "oblique runout: a host spring does not cross the capping face (spring∩cap foot declined)"
	}
	return [2]math.Point3{footA, footB}, true, ""
}

// springsForHosts maps the two springs armSprings returns onto the arm's host order (ef.a, ef.b). A canal
// arm's springs come out [plane, cone] (canalArmSprings), so they are re-ordered by which host is the
// geom.Plane; a torus arm's springs come out [sphere, plane] and are re-ordered by which host is the
// sphere; a cylinder arm's springs preserve the (hostA=ef.a, hostB=ef.b) argument order.
func springsForHosts(ef edgeFillet, springs [2]geom.Curve3) (geom.Curve3, geom.Curve3) {
	if ef.armCanalSpine != nil {
		if _, aIsPlane := ef.a.Geometry().(geom.Plane); aIsPlane {
			return springs[0], springs[1] // ef.a is the plane host → springs[0] (plane spring) is on ef.a
		}
		return springs[1], springs[0] // ef.a is the cone host → swap so the cone spring lands on ef.a
	}
	if _, isTorus := ef.armSurface.(geom.Torus); isTorus {
		if _, aIsSphere := ef.a.Geometry().(geom.Sphere); aIsSphere {
			return springs[0], springs[1] // ef.a is the sphere → springs[0] (sphere spring) is on ef.a
		}
		return springs[1], springs[0] // ef.b is the sphere → swap so springA lands on ef.a (the plane)
	}
	return springs[0], springs[1]
}

// reterminateRail re-terminates one host contact rail so its OUTER end lands on the oblique runout foot,
// keeping its inner (setback-tangent) end. A straight ruling (cylinder arm) is re-clipped foot→tHost on
// its own line; a contact arc (torus arm) is re-swept on its own contact circle from the foot's azimuth to
// tHost. Declines (do-no-harm) when the foot does not lie on the rail's underlying line/circle within tol
// — the shared-edge identity (foot == rail outer end == trim endpoint) must hold, never be snapped.
func reterminateRail(rail endSeg, foot math.Point3, tol float64) (endSeg, bool) {
	if !rail.arc {
		if distPointToInfiniteLine(rail.from, rail.to, foot) > tol {
			return endSeg{}, false // foot off the ruling line: the ruling∩cap foot must lie on this rail
		}
		return endSeg{from: foot, to: rail.to}, true
	}
	arc, ok := rail.curve.(geom.Arc3d)
	if !ok {
		return endSeg{}, false
	}
	return reweptArcRail(arc, foot, rail.to, tol)
}

// reweptArcRail re-sweeps a torus arm's contact arc on its OWN contact circle from the foot to tHost,
// keeping the circle (Center/Normal/RefDir/Radius) so the rail still welds with the host retrim that
// shares that circle. Declines when the foot or tHost is off the circle within tol.
func reweptArcRail(arc geom.Arc3d, foot, tHost math.Point3, tol float64) (endSeg, bool) {
	fa, ok0 := arcCircleAngle(arc, foot, tol)
	ta, ok1 := arcCircleAngle(arc, tHost, tol)
	if !ok0 || !ok1 {
		return endSeg{}, false
	}
	sweep := arcSweepInSense(fa, ta, signOf(arc.SweepAngle))
	re, err := geom.NewArc3d(arc.Center, arc.Normal.AsVector(), arc.RefDir.AsVector(), arc.Radius, fa, sweep)
	if err != nil {
		return endSeg{}, false
	}
	return endSeg{from: foot, to: tHost, curve: re, mid: re.PointAt(0.5), arc: true}, true
}

// arcCircleAngle certifies point p lies on the arc's full circle (in the arc plane, at radius) and returns
// its azimuth about the arc centre (the RefDir→binormal frame, matching arcAzimuthInSweep). Declines when
// p is off the plane or off the radius by more than tol.
func arcCircleAngle(arc geom.Arc3d, p math.Point3, tol float64) (float64, bool) {
	w := arc.Center.VectorTo(p)
	nv := arc.Normal.AsVector()
	axial := float64(w.Dot(nv))
	inplane := w.Sub(nv.Scale(math.Scalar(axial)))
	if stdmath.Abs(axial) > tol || stdmath.Abs(float64(inplane.Length())-arc.Radius) > tol {
		return 0, false // p off the arc's circle (out of plane or wrong radius)
	}
	bin := arc.Normal.Cross(arc.RefDir)
	return stdmath.Atan2(float64(inplane.Dot(bin)), float64(inplane.Dot(arc.RefDir.AsVector()))), true
}

// arcSweepInSense is the signed sweep from angle a to angle b turned into the given rotational sense
// (sign of the original arc's sweep), so the re-swept rail runs the same way round its circle as before.
func arcSweepInSense(a, b, sense float64) float64 {
	d := stdmath.Mod(b-a, 2*stdmath.Pi)
	if d*sense < 0 {
		d += sense * 2 * stdmath.Pi
	}
	return d
}

// distPointToInfiniteLine is the perpendicular distance from p to the infinite line through a and b (the
// ruling's outer foot lies BEYOND the segment, so the segment-clamped distPointToSeg would over-report).
func distPointToInfiniteLine(a, b, p math.Point3) float64 {
	d := a.VectorTo(b)
	l := float64(d.Length())
	if l == 0 {
		return float64(p.DistanceTo(a))
	}
	return float64(a.VectorTo(p).Cross(d).Length()) / l
}
