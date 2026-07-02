// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"sort"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
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

// loftSegmentSamples is the MINIMUM number of sub-sections each consecutive section pair is
// sampled into along the blend — the baseline longitudinal resolution. A segment that twists or
// curves a lot gets more (see segmentSamples), so a 90° twist reads smooth instead of faceted.
const loftSegmentSamples = 8

// loftMaxStepDeg bounds how much twist/curvature one densified sub-section may span: a segment
// whose point-tracks turn by θ° gets ≈ θ/loftMaxStepDeg sub-sections (never below the floor).
const loftMaxStepDeg = 5.0

// loftMaxSegmentSamples caps the adaptive count so a near-degenerate (very tight) blend cannot
// explode the section count.
const loftMaxSegmentSamples = 64

// loftAroundStepDeg bounds the twist a single across-the-width skin facet may span: a loft
// twisting θ° subdivides each section edge ≈ θ/loftAroundStepDeg times. 15° keeps the warped
// quads' fold below the renderer's crease angle so the skin reads smooth.
const loftAroundStepDeg = 15.0

// loftMaxAroundSubdiv caps the across-width subdivision so an extreme twist cannot explode the
// section point count.
const loftMaxAroundSubdiv = 16

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
	for k := 0; k < n; k++ {
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
		firstA, firstJ = faceContinuity(tan[0], sections[0], sections[1], ends.firstSurf, ends.first)
		lastA, lastJ = faceContinuity(tan[m-1], sections[m-1], sections[m-2], ends.lastSurf, ends.last)
	}
	return hermiteBlend(sections, tan, closed, wrapShift, firstA, lastA, firstJ, lastJ)
}

// continuityOrder maps a face-continuity condition to the derivative order it matches across the
// section edge: Tangent = 1 (G1), Smooth = 2 (G2, curvature), G3 = 3 (curvature-rate). Non-face
// conditions return 0. The canonical mapping lives in api/types.LoftCondition.ContinuityOrder().
func continuityOrder(c LoftCondition) int { return c.ContinuityOrder() }

// faceContinuity overrides an end section's tangents with the adjacent face's real longitudinal
// derivative (true G1, replacing the normal-only approximation) and returns the per-point second
// (G2) and third (G3) derivatives the quintic/septic end-segment blend uses to match the face's
// curvature and curvature-rate. The returned slices are nil below the requested order; both are nil
// when there is no adjacent surface or the condition is not face continuity. The takeoff speed
// follows the existing impact·chord convention so G1 magnitudes are unchanged.
func faceContinuity(tangents []math.Vector3, sec, neighbor []math.Point3, surf geom.Surface, end LoftEnd) (second, third []math.Vector3) {
	order := continuityOrder(end.Condition)
	if surf == nil || order == 0 {
		return nil, nil
	}
	impact := end.Impact
	if impact <= 0 {
		impact = 1
	}
	if order >= 2 {
		second = make([]math.Vector3, len(sec))
	}
	if order >= 3 {
		third = make([]math.Vector3, len(sec))
	}
	c := centroidOf(sec)
	for j := range sec {
		outward := c.VectorTo(sec[j]) // away from the section centroid (the flare side)
		edge := edgeDirAt(sec, j)     // boundary tangent at this point
		t1, g2, g3, ok := faceEndDeriv(surf, sec[j], edge, outward)
		if !ok {
			continue // degenerate surface here — keep the approximate tangent
		}
		speed := impact * float64(sec[j].DistanceTo(neighbor[j])) // c: matches applyFaceTangent's scale
		tangents[j] = t1.Scale(math.Scalar(speed))                // m0 = c · unit cross-boundary tangent
		// P(t)=γ(s(t)) with s=c·t and |t1|=1 ⇒ P^(k)(0)=c^k·γ^(k) — matching geometric curvature (G2)
		// and curvature-rate (G3) at the seam, reparam-invariant.
		if second != nil {
			second[j] = g2.Scale(math.Scalar(speed * speed))
		}
		if third != nil {
			third[j] = g3.Scale(math.Scalar(speed * speed * speed))
		}
	}
	return second, third
}

