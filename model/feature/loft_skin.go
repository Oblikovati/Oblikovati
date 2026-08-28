// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/math"

	stdmath "math"
)

// Loft skinning (the non-naive part). A faithful loft is NOT a straight blend between the user
// sections; it is a smooth surface skinned through them with a consistent point correspondence.
// Two finicky steps, separated here from the meshing:
//
//   1. CORRESPONDENCE (alignSections): the sections are already arc-length-resampled to a common
//      point count; this rotates each section's start to the cyclic offset that best matches the
//      previous one, so corresponding points track across sections instead of twisting (Inventor
//      exposes this as MapPointCurves; the default is this automatic minimum-twist mapping).
//   2. LONGITUDINAL BLEND (splineSections): corresponding points are joined by a cubic Hermite
//      spline and sampled densely, so a multi-section loft curves smoothly through its interior
//      sections. Interior tangents are Catmull-Rom (half the prev→next chord); the first/last
//      section tangents come from their END CONDITION (loftEnds). A Free end keeps the natural
//      Catmull-Rom tangent, so a two-section Free loft is still ruled (the straight chord) but
//      densely sampled. An Angle/Direction end tilts the takeoff at a chosen angle to the
//      section plane (weighted by an impact), which is what lets a two-section loft curve —
//      flare or neck — away from the ruled blend (e.g. a fan blade).
//
// (Tangent/Smooth conditions need adjacent faces and point-section cases need a tangent plane —
// those, plus rails / centerline / area-graph LoftTypes, are later slices.)

// alignSections corresponds each section to the previous one: it first matches winding (reversing a
// section whose world traversal is opposite the reference — issue #1495), then rotates the section's
// point order to the cyclic start offset minimizing the summed squared distance to the reference's
// corresponding points. Reversal is only applied to an oppositely-wound section, so a consistently-
// wound loft (the common case, and every twisted-but-same-side section such as a Möbius band) is
// never flipped and its surface is not inverted. Sections must already share a point count.
func alignSections(sections [][]math.Point3) [][]math.Point3 {
	if len(sections) < 2 {
		return sections
	}
	out := make([][]math.Point3, len(sections))
	out[0] = sections[0]
	for i := 1; i < len(sections); i++ {
		out[i] = rotateToBestOffset(out[i-1], matchWinding(out[i-1], sections[i]))
	}
	return out
}

// matchWinding reverses cur when its world winding is opposite the reference loop's. Each profile
// loop is sampled in its sketch's 2D order and mapped through the sketch plane, so a circle drawn
// CCW becomes world-CCW on a +Z-normal plane but world-CW on a -Z-normal plane. Two such profiles
// (issue #1495 — a user's two circles on oppositely-facing planes) would otherwise connect rib i of
// one ring to the antipodal point of the next, crossing every facet into a pinched bow-tie (correct-
// looking "valid solid" at ~1/3 the volume). Winding is compared by the loops' Newell normals;
// consistently-wound or degenerate (point/apex, <3 pts) sections are returned unchanged.
func matchWinding(ref, cur []math.Point3) []math.Point3 {
	if len(ref) < 3 || len(cur) < 3 {
		return cur
	}
	if float64(boundaryNormal(ref).Dot(boundaryNormal(cur))) < 0 {
		return reverseLoop(cur)
	}
	return cur
}

// reverseLoop returns the loop traversed the other way, holding index 0 fixed so the subsequent
// start-offset search begins from a stable anchor.
func reverseLoop(p []math.Point3) []math.Point3 {
	n := len(p)
	out := make([]math.Point3, n)
	out[0] = p[0]
	for i := 1; i < n; i++ {
		out[i] = p[n-i]
	}
	return out
}

// rotateToBestOffset returns cur cyclically shifted to the offset minimizing Σ|ref[k]−cur[k]|².
func rotateToBestOffset(ref, cur []math.Point3) []math.Point3 {
	return rotateLoop(cur, bestLoopOffset(ref, cur))
}

