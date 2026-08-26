// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Point-in-curved-face membership (M2 Phase 1, Oblikovati/Oblikovati#1334). The looped split needs to
// know, for a point on a face's surface, whether it lies inside the face's trimmed region — to pick
// which arc of an imprint conic runs through the face. brep cannot import ops (cycle), so it cannot use
// the H3 tessellation winding number; instead this uses the GEODESIC winding number directly on the
// boundary loops: the signed turning angle of the boundary seen from p, measured around the surface
// normal at p, sums to ≈ ±2π when p is inside and ≈ 0 when outside — with no pole/seam degeneracy (it is
// a 3D tangent-plane measurement, not a (u,v) one).

// windingInsideThreshold splits the geodesic winding sum: inside loops contribute ≈ ±2π, outside ≈ 0, so
// half a turn cleanly separates them.
const windingInsideThreshold = stdmath.Pi

// pointInCurvedFace reports whether p (assumed on f's surface) lies within f's trimmed region. A
// boundary-less face contains every surface point.
func pointInCurvedFace(f curvedFace, p math.Point3) bool {
	if len(f.loops) == 0 {
		return true
	}
	normal := f.surface.NormalAt(f.surface.ParamAt(p))
	if normal.LengthSquared() == 0 {
		return true // a degenerate point (pole/apex): cannot classify, treat as inside
	}
	total := 0.0
	for _, loop := range f.loops {
		total += loopWindingAngle(loop, p, normal)
	}
	// SIGNED, not magnitude: on a closed surface a separating loop winds +2π around the region it
	// encloses (outer loops are CCW about the outward normal) and −2π around the opposite region, so a
	// magnitude test would call BOTH sides of a sphere's equator inside. A hole (CW) subtracts its 2π.
	return total > windingInsideThreshold
}

// loopWindingAngle returns the signed turning angle of a loop's boundary seen from p, summed around the
// surface normal — ≈ +2π if the loop winds CCW around p (p inside it), ≈ 0 if p is outside. Each edge is
// walked in loopSampleCount base steps, and any step that subtends a large angle from p — the signature
// of the boundary passing CLOSE to p, where the turning is rapid — is refined ADAPTIVELY until it does
// not. So a thin feature can no longer slip BETWEEN samples and be mis-counted (the misclassification a
// bare fixed count risks, #1407), while a boundary far from p is summed exactly as before (its base steps
// already subtend little), keeping the established classification on every non-pathological case.
func loopWindingAngle(loop curvedLoop, p math.Point3, normal math.Vector3) float64 {
	sum := 0.0
	for _, le := range loop.edges {
		for k := range loopSampleCount {
			ta := le.t0 + (le.t1-le.t0)*float64(k)/loopSampleCount
			tb := le.t0 + (le.t1-le.t0)*float64(k+1)/loopSampleCount
			sum += edgeWindingAngle(le, p, normal, ta, tb, le.curve.PointAt(ta), le.curve.PointAt(tb), 0)
		}
	}
	return sum
}

// loopSampleCount is the BASE number of steps per loop edge for the winding sum — enough that a curved
// edge's turning is captured; edgeWindingAngle refines below it wherever a step subtends a large angle.
const loopSampleCount = 8

// maxWindingSubtend bounds the angle one step may subtend from p before it is split: keeping every
// contribution small means a near-pass to p cannot be under-counted, so the winding sum is accurate to
// that bound regardless of how close the boundary runs to p (#1407).
const maxWindingSubtend = stdmath.Pi / 6 // 30°

// maxWindingDepth caps the adaptive recursion: a boundary passing exactly through p makes the angle
// ill-defined, so the split bottoms out there rather than spinning.
const maxWindingDepth = 24

// edgeWindingAngle returns the signed turning angle the edge arc from (t0,p0) to (t1,p1) subtends about
// the normal as seen from p, refining the arc by recursive bisection until each piece subtends at most
// maxWindingSubtend (or the depth cap) — exact angle integration in place of fixed-N sampling.
func edgeWindingAngle(le loopEdge, p math.Point3, normal math.Vector3, t0, t1 float64, p0, p1 math.Point3, depth int) float64 {
	a := tangentComponent(p.VectorTo(p0), normal)
	b := tangentComponent(p.VectorTo(p1), normal)
	ang := signedAngleAround(a, b, normal)
	if stdmath.Abs(ang) <= maxWindingSubtend || depth >= maxWindingDepth {
		return ang
	}
	tm := (t0 + t1) / 2
	pm := le.curve.PointAt(tm)
	return edgeWindingAngle(le, p, normal, t0, tm, p0, pm, depth+1) +
		edgeWindingAngle(le, p, normal, tm, t1, pm, p1, depth+1)
}

// tangentComponent projects v onto the plane perpendicular to the unit normal (the surface tangent
// plane), so angles are measured in that plane.
func tangentComponent(v, normal math.Vector3) math.Vector3 {
	return v.Sub(normal.Scale(v.Dot(normal)))
}

// signedAngleAround returns the signed angle from a to b about the axis normal (right-handed), in
// (−π, π]. Zero when either vector is degenerate.
func signedAngleAround(a, b, normal math.Vector3) float64 {
	cross := a.Cross(b).Dot(normal)
	dot := a.Dot(b)
	if a.LengthSquared() == 0 || b.LengthSquared() == 0 {
		return 0
	}
	return stdmath.Atan2(float64(cross), float64(dot))
}
