// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Loft skinning — the SKIN SURFACE FIT (M48 #2236 split of loft_skin.go). Interpolates the skin between
// the aligned sections: the hermite blend (C0/C1/G2/G3), the adaptive segment/around sampling that keeps
// the chord and twist within tolerance, and the low-level hermite basis (hermite3/5/7). The section
// tangents it interpolates come from loft_skin_continuity.go.

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
	n := min(max(int(stdmath.Ceil(turn/(loftMaxStepDeg*stdmath.Pi/180))), loftSegmentSamples), loftMaxSegmentSamples)
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
	n := min(max(int(stdmath.Ceil(turn/(loftMaxStepDeg*stdmath.Pi/180))), loftSegmentSamples), loftMaxSegmentSamples)
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
	k := min(max(int(stdmath.Ceil(maxTwist/(loftAroundStepDeg*stdmath.Pi/180))), 1), loftMaxAroundSubdiv)
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
		for j := range m {
			a, b := sec[j], sec[(j+1)%m]
			for t := range k {
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
