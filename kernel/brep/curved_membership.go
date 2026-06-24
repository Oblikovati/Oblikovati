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
// surface normal — ≈ +2π if the loop winds CCW around p (p inside it), ≈ 0 if p is outside.
func loopWindingAngle(loop curvedLoop, p math.Point3, normal math.Vector3) float64 {
	pts := sampleLoopPoints(loop)
	if len(pts) < 3 {
		return 0
	}
	sum := 0.0
	for i := range pts {
		a := tangentComponent(p.VectorTo(pts[i]), normal)
		b := tangentComponent(p.VectorTo(pts[(i+1)%len(pts)]), normal)
		sum += signedAngleAround(a, b, normal)
	}
	return sum
}

// loopSampleCount is how many points to sample per loop edge for the winding sum — enough that a curved
// edge's turning is captured, cheap enough for the inner loop of the split.
const loopSampleCount = 8

// sampleLoopPoints samples a loop's boundary into ordered 3D points (each edge sampled across its
// parameter range, the shared endpoints deduplicated by skipping each edge's last sample).
func sampleLoopPoints(loop curvedLoop) []math.Point3 {
	var pts []math.Point3
	for _, le := range loop.edges {
		for k := 0; k < loopSampleCount; k++ {
			t := le.t0 + (le.t1-le.t0)*float64(k)/loopSampleCount
			pts = append(pts, le.curve.PointAt(t))
		}
	}
	return pts
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