// faceEndDeriv returns, at the boundary point nearest p, the adjacent face surface's unit
// cross-boundary tangent direction (perpendicular to the boundary edge, in the surface tangent
// plane, pointing outward) and the surface's SECOND and THIRD derivatives along that same direction.
// The loft leaves the face along this tangent (G1) and, for Smooth/G3 ends, with this curvature (G2)
// and curvature-rate (G3) — continuing the real surface. The directional 2nd derivative is analytic
// (includes the mixed term); the 3rd is a central difference of it along the direction, so it is
// exact-to-tolerance without the mixed third partials the kernel does not expose. ok is false at a
// degenerate point.
func faceEndDeriv(surf geom.Surface, p math.Point3, edge, outwardRef math.Vector3) (t1, g2, g3 math.Vector3, ok bool) {
	u, v := surf.ParamAt(p)
	su, sv := surf.DerivativesAt(u, v)
	n := surf.NormalAt(u, v)
	cross := n.Cross(edge) // in-surface direction perpendicular to the boundary edge
	if cross.Dot(outwardRef) < 0 {
		cross = cross.Scale(-1) // orient outward (away from the face interior)
	}
	if cross.Length() < 1e-9 {
		return math.Vector3{}, math.Vector3{}, math.Vector3{}, false
	}
	t1 = cross.Scale(1 / cross.Length())
	// Solve du·Su + dv·Sv = t1 (t1 lies in the tangent plane) via the first fundamental form, so the
	// directional 2nd derivative is exact for this direction.
	e, f, g := su.Dot(su), su.Dot(sv), sv.Dot(sv)
	det := e*g - f*f
	if stdmath.Abs(float64(det)) < 1e-18 {
		return t1, math.Vector3{}, math.Vector3{}, true // degenerate metric ⇒ straight (G2/G3 → 0)
	}
	b1, b2 := t1.Dot(su), t1.Dot(sv)
	du := float64((g*b1 - f*b2) / det)
	dv := float64((e*b2 - f*b1) / det)
	g2 = dirSecond(surf, u, v, du, dv)
	const h = 1e-4 // central-difference step in the param direction for the directional 3rd derivative
	ahead := dirSecond(surf, u+h*du, v+h*dv, du, dv)
	behind := dirSecond(surf, u-h*du, v-h*dv, du, dv)
	g3 = ahead.Sub(behind).Scale(math.Scalar(1 / (2 * h)))
	return t1, g2, g3, true
}

// dirSecond is the surface's second derivative at (u,v) along the param direction (du,dv):
// du²·Suu + 2·du·dv·Suv + dv²·Svv (the analytic directional 2nd derivative, mixed term included).
func dirSecond(surf geom.Surface, u, v, du, dv float64) math.Vector3 {
	puu, puv, pvv := geom.SurfaceSecondPartials(surf, u, v)
	return puu.Scale(math.Scalar(du * du)).Add(puv.Scale(math.Scalar(2 * du * dv))).Add(pvv.Scale(math.Scalar(dv * dv)))
}

// railGuide deforms the densified sections so the loft follows guide rails (the kLoftWithRails
// mode): each rail pulls the nearest point-track onto it, with a smooth angular falloff to the
// neighbouring tracks, so the loft bulges toward the rail BETWEEN sections while the end sections
// stay put (the rail's ends are snapped to the section corner it tracks). Rails are model-space
// polylines; multiple rails' pulls sum. A no-op without rails.
func railGuide(sections [][]math.Point3, rails [][]math.Point3) [][]math.Point3 {
	levels := len(sections)
	if len(rails) == 0 || levels < 2 {
		return sections
	}
	n := len(sections[0])
	// Every rail's pull is measured against the ORIGINAL sections and the pulls are summed before
	// being applied, so multiple rails don't cross-talk (a sequential apply would let one rail's
	// deformation skew the next rail's measured displacement).
	disp := make([][]math.Vector3, levels)
	for s := range disp {
		disp[s] = make([]math.Vector3, n)
	}
	for _, rail := range rails {
		if len(rail) < 2 {
			continue
		}
		samples := resamplePath(rail, levels)
		jr := nearestTrack(sections[0], samples[0])
		samples[0], samples[levels-1] = sections[0][jr], sections[levels-1][jr] // pin the ends
		for s := 0; s < levels; s++ {
			d := sections[s][jr].VectorTo(samples[s])
			for j := 0; j < n; j++ {
				disp[s][j] = disp[s][j].Add(d.Scale(railFalloff(j, jr, n)))
			}
		}
	}
	for s := 0; s < levels; s++ {
		for j := 0; j < n; j++ {
			sections[s][j] = sections[s][j].TranslateBy(disp[s][j])
		}
	}
	return sections
}

// mapAlign sets the point correspondence. With no map curves it falls back to the automatic
// minimum-twist alignment (alignSections). With explicit map curves (the MapPointCurves mode) the
// first curve gives one anchor point per section; each section is rotated so the vertex nearest its
// anchor becomes index 0, so the anchors correspond across sections — letting the user pin (or
// deliberately twist) the correspondence the auto-alignment would otherwise choose.
func mapAlign(sections [][]math.Point3, mapCurves [][]math.Point3) [][]math.Point3 {
	if len(mapCurves) == 0 || len(mapCurves[0]) == 0 {
		return alignSections(sections)
	}
	mc := mapCurves[0]
	out := make([][]math.Point3, len(sections))
	for i, sec := range sections {
		if i >= len(mc) {
			out[i] = sec
			continue
		}
		out[i] = rotateLoop(sec, nearestTrack(sec, mc[i]))
	}
	return out
}

// rotateLoop returns sec cyclically shifted so index off becomes index 0.
func rotateLoop(sec []math.Point3, off int) []math.Point3 {
	if off == 0 {
		return sec
	}
	n := len(sec)
	out := make([]math.Point3, n)
	for k := 0; k < n; k++ {
		out[k] = sec[(k+off)%n]
	}
	return out
}

