// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"sort"

	"oblikovati.org/math"
)

// Profile extraction — CIRCLE-to-POLYGON conversion (M48 #2242 split of profile.go). Samples a sketch
// circle into the polygon a profile loop carries: the base N-gon, the variant that passes through given
// touch points (so a tangent/coincident circle shares a vertex), and the angle de-duplication. The loop
// tracing and nesting live in profile_trace.go and profile_nesting.go.

// circleSamples is how many points a closed curve is sampled into for containment.
const circleSamples = 24

// sampleCircle returns a polygon approximating a circle.
func sampleCircle(c *Circle) []math.Point2 {
	return sampleCircleN(c, circleSamples)
}

// onCircleTol is how far a point may sit from a circle and still count as touching it. The touch is
// exact in the authored geometry (a line constrained to end on the circle), so this only absorbs
// float noise in re-deriving the radius; it matches the arrangement's own node merge.
const onCircleTol = arrMergeTol // tol:calibrated — point-on-circle touch; see arrMergeTol

// sampleCircleThrough samples a circle at the uniform angles PLUS the angle of every point that
// lies ON it, so a curve ending there shares a vertex with the circle instead of missing it by up
// to a sagitta. It ADDS angles and moves none, so the polygon stays inscribed and no existing
// vertex shifts — a circle nothing touches samples exactly as sampleCircle does.
func sampleCircleThrough(c *Circle, touches []math.Point2) []math.Point2 {
	angles := make([]float64, 0, circleSamples+len(touches))
	for i := range circleSamples {
		angles = append(angles, 2*stdmath.Pi*float64(i)/float64(circleSamples))
	}
	for _, p := range touches {
		if a, ok := circleTouchAngle(c, p); ok {
			angles = append(angles, a)
		}
	}
	return circlePointsAt(c, dedupeAngles(angles))
}

// circleTouchAngle returns p's angle about the circle's centre when p lies on the circle.
func circleTouchAngle(c *Circle, p math.Point2) (float64, bool) {
	dx, dy := float64(p.X-c.Center.X), float64(p.Y-c.Center.Y)
	d := stdmath.Hypot(dx, dy)
	if stdmath.Abs(d-float64(c.Radius)) > onCircleTol {
		return 0, false
	}
	a := stdmath.Atan2(dy, dx)
	if a < 0 {
		a += 2 * stdmath.Pi
	}
	return a, true
}

// dedupeAngles sorts and drops angles closer than the smallest gap worth a vertex, so a touch that
// lands on (or beside) a uniform sample does not add a degenerate zero-length segment.
func dedupeAngles(angles []float64) []float64 {
	sort.Float64s(angles)
	out := angles[:0]
	for i, a := range angles {
		if i == 0 || a-out[len(out)-1] > angleMergeTol {
			out = append(out, a)
		}
	}
	// the wrap-around pair can also collide once the last angle nears 2*pi
	if len(out) > 1 && 2*stdmath.Pi-out[len(out)-1]+out[0] <= angleMergeTol {
		out = out[:len(out)-1]
	}
	return out
}

// angleMergeTol is the angular spacing below which two samples are the same vertex: it is
// arrMergeTol at unit radius, the same grid the arrangement merges nodes on.
const angleMergeTol = arrMergeTol // tol:calibrated — angular sample merge; see arrMergeTol

// circlePointsAt evaluates the circle at each angle.
func circlePointsAt(c *Circle, angles []float64) []math.Point2 {
	pts := make([]math.Point2, len(angles))
	for i, a := range angles {
		pts[i] = math.P2(c.Center.X+c.Radius*math.Scalar(stdmath.Cos(a)), c.Center.Y+c.Radius*math.Scalar(stdmath.Sin(a)))
	}
	return pts
}

// sampleCircleN is sampleCircle at caller-chosen density (region properties
// scale it with the requested accuracy — M06-F08, #623).
func sampleCircleN(c *Circle, n int) []math.Point2 {
	pts := make([]math.Point2, n)
	for i := range pts {
		a := 2 * stdmath.Pi * float64(i) / float64(n)
		pts[i] = math.P2(c.Center.X+c.Radius*stdmath.Cos(a), c.Center.Y+c.Radius*stdmath.Sin(a))
	}
	return pts
}