// bestLoopOffset is the cyclic shift of cur that minimizes Σ|ref[k]−cur[(k+off)]|² — the start
// offset that best lines cur up with ref. For a CLOSED loft it is also the loop's correspondence
// monodromy: going once around a twisted closed loft, index k returns shifted by this offset (e.g.
// half the points for a 180°-twisted Möbius band, whose rectangular section maps onto itself under
// that shift). The closure (blend + mesh wrap) must apply it, or the wrap crams the whole twist
// into one segment — the seam notch.
func bestLoopOffset(ref, cur []math.Point3) int {
	n := len(cur)
	best, bestCost := 0, stdmath.Inf(1)
	for off := range n {
		cost := 0.0
		for k := range n {
			d := float64(ref[k].DistanceTo(cur[(k+off)%n]))
			cost += d * d
		}
		if cost < bestCost {
			bestCost, best = cost, off
		}
	}
	return best
}

// closureShift is the wrap correspondence offset of a closed section sequence: 0 for an open or
// trivially-closing loft, else the best alignment of the last section back onto the first. It is
// the monodromy the blend and the mesh wrap must apply so a twisted closed loft (a Möbius band)
// closes seamlessly instead of pinching at the seam. A non-twisted closed loft (a full revolve,
// an untwisted ring) returns 0, so existing closed sweeps are unaffected.
func closureShift(sections [][]math.Point3, closed bool) int {
	if !closed || len(sections) < 3 {
		return 0
	}
	return bestLoopOffset(sections[len(sections)-1], sections[0])
}

// rotateVecLoop is rotateLoop for a tangent array (cyclically shifts so index off becomes 0).
func rotateVecLoop(vecs []math.Vector3, off int) []math.Vector3 {
	if off == 0 {
		return vecs
	}
	n := len(vecs)
	out := make([]math.Vector3, n)
	for k := range n {
		out[k] = vecs[(k+off)%n]
	}
	return out
}

// splineSections inserts loftSegmentSamples interpolated sub-sections between each consecutive
// section pair (periodic when closed), so the loft skins a smooth surface through the sections.
// Interior sections take Catmull-Rom tangents; the first and last sections take the tangent
// dictated by their end condition (Free keeps the natural Catmull-Rom tangent, so an all-Free
// loft is the same ruled/curved blend as before — see loftEnds). Corresponding points must
// already be aligned.
func splineSections(sections [][]math.Point3, closed bool, ends loftEnds, wrapShift int) [][]math.Point3 {
	m := len(sections)
	if m < 2 || loftSegmentSamples < 2 {
		return sections
	}
	tan := sectionTangents(sections, closed, ends, wrapShift)
	// Real face continuity (M36-F06): when an end section is a body face and its condition is
	// Tangent/Smooth, override the end tangent with the adjacent surface's actual derivative (G1) and,
	// for Smooth, supply its second derivative so the end segment blends with a quintic that matches
	// the face's curvature (G2). Closed lofts have no end sections, so this is open-only.
	var firstA, lastA, firstJ, lastJ []math.Vector3
	if !closed {
		firstA, firstJ = faceContinuity(tan[0], sections[0], sections[1], ends.firstSurf, ends.first, true)
		lastA, lastJ = faceContinuity(tan[m-1], sections[m-1], sections[m-2], ends.lastSurf, ends.last, false)
	}
	return hermiteBlend(sections, tan, closed, wrapShift, firstA, lastA, firstJ, lastJ)
}

// rotateLoop returns sec cyclically shifted so index off becomes index 0.
func rotateLoop(sec []math.Point3, off int) []math.Point3 {
	if off == 0 {
		return sec
	}
	n := len(sec)
	out := make([]math.Point3, n)
	for k := range n {
		out[k] = sec[(k+off)%n]
	}
	return out
}

// collapsedLoop reports whether every point of a section coincides — a point (apex) section.
func collapsedLoop(sec []math.Point3) bool {
	for _, p := range sec[1:] {
		if p.DistanceTo(sec[0]) > 1e-9 {
			return false
		}
	}
	return true
}

// sectionCentroid averages a section's points.
func sectionCentroid(p []math.Point3) math.Point3 {
	var x, y, z math.Scalar
	for _, q := range p {
		x, y, z = x+q.X, y+q.Y, z+q.Z
	}
	n := math.Scalar(len(p))
	return math.P3(x/n, y/n, z/n)
}