// areaGraphScale resizes each densified section about its centroid so the loft's cross-sectional
// area follows the area graph (the kLoftWithAreaGraphSections mode): at longitudinal fraction t the
// section is scaled by sqrt(graph(t)), so its area becomes graph(t)× the blended area. The end
// sections (t=0,1) are pinned to scale 1 so the user's profiles are preserved. No-op without stops.
func areaGraphScale(sections [][]math.Point3, stops []types.LoftAreaStop) [][]math.Point3 {
	levels := len(sections)
	if len(stops) == 0 || levels < 2 {
		return sections
	}
	graph := pinnedAreaStops(stops)
	for s := 1; s < levels-1; s++ { // ends stay scale 1 (the real sections)
		k := stdmath.Sqrt(areaGraphAt(float64(s)/float64(levels-1), graph))
		if k == 1 {
			continue
		}
		c := centroidOf(sections[s])
		for j := range sections[s] {
			sections[s][j] = c.TranslateBy(c.VectorTo(sections[s][j]).Scale(k))
		}
	}
	return sections
}

// pinnedAreaStops returns the stops sorted by T with implicit scale-1 endpoints added at T=0/1 (so
// the graph pins to the unscaled end sections unless the user overrode those positions).
func pinnedAreaStops(stops []types.LoftAreaStop) []types.LoftAreaStop {
	out := append([]types.LoftAreaStop(nil), stops...)
	has := func(t float64) bool {
		for _, s := range out {
			if stdmath.Abs(s.T-t) < 1e-9 {
				return true
			}
		}
		return false
	}
	if !has(0) {
		out = append(out, types.LoftAreaStop{T: 0, Scale: 1})
	}
	if !has(1) {
		out = append(out, types.LoftAreaStop{T: 1, Scale: 1})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].T < out[j].T })
	return out
}

// areaGraphAt linearly interpolates the area scale at fraction t over the (sorted, end-pinned)
// stops; clamped to the first/last stop outside their range.
func areaGraphAt(t float64, sorted []types.LoftAreaStop) float64 {
	if t <= sorted[0].T {
		return sorted[0].Scale
	}
	last := sorted[len(sorted)-1]
	if t >= last.T {
		return last.Scale
	}
	for i := 1; i < len(sorted); i++ {
		if t <= sorted[i].T {
			a, b := sorted[i-1], sorted[i]
			f := (t - a.T) / (b.T - a.T)
			return a.Scale + f*(b.Scale-a.Scale)
		}
	}
	return last.Scale
}

// centerlineGuide bends the loft's spine (the kLoftWithCenterline mode): each densified
// sub-section is translated bodily so its centroid lies on the centerline curve, so the loft
// follows a curved path while keeping its cross-sections. Unlike a rail (which pulls one side with
// falloff), the whole section moves. The centerline's ends are pinned to the end-section
// centroids. A no-op without a centerline.
func centerlineGuide(sections [][]math.Point3, centerline []math.Point3) [][]math.Point3 {
	levels := len(sections)
	if len(centerline) < 2 || levels < 2 {
		return sections
	}
	samples := resamplePath(centerline, levels)
	samples[0], samples[levels-1] = centroidOf(sections[0]), centroidOf(sections[levels-1]) // pin ends
	for s := 0; s < levels; s++ {
		d := centroidOf(sections[s]).VectorTo(samples[s])
		for j := range sections[s] {
			sections[s][j] = sections[s][j].TranslateBy(d)
		}
	}
	return sections
}

// nearestTrack returns the index of the section point closest to p.
func nearestTrack(section []math.Point3, p math.Point3) int {
	best, bestD := 0, stdmath.Inf(1)
	for j, q := range section {
		if d := float64(q.DistanceTo(p)); d < bestD {
			best, bestD = j, d
		}
	}
	return best
}

// railFalloff weights a rail's pull on track j: 1 at the rail's track jr, falling on a raised
// cosine to 0 at the diametrically opposite track (a localized, smooth bulge).
func railFalloff(j, jr, n int) float64 {
	d := j - jr
	if d < 0 {
		d = -d
	}
	if n-d < d {
		d = n - d
	}
	frac := float64(d) / (float64(n) / 2)
	if frac >= 1 {
		return 0
	}
	return 0.5 * (1 + stdmath.Cos(stdmath.Pi*frac))
}

// resamplePath returns count points spaced evenly by arc length along an OPEN polyline (endpoints
// included). A zero-length path yields count copies of its first point.
func resamplePath(poly []math.Point3, count int) []math.Point3 {
	if count < 2 || len(poly) < 2 {
		return poly
	}
	segLen := make([]float64, len(poly)-1)
	total := 0.0
	for i := range segLen {
		segLen[i] = float64(poly[i].DistanceTo(poly[i+1]))
		total += segLen[i]
	}
	out := make([]math.Point3, count)
	if total == 0 {
		for k := range out {
			out[k] = poly[0]
		}
		return out
	}
	out[0], out[count-1] = poly[0], poly[len(poly)-1]
	step, seg, acc := total/float64(count-1), 0, 0.0
	for k := 1; k < count-1; k++ {
		target := step * float64(k)
		for seg < len(segLen)-1 && acc+segLen[seg] < target {
			acc += segLen[seg]
			seg++
		}
		f := 0.0
		if segLen[seg] > 0 {
			f = (target - acc) / segLen[seg]
		}
		out[k] = poly[seg].TranslateBy(poly[seg].VectorTo(poly[seg+1]).Scale(f))
	}
	return out
}

