// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Span refinement for warped swept sides (Oblikovati#2078). A side quad between two consecutive
// sections is a RULED surface; sideQuad approximates it with two triangles when it is not planar.
// Over a strongly twisted span those two triangles are nowhere near the surface they stand for:
// for the twisted blade of #2078 they deviate 0.13 from it, on a blade only 0.16 thick, so opposite
// sides of the blade cut straight through each other and the body's volume comes out 55% low.
//
// Choosing the other diagonal does NOT help — the two choices bracket the true volume almost
// symmetrically (0.173 and 0.599 against a true 0.386). The approximation is simply too coarse, so
// the fix is to subdivide the span until each facet is close to the surface.

// maxFacetWarpRatio is how far a triangulated side facet may sit from the ruled surface it stands
// for, as a fraction of that facet's own size. Being a ratio of two lengths it carries no model
// scale (ADR-0042).
//
// A quad split into two triangles meets the ruled surface at its four corners and along the split
// diagonal; the gap is widest at the centre, where it is |a−b+c−d|/4. Halving a span halves that
// gap exactly (the warp vector is linear in the span fraction — see spanWarpRatio), so the span
// count needed is just the ratio of the measured warp to this bound.
// The value sits in a measured plateau with a real bound on each side:
//   - BELOW 0.0245 it starts subdividing spans the feature generators already sample adequately —
//     a helical coil emits exactly 0.0245 — which doubles their mesh for nothing and reinstates the
//     #879 fine-pitch weld failure (dense coincident vertices straddling weld-grid cells).
//   - ABOVE 0.0747 the #2078 twisted blade is left at one span and cuts through itself.
//
// So this is not free to move: it is bounded below by cost and above by correctness.
const maxFacetWarpRatio = 0.03 // tol:numeric — a ratio of two lengths, so it carries no model scale

// refineWarpedSpans returns the sections with linearly interpolated sections inserted wherever a
// span's side facets would stray too far from the ruled surface. Interpolating linearly is exactly
// the surface sectionMesh already implies between two sections, so this refines the approximation
// without changing the shape it approximates.
//
// A CLOSED loop is returned untouched: its wrap span carries the correspondence offset that
// closureShift resolved (a twisted closed loft joins across the seam by that shift), and there is
// no meaningful way to interpolate a section part-way through that offset.
//
// Example: refineWarpedSpans([][]math.Point3{root, tip}, false) // 2 sections in, 9 out at 0.3 rad
func refineWarpedSpans(sections [][]math.Point3, closedLoop bool) [][]math.Point3 {
	if closedLoop {
		return sections
	}
	out := [][]math.Point3{sections[0]}
	for s := 0; s+1 < len(sections); s++ {
		a, b := sections[s], sections[s+1]
		m := spanSubdivision(a, b)
		for step := 1; step < m; step++ {
			out = append(out, lerpSection(a, b, float64(step)/float64(m)))
		}
		out = append(out, b)
	}
	return out
}

// spanSubdivision is how many sub-spans one section-to-section span needs to bring every facet
// within maxFacetWarpRatio of the ruled surface.
//
// It needs no explicit cap: spanWarpRatio cannot exceed 1/2 (see its proof), so at the current
// budget the count is at most ceil(0.5/0.03) = 17 however degenerate the input.
func spanSubdivision(a, b []math.Point3) int {
	m := int(stdmath.Ceil(spanWarpRatio(a, b) / maxFacetWarpRatio))
	if m < 1 {
		return 1
	}
	return m
}

// spanWarpRatio is the worst facet deviation across one span, relative to the facet's own size.
//
// For quad (a_i, a_j, b_j, b_i) the two-triangle split misses the ruled surface by |W|/4 at the
// centre, where W = (a_i − a_j) − (b_i − b_j). Subdividing the span into m sub-spans scales W by
// 1/m: the corner difference is linear in the span parameter, so its change across a sub-span of
// length 1/m is exactly 1/m of its change across the whole span. Hence the linear span count.
//
// The ratio is bounded by 1/2, which is why refinement needs no separate cap: a_i−a_j and b_i−b_j
// are both SIDES of the quad, so each is at most quadScale, hence |W| <= 2*quadScale and
// |W|/4/quadScale <= 1/2. See TestSpanWarpRatioCannotExceedAHalf.
func spanWarpRatio(a, b []math.Point3) float64 {
	worst := 0.0
	for i := range a {
		j := (i + 1) % len(a)
		w := a[i].VectorTo(a[j]).Sub(b[i].VectorTo(b[j]))
		scale := quadScale(a[i], a[j], b[j], b[i])
		if scale == 0 {
			continue // a collapsed quad has no size to be warped relative to
		}
		worst = stdmath.Max(worst, float64(w.Length())/4/scale)
	}
	return worst
}

// quadScale is the quad's longest side — the length the facet deviation is judged against.
func quadScale(a, b, c, d math.Point3) float64 {
	corners := [4]math.Point3{a, b, c, d}
	longest := 0.0
	for i := range corners {
		longest = stdmath.Max(longest, float64(corners[i].DistanceTo(corners[(i+1)%4])))
	}
	return longest
}

// lerpSection is the section part-way along a span — the same ruled surface sectionMesh spans.
func lerpSection(a, b []math.Point3, t float64) []math.Point3 {
	out := make([]math.Point3, len(a))
	for i := range a {
		out[i] = math.P3(
			float64(a[i].X)+(float64(b[i].X)-float64(a[i].X))*t,
			float64(a[i].Y)+(float64(b[i].Y)-float64(a[i].Y))*t,
			float64(a[i].Z)+(float64(b[i].Z)-float64(a[i].Z))*t,
		)
	}
	return out
}
