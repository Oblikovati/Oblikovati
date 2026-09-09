// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// CurveBox returns the exact axis-aligned bounding box a curve reaches over its parameter span
// [t0, t1] (given in either order). It bounds each world axis in closed form with [AxialExtent], so a
// conic arc's box covers the arc's true reach and not merely its two ends. ok=false for a curve kind
// with no closed-form extent; the caller must then bound that curve by its own sampling.
//
// A face bounded by ONE closed conic has both ends at the same point, so a box taken from loop
// endpoints alone degenerates to that point — and a resolution derived from it is the model-size
// floor rather than the face's own scale (ADR-0042, ADR-0061 stage 4).
//
// Example:
//
//	box, ok := geom.CurveBox(rim, 0, 1)
func CurveBox(c Curve3, t0, t1 float64) (math.Box, bool) {
	var lo, hi [3]float64
	for i, axis := range [3]math.Vector3{math.V3(1, 0, 0), math.V3(0, 1, 0), math.V3(0, 0, 1)} {
		l, h, ok := AxialExtent(c, t0, t1, math.P3(0, 0, 0), axis)
		if !ok {
			return math.EmptyBox(), false
		}
		lo[i], hi[i] = l, h
	}
	return math.NewBox(math.P3(lo[0], lo[1], lo[2]), math.P3(hi[0], hi[1], hi[2])), true
}

// CurveSpanBox bounds a curve over [t0, t1]. Read the restriction first: where the kind HAS a
// closed-form axial extent this is [CurveBox]'s exact box, and where it has NOT the box is built by
// WALKING the span, so the hull of that walk is an under-bound of the curve's true reach — a helix of
// 6.4 turns walked in curveSpanSamples steps is sampled five times a turn, and its hull is the
// inscribed pentagon, short of the tube everywhere the walk does not land. The walk's hull is
// therefore GROWN by how far the curve can leave the walk between two stations (sampledSpanBox), so
// what this returns is a bound and not a hull.
//
// It is the "the caller must then bound that curve by its own sampling" half of CurveBox's contract,
// written ONCE instead of at each call site — the call site that skipped it (brep's faceLoopBox)
// kept an endpoint-only box, so a face bounded by a single CLOSED non-conic section — the torus
// figure-eight's two lobes, whose section is a SpiricArc — measured its own scale as a POINT and
// took the model-size floor (1e-9) as its resolution. Its stitch weld grid then came out at 1e-15,
// finer than the rounding of the coordinates it compares, and the two readings of the lobes' shared
// pinch point stayed unmerged wherever they were not bit-identical (CI run 34280554924
// macos-latest: arm64 fuses x*y+z, so they differed by 1.3e-15 and the seam tore open).
//
// Example:
//
//	res := geom.ResolutionForBox(geom.CurveSpanBox(lobe, t0, t1))
func CurveSpanBox(c Curve3, t0, t1 float64) math.Box {
	if box, ok := CurveBox(c, t0, t1); ok {
		return box
	}
	return sampledSpanBox(c, t0, t1)
}

// sampledSpanBox bounds a span with no closed-form extent: the walk's hull, grown by the widest step
// reach on it. The grown box is what the walk can PROVE given the speeds it read — the one thing it
// cannot see is a speed spike strictly between two stations, which is the residual a walk of a
// black-box curve always carries and the reason the closed form is taken whenever there is one.
func sampledSpanBox(c Curve3, t0, t1 float64) math.Box {
	box, pad, prev := math.EmptyBox(), 0.0, 0.0
	for i := 0; i <= curveSpanSamples; i++ {
		t := t0 + (t1-t0)*float64(i)/curveSpanSamples
		box = box.ExtendPoint(c.PointAt(t))
		speed := float64(c.TangentAt(t).Length())
		if i > 0 {
			pad = stdmath.Max(pad, stepReach(t1-t0, stdmath.Max(prev, speed)))
		}
		prev = speed
	}
	return grownBox(box, pad)
}

// stepReach is how far the curve can get from the NEARER end of one walk step: half the step's
// parameter width times the fastest it goes on that step. Every point of a step is within a half-step
// of a station, so the hull grown by this covers the step. The pad is the curve's OWN derivative, so
// it carries the curve's units and its scale — there is no constant here to be wrong at another size.
func stepReach(span, speed float64) float64 {
	return stdmath.Abs(span) / curveSpanSamples / 2 * speed
}

// grownBox returns box with every face moved out by d.
func grownBox(box math.Box, d float64) math.Box {
	g := math.Scalar(d)
	return math.NewBox(math.P3(box.Min.X-g, box.Min.Y-g, box.Min.Z-g),
		math.P3(box.Max.X+g, box.Max.Y+g, box.Max.Z+g))
}

// curveSpanSamples is how many steps a span with no closed-form extent is walked in. It is a step
// COUNT, not a tolerance: it trades the walk's cost against the size of the step reach the hull is
// then grown by, and the bound holds at any count.
const curveSpanSamples = 32