// sectionTangents is the longitudinal tangent at every section point: a Catmull-Rom tangent
// (half the chord from the previous to the next section) at interior sections, overridden at
// the first/last section by an angled end condition. Feeding these to a Hermite blend with the
// Catmull-Rom tangents reproduces the Catmull-Rom curve exactly, so Free ends are unchanged.
func sectionTangents(sections [][]math.Point3, closed bool, ends loftEnds, wrapShift int) [][]math.Vector3 {
	m, n := len(sections), len(sections[0])
	idx := func(i int) int {
		if closed {
			return ((i % m) + m) % m
		}
		return math.Clamp(i, 0, m-1)
	}
	// Across the closed seam (between section m-1 and 0) the correspondence is offset by the
	// monodromy wrapShift, so a periodic neighbour's matching point is reindexed by it. Without this
	// the Catmull-Rom tangent at the seam sections points across the cross-section (a kink/crease at
	// the seam) instead of along the loop.
	corr := func(j, shift int) int { return ((j+shift)%n + n) % n }
	tan := make([][]math.Vector3, m)
	for i := 0; i < m; i++ {
		tan[i] = make([]math.Vector3, n)
		predShift, succShift := 0, 0
		if closed && i == 0 {
			predShift = -wrapShift // stepping back to section m-1 crosses the seam
		}
		if closed && i == m-1 {
			succShift = wrapShift // stepping forward to section 0 crosses the seam
		}
		for j := 0; j < n; j++ {
			p := sections[idx(i-1)][corr(j, predShift)]
			q := sections[idx(i+1)][corr(j, succShift)]
			tan[i][j] = p.VectorTo(q).Scale(0.5)
		}
	}
	if !closed {
		applyEnd(tan[0], sections, 0, 1, ends.first, ends.firstN)
		applyEnd(tan[m-1], sections, m-1, m-2, ends.last, ends.lastN)
	}
	return tan
}

