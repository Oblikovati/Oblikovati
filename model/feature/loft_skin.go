// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"oblikovati/math"
)

// Loft skinning (the non-naive part). A faithful loft is NOT a straight blend between the user
// sections; it is a smooth surface skinned through them with a consistent point correspondence.
// Two finicky steps, separated here from the meshing:
//
//   1. CORRESPONDENCE (alignSections): the sections are already arc-length-resampled to a common
//      point count; this rotates each section's start to the cyclic offset that best matches the
//      previous one, so corresponding points track across sections instead of twisting (Inventor
//      exposes this as MapPointCurves; the default is this automatic minimum-twist mapping).
//   2. LONGITUDINAL BLEND (splineSections): corresponding points are joined by a Catmull-Rom
//      spline and sampled densely, so a multi-section loft curves smoothly through its interior
//      sections. With only two sections the spline reduces to the straight chord (a 2-section
//      Free-condition loft is ruled in Inventor too) — but it is still sampled densely so a
//      twisted ruled blade renders smooth rather than as a couple of facets.
//
// (Curving a 2-section loft requires end-section tangency conditions — a later slice. Rails,
// centerline and area-graph sections are the other LoftType modes, also later.)

// loftSegmentSamples is how many sub-sections each consecutive section pair is sampled into
// along the blend — the longitudinal resolution that makes the skinned surface read smooth.
const loftSegmentSamples = 8

// alignSections rotates each section's point order to the cyclic start offset that minimizes the
// summed squared distance to the previous section's corresponding points. Winding is assumed
// consistent (sketch profiles are CCW), so it never flips a section — that would invert the
// surface; it only chooses the start point. Sections must already share a point count.
func alignSections(sections [][]math.Point3) [][]math.Point3 {
	if len(sections) < 2 {
		return sections
	}
	out := make([][]math.Point3, len(sections))
	out[0] = sections[0]
	for i := 1; i < len(sections); i++ {
		out[i] = rotateToBestOffset(out[i-1], sections[i])
	}
	return out
}

// rotateToBestOffset returns cur cyclically shifted to the offset minimizing Σ|ref[k]−cur[k]|².
func rotateToBestOffset(ref, cur []math.Point3) []math.Point3 {
	n := len(cur)
	best, bestCost := 0, stdmath.Inf(1)
	for off := 0; off < n; off++ {
		cost := 0.0
		for k := 0; k < n; k++ {
			d := float64(ref[k].DistanceTo(cur[(k+off)%n]))
			cost += d * d
		}
		if cost < bestCost {
			bestCost, best = cost, off
		}
	}
	if best == 0 {
		return cur
	}
	res := make([]math.Point3, n)
	for k := 0; k < n; k++ {
		res[k] = cur[(k+best)%n]
	}
	return res
}

// splineSections inserts loftSegmentSamples Catmull-Rom-interpolated sub-sections between each
// consecutive pair (periodic when closed), so the loft skins a smooth surface through the
// sections. Corresponding points must already be aligned.
func splineSections(sections [][]math.Point3, closed bool) [][]math.Point3 {
	m := len(sections)
	if m < 2 || loftSegmentSamples < 2 {
		return sections
	}
	idx := func(i int) int {
		if closed {
			return ((i % m) + m) % m
		}
		return clampInt(i, 0, m-1)
	}
	segs := m - 1
	if closed {
		segs = m
	}
	out := make([][]math.Point3, 0, 1+segs*loftSegmentSamples)
	out = append(out, sections[0])
	for i := 0; i < segs; i++ {
		p0, p1, p2, p3 := sections[idx(i-1)], sections[idx(i)], sections[idx(i+1)], sections[idx(i+2)]
		for s := 1; s <= loftSegmentSamples; s++ {
			out = append(out, catmullSection(p0, p1, p2, p3, float64(s)/float64(loftSegmentSamples)))
		}
	}
	if closed {
		out = out[:len(out)-1] // the final sample equals sections[0]; drop it (sweptSolid closes the loop)
	}
	return out
}