// applyEnd overrides one end section's tangents from its condition, dispatching on whether the
// section is a point (apex) or a profile. endIdx is the end section, neighIdx its neighbour.
func applyEnd(tangents []math.Vector3, sections [][]math.Point3, endIdx, neighIdx int, end LoftEnd, normal math.UnitVector3) {
	sec, neighbor := sections[endIdx], sections[neighIdx]
	if collapsedLoop(sec) {
		applyApexCondition(tangents, sec, neighbor, end, normal, endIdx < neighIdx)
		return
	}
	var forward math.Vector3 // the +section (increasing-index) direction at this end
	if endIdx < neighIdx {
		forward = loopCentroid(sec).VectorTo(loopCentroid(neighbor))
	} else {
		forward = loopCentroid(neighbor).VectorTo(loopCentroid(sec))
	}
	applyEndCondition(tangents, sec, neighbor, end, normal, forward)
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

// applyApexCondition shapes how the loft meets a point (apex) section. Sharp/Free keep the
// natural Catmull-Rom tangent — a straight taper (a cone). TangentToPlane makes the surface leave
// the apex tangent to its plane (normal), so each meridian's apex tangent lies in that plane,
// pointing along the neighbour point's radial direction — a rounded dome. Impact scales the dome
// reach; Reversed flips it to a concave (dished) apex.
func applyApexCondition(tangents []math.Vector3, apexSec, neighbor []math.Point3, end LoftEnd, normal math.UnitVector3, isStart bool) {
	if !end.Condition.IsTangentToPlane() {
		return
	}
	impact := end.Impact
	if impact <= 0 {
		impact = 1
	}
	sign := 1.0 // a start apex leaves outward (+u toward the neighbour); an end apex arrives inward
	if !isStart {
		sign = -1
	}
	if end.Reversed {
		sign = -sign
	}
	apex, nc := apexSec[0], centroidOf(neighbor)
	for j := range apexSec {
		r := radialDir(neighbor[j], nc, normal)
		chord := float64(apex.DistanceTo(neighbor[j]))
		tangents[j] = unitOrFallback(r.Scale(sign), normal.AsVector()).Scale(impact * chord)
	}
}

// applyEndCondition overrides a profile end section's tangents from its condition: an
// angle/direction takeoff (a chosen angle to the section plane) or a face-continuity takeoff
// (Tangent/Smooth — leave the source face tangent to its surface). Free keeps the natural
// Catmull-Rom tangent. neighbor is the adjacent section, normal the section's plane/face normal,
// forward the +u (increasing-section) direction here.
func applyEndCondition(tangents []math.Vector3, sec, neighbor []math.Point3, end LoftEnd, normal math.UnitVector3, forward math.Vector3) {
	switch {
	case end.Condition.CurvesViaAngle():
		applyAngleTakeoff(tangents, sec, neighbor, end, normal, forward)
	case end.Condition.IsFaceContinuity():
		applyFaceTangent(tangents, sec, neighbor, end, normal)
	}
}

// applyAngleTakeoff aims each tangent at end.Angle from the section plane (sin·normal + cos·
// radial), scaled by impact·chord; Reversed flips the through-plane component (an undercut).
func applyAngleTakeoff(tangents []math.Vector3, sec, neighbor []math.Point3, end LoftEnd, normal math.UnitVector3, forward math.Vector3) {
	nf := alignToward(normal, forward)
	if end.Reversed {
		nf = nf.Negate()
	}
	impact := end.Impact
	if impact <= 0 {
		impact = 1
	}
	sa, ca := stdmath.Sin(end.Angle), stdmath.Cos(end.Angle)
	c := centroidOf(sec)
	for j := range sec {
		r := radialDir(sec[j], c, normal)
		base := float64(sec[j].DistanceTo(neighbor[j]))
		dir := nf.AsVector().Scale(sa).Add(r.Scale(ca))
		tangents[j] = unitOrFallback(dir, forward).Scale(impact * base)
	}
}

// applyFaceTangent makes the loft leave the source face tangent to its surface: each tangent is
// the in-surface direction perpendicular to the boundary edge (normal × edgeDir), oriented
// outward, scaled by impact·chord. For a planar source face the loft's tangent plane along the
// shared edge equals the face plane — exact G1 continuity. Smooth (G2) reuses this as a faceted-
// kernel approximation. Reversed flips the takeoff inward.
func applyFaceTangent(tangents []math.Vector3, sec, neighbor []math.Point3, end LoftEnd, normal math.UnitVector3) {
	impact := end.Impact
	if impact <= 0 {
		impact = 1
	}
	c, n := centroidOf(sec), normal.AsVector()
	for j := range sec {
		t := n.Cross(edgeDirAt(sec, j))
		if t.Dot(c.VectorTo(sec[j])) < 0 { // orient outward (away from the section centroid)
			t = t.Negate()
		}
		if end.Reversed {
			t = t.Negate()
		}
		base := float64(sec[j].DistanceTo(neighbor[j]))
		tangents[j] = unitOrFallback(t, c.VectorTo(sec[j])).Scale(impact * base)
	}
}

// edgeDirAt is the boundary direction at loop point j (the chord from the previous to the next).
func edgeDirAt(loop []math.Point3, j int) math.Vector3 {
	n := len(loop)
	return loop[(j-1+n)%n].VectorTo(loop[(j+1)%n])
}

// alignToward returns n flipped, if needed, to point into the same half-space as ref.
func alignToward(n math.UnitVector3, ref math.Vector3) math.UnitVector3 {
	if n.AsVector().Dot(ref) < 0 {
		return n.Negate()
	}
	return n
}

// radialDir is the outward in-plane unit direction from the section centroid c to point p
// (zero when p sits on the section axis, e.g. a point section).
func radialDir(p, c math.Point3, normal math.UnitVector3) math.Vector3 {
	nrm := normal.AsVector()
	v := c.VectorTo(p)
	v = v.Sub(nrm.Scale(v.Dot(nrm)))
	l := v.Length()
	if l < 1e-12 {
		return math.V3(0, 0, 0)
	}
	return v.Scale(1 / l)
}

// unitOrFallback returns the unit of v, or the unit of fallback when v is ~zero.
func unitOrFallback(v, fallback math.Vector3) math.Vector3 {
	if v.Length() < 1e-12 {
		v = fallback
	}
	l := v.Length()
	if l == 0 {
		return math.V3(0, 0, 0)
	}
	return v.Scale(1 / l)
}

// hermiteBlend samples a cubic Hermite spline through each corresponding-point track, using the
// per-section tangents, into loftSegmentSamples sub-sections per segment (periodic when closed).
// For a closed loft with a correspondence monodromy (wrapShift != 0 — a twisted band such as a
// Möbius), the closing segment blends toward the start section REINDEXED by wrapShift, so the wrap
// is a small twist (one step) instead of cramming the whole accumulated twist into one segment.
// sweptSolid's mesh wrap applies the same shift, so the two stay consistent.
// firstA/lastA are the per-point second derivatives at the first/last section for a curvature (G2)
// face-continuity end (nil otherwise): the first segment then blends with a QUINTIC Hermite that
// matches firstA at its start, the last segment one that matches lastA at its end, so the loft
// continues the adjacent face's curvature across the seam. The interior side of each such segment
// keeps a natural (zero) second derivative.
// firstA/lastA and firstJ/lastJ are the per-point second (G2) and third (G3) derivatives at the
// first/last section for a curvature/curvature-rate face-continuity end (nil otherwise): the first
// segment then blends with a quintic (G2) or septic (G3) Hermite matching them at its start, the last
// segment one matching them at its end, so the loft continues the adjacent face's curvature (and
// curvature-rate) across the seam. The interior side of each such segment stays natural (zero).
func hermiteBlend(sections [][]math.Point3, tan [][]math.Vector3, closed bool, wrapShift int, firstA, lastA, firstJ, lastJ []math.Vector3) [][]math.Point3 {
	m := len(sections)
	segs := m - 1
	if closed {
		segs = m
	}
	out := make([][]math.Point3, 0, 1+segs*loftSegmentSamples)
	out = append(out, sections[0])
	for i := 0; i < segs; i++ {
		p1, t1 := sections[(i+1)%m], tan[(i+1)%m]
		if closed && i == segs-1 && wrapShift != 0 { // the wrap: aim at the start reindexed by the monodromy
			p1, t1 = rotateLoop(sections[0], wrapShift), rotateVecLoop(tan[0], wrapShift)
		}
		var startA, endA, startJ, endJ []math.Vector3
		if i == 0 {
			startA, startJ = firstA, firstJ
		}
		if i == segs-1 && !closed {
			endA, endJ = lastA, lastJ
		}
		n := segmentSamplesG2(sections[i], p1, tan[i], t1, startA, endA, startJ, endJ)
		for s := 1; s <= n; s++ {
			out = append(out, blendSection(sections[i], p1, tan[i], t1, startA, endA, startJ, endJ, float64(s)/float64(n)))
		}
	}
	if closed {
		out = out[:len(out)-1] // the final sample equals the (reindexed) start; drop it (sweptSolid closes the loop)
	}
	return out
}

// blendSection blends one sub-section: a cubic Hermite by default; a quintic when a G2 second
// derivative is supplied at either end (startA/endA); a septic when a G3 third derivative is supplied
// (startJ/endJ). A nil higher-derivative end matches a natural (zero) value there.
func blendSection(p0, p1 []math.Point3, m0, m1, startA, endA, startJ, endJ []math.Vector3, t float64) []math.Point3 {
	if startA == nil && endA == nil {
		return hermiteSection(p0, p1, m0, m1, t)
	}
	out := make([]math.Point3, len(p0))
	g3 := startJ != nil || endJ != nil
	for j := range p0 {
		if g3 {
			out[j] = hermite7(p0[j], p1[j], m0[j], m1[j], vecAt(startA, j), vecAt(endA, j), vecAt(startJ, j), vecAt(endJ, j), t)
			continue
		}
		out[j] = hermite5(p0[j], p1[j], m0[j], m1[j], vecAt(startA, j), vecAt(endA, j), t)
	}
	return out
}

// vecAt returns a[j], or the zero vector when a is nil (a natural, unconstrained derivative).
func vecAt(a []math.Vector3, j int) math.Vector3 {
	if a == nil {
		return math.Vector3{}
	}
	return a[j]
}

// segmentSamplesG2 is segmentSamples, but probes the actual blend (quintic/septic when a G2/G3
// derivative is present) so a curvature-matched end segment that bends more than its chord gets
// enough sub-sections to read smooth.
func segmentSamplesG2(p0, p1 []math.Point3, m0, m1, startA, endA, startJ, endJ []math.Vector3) int {
	if startA == nil && endA == nil {
		return segmentSamples(p0, p1, m0, m1)
	}
	const probes = 12
	sec := make([][]math.Point3, probes+1)
	for s := 0; s <= probes; s++ {
		sec[s] = blendSection(p0, p1, m0, m1, startA, endA, startJ, endJ, float64(s)/float64(probes))
	}
	turn := stdmath.Max(segmentTwist(p0, p1), maxTrackTurn(sec))
	n := int(stdmath.Ceil(turn / (loftMaxStepDeg * stdmath.Pi / 180)))
	if n < loftSegmentSamples {
		n = loftSegmentSamples
	}
	if n > loftMaxSegmentSamples {
		n = loftMaxSegmentSamples
	}
	return n
}

// segmentSamples is how many sub-sections to blend a section pair into: at least
// loftSegmentSamples, more when the segment twists or curves so each longitudinal facet spans no
// more than loftMaxStepDeg. It accounts for the two ways a lofted segment is non-planar:
//   - twist: the cross-section ROTATES about the loft axis (rulings stay straight, so a 90° twist
//     between two squares looks faceted) — measured as the max rotation of a point about the axis;
//   - curvature: the corresponding-point tracks bend (a tangent/smooth end condition) — measured
//     by probing each Hermite track's turning.
//
// A 90° twist → ~18 sub-sections (smooth); a straight, ruled segment → the floor.
func segmentSamples(p0, p1 []math.Point3, m0, m1 []math.Vector3) int {
	turn := stdmath.Max(segmentTwist(p0, p1), segmentTrackTurn(p0, p1, m0, m1))
	n := int(stdmath.Ceil(turn / (loftMaxStepDeg * stdmath.Pi / 180)))
	if n < loftSegmentSamples {
		n = loftSegmentSamples
	}
	if n > loftMaxSegmentSamples {
		n = loftMaxSegmentSamples
	}
	return n
}

// aroundSubdivisions is how many times to split each section edge across the loft's width: 1
// (no extra) for an untwisted loft, more as the loft twists, so the skin's warped quads stay
// narrow enough to read smooth (a wide quad on a twisting surface folds steeply when split into
// triangles, regardless of longitudinal density). Proportional to the max inter-section twist.
func aroundSubdivisions(sections [][]math.Point3, closed bool, wrapShift int) int {
	if len(sections) < 2 {
		return 1
	}
	var maxTwist float64
	for i := 0; i+1 < len(sections); i++ {
		if a := segmentTwist(sections[i], sections[i+1]); a > maxTwist {
			maxTwist = a
		}
	}
	if closed {
		// Measure the wrap twist against the start REINDEXED by the monodromy (wrapShift); otherwise
		// a twisted/non-orientable closure (a Möbius band) reads its seam as the full accumulated
		// twist (~180°) and over-subdivides EVERY cross-section to "smooth" a twist that does not
		// happen between adjacent sections — a 12× mesh blow-up on the ellipse Möbius.
		if a := segmentTwist(sections[len(sections)-1], rotateLoop(sections[0], wrapShift)); a > maxTwist {
			maxTwist = a
		}
	}
	k := int(stdmath.Ceil(maxTwist / (loftAroundStepDeg * stdmath.Pi / 180)))
	if k < 1 {
		k = 1
	}
	if k > loftMaxAroundSubdiv {
		k = loftMaxAroundSubdiv
	}
	return k
}

// densifyAround subdivides every section edge into k, preserving the original vertices (corners)
// and inserting collinear points along each edge — so the section shape (and volume) is
// unchanged, only sampled finer across its width. All sections subdivide identically, preserving
// the point correspondence the blend relies on. k≤1 is a no-op.
func densifyAround(sections [][]math.Point3, k int) [][]math.Point3 {
	if k <= 1 {
		return sections
	}
	out := make([][]math.Point3, len(sections))
	for s, sec := range sections {
		m := len(sec)
		if m < 2 {
			out[s] = sec // a point/degenerate section — nothing to subdivide
			continue
		}
		dense := make([]math.Point3, 0, m*k)
		for j := 0; j < m; j++ {
			a, b := sec[j], sec[(j+1)%m]
			for t := 0; t < k; t++ {
				f := math.Scalar(float64(t) / float64(k))
				dense = append(dense, a.TranslateBy(a.VectorTo(b).Scale(f)))
			}
		}
		out[s] = dense
	}
	return out
}

// segmentTwist is the largest angle by which a section point rotates about the loft axis
// (centroid c0→c1) going from p0 to p1 — the cross-section's twist over the segment.
func segmentTwist(p0, p1 []math.Point3) float64 {
	c0, c1 := sectionCentroid(p0), sectionCentroid(p1)
	axis := unit3(c0.VectorTo(c1))
	var maxTwist float64
	for j := range p0 {
		r0 := perpTo(c0.VectorTo(p0[j]), axis)
		r1 := perpTo(c1.VectorTo(p1[j]), axis)
		if a := angleBetween(r0, r1); a > maxTwist {
			maxTwist = a
		}
	}
	return maxTwist
}

// segmentTrackTurn is the most any corresponding-point Hermite track bends across the segment
// (zero for straight rulings, nonzero for a curved/tangent blend).
func segmentTrackTurn(p0, p1 []math.Point3, m0, m1 []math.Vector3) float64 {
	const probes = 12
	sec := make([][]math.Point3, probes+1)
	for s := 0; s <= probes; s++ {
		sec[s] = hermiteSection(p0, p1, m0, m1, float64(s)/float64(probes))
	}
	return maxTrackTurn(sec)
}

// maxTrackTurn is the most any corresponding-point track bends across a sequence of probed
// sub-sections — the shared turning measure for cubic and quintic segment sampling.
func maxTrackTurn(sec [][]math.Point3) float64 {
	probes := len(sec) - 1
	var maxTurn float64
	for j := range sec[0] {
		var turn float64
		for s := 1; s < probes; s++ {
			turn += turnAngle(sec[s-1][j], sec[s][j], sec[s+1][j])
		}
		if turn > maxTurn {
			maxTurn = turn
		}
	}
	return maxTurn
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

// perpTo returns the component of v perpendicular to the unit axis a.
func perpTo(v math.Vector3, a math.Vector3) math.Vector3 {
	return v.Add(a.Scale(-v.Dot(a)))
}

// unit3 normalizes v, or returns +Z when degenerate (coincident centroids).
func unit3(v math.Vector3) math.Vector3 {
	if l := v.Length(); l > 1e-12 {
		return v.Scale(1 / l)
	}
	return math.V3(0, 0, 1)
}

// angleBetween is the unsigned angle (radians) between two vectors, 0 if either is degenerate.
func angleBetween(a, b math.Vector3) float64 {
	la, lb := float64(a.Length()), float64(b.Length())
	if la < 1e-12 || lb < 1e-12 {
		return 0
	}
	cos := float64(a.Dot(b)) / (la * lb)
	if cos < -1 {
		cos = -1
	} else if cos > 1 {
		cos = 1
	}
	return stdmath.Acos(cos)
}

// turnAngle is the angle (radians) between the chords a→b and b→c — how much a point-track bends
// at b. Zero on a straight run, so a ruled (straight) loft adds no extra sub-sections.
func turnAngle(a, b, c math.Point3) float64 {
	d1, d2 := a.VectorTo(b), b.VectorTo(c)
	l1, l2 := float64(d1.Length()), float64(d2.Length())
	if l1 < 1e-12 || l2 < 1e-12 {
		return 0
	}
	cos := float64(d1.Dot(d2)) / (l1 * l2)
	if cos < -1 {
		cos = -1
	} else if cos > 1 {
		cos = 1
	}
	return stdmath.Acos(cos)
}

// hermiteSection blends one sub-section: each point is the Hermite interpolant of the two
// sections' corresponding points and their tangents at parameter t (t∈(0,1] along p0→p1).
func hermiteSection(p0, p1 []math.Point3, m0, m1 []math.Vector3, t float64) []math.Point3 {
	out := make([]math.Point3, len(p0))
	for j := range p0 {
		out[j] = hermite3(p0[j], p1[j], m0[j], m1[j], t)
	}
	return out
}

// hermite3 is the cubic Hermite interpolant of endpoints p0,p1 with tangents m0,m1 at t. With
// m0==m1 parallel to p1−p0 (the Free/Catmull-Rom two-section case) the result stays on the
// straight p0→p1 chord (ruled), so a Free loft is unchanged.
func hermite3(p0, p1 math.Point3, m0, m1 math.Vector3, t float64) math.Point3 {
	t2, t3 := t*t, t*t*t
	h00 := 2*t3 - 3*t2 + 1
	h10 := t3 - 2*t2 + t
	h01 := -2*t3 + 3*t2
	h11 := t3 - t2
	axis := func(a0, d0, a1, d1 float64) float64 {
		return h00*a0 + h10*d0 + h01*a1 + h11*d1
	}
	return math.P3(
		axis(p0.X, m0.X, p1.X, m1.X),
		axis(p0.Y, m0.Y, p1.Y, m1.Y),
		axis(p0.Z, m0.Z, p1.Z, m1.Z),
	)
}

// hermite5 is the quintic Hermite interpolant matching position, first AND second derivatives at
// both ends: p0,m0,a0 at t=0 and p1,m1,a1 at t=1. With a0=a1=0 it does NOT reduce to the cubic
// (the extra basis is nonzero), so it is only used where a curvature (G2) end condition supplies a
// real second derivative; the other end passes a natural a=0. Basis (the standard quintic Hermite):
//
//	H0=1−10t³+15t⁴−6t⁵  H1=t−6t³+8t⁴−3t⁵  H2=½t²−1.5t³+1.5t⁴−0.5t⁵
//	H3=10t³−15t⁴+6t⁵    H4=−4t³+7t⁴−3t⁵   H5=½t³−t⁴+½t⁵
func hermite5(p0, p1 math.Point3, m0, m1, a0, a1 math.Vector3, t float64) math.Point3 {
	t2 := t * t
	t3 := t2 * t
	t4 := t3 * t
	t5 := t4 * t
	h0 := 1 - 10*t3 + 15*t4 - 6*t5
	h1 := t - 6*t3 + 8*t4 - 3*t5
	h2 := 0.5*t2 - 1.5*t3 + 1.5*t4 - 0.5*t5
	h3 := 10*t3 - 15*t4 + 6*t5
	h4 := -4*t3 + 7*t4 - 3*t5
	h5 := 0.5*t3 - t4 + 0.5*t5
	axis := func(p0c, m0c, a0c, p1c, m1c, a1c float64) float64 {
		return h0*p0c + h1*m0c + h2*a0c + h3*p1c + h4*m1c + h5*a1c
	}
	return math.P3(
		axis(p0.X, m0.X, a0.X, p1.X, m1.X, a1.X),
		axis(p0.Y, m0.Y, a0.Y, p1.Y, m1.Y, a1.Y),
		axis(p0.Z, m0.Z, a0.Z, p1.Z, m1.Z, a1.Z),
	)
}

// hermite7 is the septic Hermite interpolant matching position, first, second AND third derivatives
// at both ends (p,m,a,j at t=0 and t=1) — the G3 end blend. It is built as a degree-7 Bézier whose
// end control points encode the endpoint derivatives (b1..b3 from p0,m0,a0,j0 and b4..b6 from
// p1,m1,a1,j1), then evaluated by de Casteljau. A natural (zero) higher derivative at the interior
// end leaves that side curvature-free.
func hermite7(p0, p1 math.Point3, m0, m1, a0, a1, j0, j1 math.Vector3, t float64) math.Point3 {
	axis := func(p0c, m0c, a0c, j0c, p1c, m1c, a1c, j1c float64) float64 {
		b0 := p0c
		b1 := p0c + m0c/7
		b2 := a0c/42 + 2*b1 - b0
		b3 := j0c/210 + 3*b2 - 3*b1 + b0
		b7 := p1c
		b6 := p1c - m1c/7
		b5 := a1c/42 + 2*b6 - b7
		b4 := b7 - 3*b6 + 3*b5 - j1c/210
		b := [8]float64{b0, b1, b2, b3, b4, b5, b6, b7}
		for n := 7; n > 0; n-- {
			for i := 0; i < n; i++ {
				b[i] += (b[i+1] - b[i]) * t
			}
		}
		return b[0]
	}
	return math.P3(
		math.Scalar(axis(float64(p0.X), float64(m0.X), float64(a0.X), float64(j0.X), float64(p1.X), float64(m1.X), float64(a1.X), float64(j1.X))),
		math.Scalar(axis(float64(p0.Y), float64(m0.Y), float64(a0.Y), float64(j0.Y), float64(p1.Y), float64(m1.Y), float64(a1.Y), float64(j1.Y))),
		math.Scalar(axis(float64(p0.Z), float64(m0.Z), float64(a0.Z), float64(j0.Z), float64(p1.Z), float64(m1.Z), float64(a1.Z), float64(j1.Z))),
	)
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