// extendEnds pushes a hole loft's first and last sections slightly OUTWARD along the loft
// direction, so a bore Cut overshoots the outer body's end caps instead of meeting them
// coplanar — a coplanar-cap Difference leaves the tube open. Open lofts only (eps<=0 is a
// no-op, e.g. for closed lofts which have no ends).
func extendEnds(loops [][]math.Point3, eps float64) [][]math.Point3 {
	m := len(loops)
	if m < 2 || eps <= 0 {
		return loops
	}
	out := make([][]math.Point3, m)
	copy(out, loops)
	out[0] = shiftLoop(loops[0], unitFromTo(loopCentroid(loops[1]), loopCentroid(loops[0])), eps)
	out[m-1] = shiftLoop(loops[m-1], unitFromTo(loopCentroid(loops[m-2]), loopCentroid(loops[m-1])), eps)
	return out
}

// loftOvershoot is the bore-extension distance for a loft of these (outer) sections: a small
// fraction of the loft's end-to-end length (with a floor), enough to clear the coplanar-cap
// degeneracy without being visible.
func loftOvershoot(outers [][]math.Point3) float64 {
	if len(outers) < 2 {
		return 0
	}
	span := float64(loopCentroid(outers[0]).DistanceTo(loopCentroid(outers[len(outers)-1])))
	if e := 0.01 * span; e > 1e-3 {
		return e
	}
	return 1e-3
}

func loopCentroid(loop []math.Point3) math.Point3 {
	var sx, sy, sz float64
	for _, p := range loop {
		sx, sy, sz = sx+float64(p.X), sy+float64(p.Y), sz+float64(p.Z)
	}
	n := float64(len(loop))
	return math.P3(math.Scalar(sx/n), math.Scalar(sy/n), math.Scalar(sz/n))
}

// unitFromTo returns the unit vector from a to b (zero when coincident).
func unitFromTo(a, b math.Point3) math.Vector3 {
	v := a.VectorTo(b)
	l := v.Length()
	if l == 0 {
		return math.V3(0, 0, 0)
	}
	return v.Scale(1 / float64(l))
}

func shiftLoop(loop []math.Point3, dir math.Vector3, d float64) []math.Point3 {
	out := make([]math.Point3, len(loop))
	delta := dir.Scale(d)
	for i, p := range loop {
		out[i] = p.TranslateBy(delta)
	}
	return out
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// catmullSection blends one sub-section: each point is the Catmull-Rom of the four sections'
// corresponding points at parameter t (t∈(0,1] along p1→p2).
func catmullSection(p0, p1, p2, p3 []math.Point3, t float64) []math.Point3 {
	out := make([]math.Point3, len(p1))
	for j := range p1 {
		out[j] = catmullRom3(p0[j], p1[j], p2[j], p3[j], t)
	}
	return out
}

// catmullRom3 is the uniform Catmull-Rom interpolant of four points at t (segment p1→p2). With
// p0==p1 and p3==p2 (the 2-section / end cases) it stays on the straight p1→p2 chord.
func catmullRom3(p0, p1, p2, p3 math.Point3, t float64) math.Point3 {
	t2, t3 := t*t, t*t*t
	c := func(a, b, cc, d float64) math.Scalar {
		return math.Scalar(0.5 * (2*b + (-a+cc)*t + (2*a-5*b+4*cc-d)*t2 + (-a+3*b-3*cc+d)*t3))
	}
	return math.P3(
		c(float64(p0.X), float64(p1.X), float64(p2.X), float64(p3.X)),
		c(float64(p0.Y), float64(p1.Y), float64(p2.Y), float64(p3.Y)),
		c(float64(p0.Z), float64(p1.Z), float64(p2.Z), float64(p3.Z)),
	)
}
