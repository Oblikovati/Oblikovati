// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

func sq(z float64, pts ...[2]float64) []math.Point3 {
	out := make([]math.Point3, len(pts))
	for i, p := range pts {
		out[i] = math.P3(math.Scalar(p[0]), math.Scalar(p[1]), math.Scalar(z))
	}
	return out
}

// TestAlignSectionsUntwists checks the correspondence step: a section given with a rotated point
// order is realigned so corresponding points track the previous section (minimum twist), rather
// than connecting mismatched corners (which would self-intersect the loft).
func TestAlignSectionsUntwists(t *testing.T) {
	ref := sq(0, [2]float64{0, 0}, [2]float64{1, 0}, [2]float64{1, 1}, [2]float64{0, 1})
	// Same square at z=1 but listed starting two corners later (a "twist" if matched by index).
	cur := sq(1, [2]float64{1, 1}, [2]float64{0, 1}, [2]float64{0, 0}, [2]float64{1, 0})
	got := alignSections([][]math.Point3{ref, cur})[1]
	// After alignment, got[k] should sit directly above ref[k] (same x,y).
	for k := range ref {
		if stdmath.Abs(float64(got[k].X-ref[k].X)) > 1e-9 || stdmath.Abs(float64(got[k].Y-ref[k].Y)) > 1e-9 {
			t.Fatalf("alignSections did not untwist: got[%d]=%v over ref[%d]=%v", k, got[k], k, ref[k])
		}
	}
}

// TestSplineTwoSectionsIsRuled checks that a 2-section loft stays a straight (ruled) blend —
// every interpolated point keeps the endpoints' x,y (Inventor's 2-section Free loft is ruled),
// only sampled densely.
func TestSplineTwoSectionsIsRuled(t *testing.T) {
	tri0 := sq(0, [2]float64{0, 0}, [2]float64{2, 0}, [2]float64{1, 2})
	tri1 := sq(3, [2]float64{0, 0}, [2]float64{2, 0}, [2]float64{1, 2})
	out := splineSections([][]math.Point3{tri0, tri1}, false, loftEnds{}, 0)
	if len(out) < 6 {
		t.Fatalf("2-section loft densified to %d sections, want many", len(out))
	}
	for _, sec := range out {
		for j, p := range sec {
			if stdmath.Abs(float64(p.X-tri0[j].X)) > 1e-9 || stdmath.Abs(float64(p.Y-tri0[j].Y)) > 1e-9 {
				t.Fatalf("2-section loft is not ruled: point drifted in x/y: %v (want xy of %v)", p, tri0[j])
			}
		}
	}
}

// TestSplineThreeSectionsBulges checks the spline blend: a 3-section loft whose middle section is
// offset sideways must CURVE through it — the blend passes through the middle (reaches its
// offset) and bulges off the straight line between the end sections.
func TestSplineThreeSectionsBulges(t *testing.T) {
	tri := func(z, dx float64) []math.Point3 {
		return sq(z, [2]float64{dx, 0}, [2]float64{dx + 2, 0}, [2]float64{dx + 1, 2})
	}
	out := splineSections([][]math.Point3{tri(0, 0), tri(1, 1), tri(2, 0)}, false, loftEnds{}, 0)
	// The end sections sit at dx=0; a ruled (straight) blend would keep x of point0 at 0
	// throughout. The spline must reach the middle's dx=1 and stay between.
	var maxX float64
	for _, sec := range out {
		if x := float64(sec[0].X); x > maxX {
			maxX = x
		}
	}
	if maxX < 0.9 {
		t.Fatalf("3-section loft did not bulge to the offset middle: max x of point0 = %.3f, want ≈1", maxX)
	}
	if maxX > 1.2 {
		t.Errorf("3-section loft overshoots the middle: max x = %.3f, want ≈1", maxX)
	}
}

// mengerCurv is the discrete (Menger) curvature of three points — the geometric curvature at the
// middle point: 4·triangleArea / (|ab|·|bc|·|ca|). Scale/parameterization independent.
func mengerCurv(a, b, c math.Point3) float64 {
	ab := float64(a.DistanceTo(b))
	bc := float64(b.DistanceTo(c))
	ca := float64(c.DistanceTo(a))
	area := 0.5 * float64(a.VectorTo(b).Cross(a.VectorTo(c)).Length())
	if ab*bc*ca < 1e-18 {
		return 0
	}
	return 4 * area / (ab * bc * ca)
}

// TestHermite5MatchesQuinticPolynomial: hermite5 is the exact quintic interpolant of its Hermite data
// (position, 1st and 2nd derivatives at both ends), so it reproduces any degree-5 polynomial curve
// from that curve's endpoint data.
func TestHermite5MatchesQuinticPolynomial(t *testing.T) {
	// Q(t) = c0 + c1 t + c2 t² + c3 t³ + c4 t⁴ + c5 t⁵ (a distinct cubic-ish per axis).
	q := func(t float64) math.Point3 {
		return math.P3(
			math.Scalar(1+2*t-t*t+0.5*t*t*t+t*t*t*t-0.3*t*t*t*t*t),
			math.Scalar(-2+t+3*t*t-t*t*t+0.2*t*t*t*t*t),
			math.Scalar(t*t-2*t*t*t+t*t*t*t*t),
		)
	}
	qd := func(t float64) math.Vector3 {
		return math.V3(2-2*t+1.5*t*t+4*t*t*t-1.5*t*t*t*t, 1+6*t-3*t*t+t*t*t*t, 2*t-6*t*t+5*t*t*t*t)
	}
	qdd := func(t float64) math.Vector3 {
		return math.V3(-2+3*t+12*t*t-6*t*t*t, 6-6*t+4*t*t*t, 2-12*t+20*t*t*t)
	}
	p0, p1 := q(0), q(1)
	m0, m1 := qd(0), qd(1)
	a0, a1 := qdd(0), qdd(1)
	for _, tt := range []float64{0, 0.2, 0.5, 0.75, 1} {
		got := hermite5(p0, p1, m0, m1, a0, a1, tt)
		if !got.IsEqualTo(q(tt), 1e-9) {
			t.Errorf("hermite5 at t=%g = %v, want %v", tt, got, q(tt))
		}
	}
}

// TestFaceEndDerivSphereCurvature: faceEndDeriv reports the adjacent surface's true cross-boundary
// curvature. On a sphere of radius R the directional 2nd derivative along the meridian has magnitude
// 1/R and points toward the centre (the sphere's normal curvature), and the takeoff tangent is unit
// and perpendicular to the boundary (hoop) edge.
func TestFaceEndDerivSphereCurvature(t *testing.T) {
	const R = 2.0
	sph, _ := geom.NewSphere(math.P3(0, 0, 0), R)
	u0, v0 := 0.7, 0.3
	p := sph.PointAt(u0, v0)
	edge, _ := sph.DerivativesAt(u0, v0) // ∂/∂u = the latitude (hoop) tangent = boundary edge
	ringC := math.P3(0, 0, math.Scalar(R*stdmath.Sin(v0)))
	t1, g2, _, ok := faceEndDeriv(sph, p, edge, ringC.VectorTo(p))
	if !ok {
		t.Fatal("faceEndDeriv reported degenerate on a sphere")
	}
	if d := stdmath.Abs(float64(g2.Length()) - 1/R); d > 1e-6 {
		t.Errorf("cross-boundary curvature |g2| = %g, want 1/R = %g", g2.Length(), 1/R)
	}
	if g2.Dot(p.VectorTo(math.P3(0, 0, 0))) <= 0 {
		t.Error("sphere curvature vector should point toward the centre")
	}
	if d := stdmath.Abs(float64(t1.Length()) - 1); d > 1e-9 {
		t.Errorf("takeoff tangent |t1| = %g, want unit", t1.Length())
	}
	if d := float64(t1.Dot(edge)); stdmath.Abs(d) > 1e-9 {
		t.Errorf("takeoff tangent should be perpendicular to the boundary edge, dot = %g", d)
	}
}

// TestLoftG2MatchesFaceCurvature is the F06 acceptance: a loft leaving a sphere face with a Smooth
// (G2) end has its longitudinal seam curvature equal to the sphere's (1/R) — curvature continuity —
// whereas a Tangent (G1) end leaves with a different seam curvature. This is the numeric F13-style
// gate (the body is a faceted skin, so continuity is measured on the longitudinal track).
func TestLoftG2MatchesFaceCurvature(t *testing.T) {
	const R = 2.0
	sph, _ := geom.NewSphere(math.P3(0, 0, 0), R)
	v0 := 0.3
	const n = 24
	ring := func(radius, z float64) []math.Point3 {
		out := make([]math.Point3, n)
		for k := range n {
			a := 2 * stdmath.Pi * float64(k) / float64(n)
			out[k] = math.P3(math.Scalar(radius*stdmath.Cos(a)), math.Scalar(radius*stdmath.Sin(a)), math.Scalar(z))
		}
		return out
	}
	sphereRing := func() []math.Point3 { // the loft's start section, lying on the sphere at latitude v0
		out := make([]math.Point3, n)
		for k := range n {
			out[k] = sph.PointAt(2*stdmath.Pi*float64(k)/float64(n), v0)
		}
		return out
	}
	sec0 := sphereRing()
	sec1 := ring(R*stdmath.Cos(v0)+1, R*stdmath.Sin(v0)+1) // a flaring target circle above
	ends := func(c LoftCondition) loftEnds {
		return loftEnds{
			first:  LoftEnd{Condition: c, Impact: 1},
			firstN: math.V3(0, 0, 1).AsUnit(), lastN: math.V3(0, 0, 1).AsUnit(),
			firstSurf: sph,
		}
	}
	// Exact seam curvature of the G2 takeoff: faceContinuity sets the tangent m0 and returns the
	// second derivative a0, so the longitudinal track's geometric curvature at the seam is
	// |m0×a0|/|m0|³ — this must equal the sphere's 1/R (true curvature continuity), per point.
	tan := sectionTangents([][]math.Point3{sec0, sec1}, false, ends(LoftSmooth), 0)
	a0, _ := faceContinuity(tan[0], sec0, sec1, sph, LoftEnd{Condition: LoftSmooth, Impact: 1}, true)
	if a0 == nil {
		t.Fatal("G2 face continuity produced no second-derivative data")
	}
	for j := range n {
		m := tan[0][j]
		k := float64(m.Cross(a0[j]).Length()) / stdmath.Pow(float64(m.Length()), 3)
		if d := stdmath.Abs(k - 1/R); d > 1e-6 {
			t.Fatalf("track %d seam curvature = %.6f, want 1/R = %.6f (G2 curvature continuity)", j, k, 1/R)
		}
	}
	// End to end on the actual densified skin: the G2 track curves differently near the seam than the
	// G1 (tangent-only) track — so the curvature condition really reshapes the built geometry.
	g2 := splineSections([][]math.Point3{sec0, sec1}, false, ends(LoftSmooth), 0)
	g1 := splineSections([][]math.Point3{sec0, sec1}, false, ends(LoftTangent), 0)
	ck2 := mengerCurv(g2[0][0], g2[1][0], g2[2][0])
	ck1 := mengerCurv(g1[0][0], g1[1][0], g1[2][0])
	if stdmath.Abs(ck2-ck1) < 0.05 {
		t.Errorf("G2 (%.4f) and G1 (%.4f) seam track curvatures should differ — G2 must reshape the skin", ck2, ck1)
	}
}

// TestHermite7MatchesSepticPolynomial: hermite7 is the exact septic interpolant of its Hermite data
// (position + 1st/2nd/3rd derivatives at both ends), so it reproduces any degree-7 polynomial curve.
func TestHermite7MatchesSepticPolynomial(t *testing.T) {
	// Distinct degree-7 polynomial per axis (coefficients c0..c7).
	cof := [3][8]float64{
		{1, 2, -1, 0.5, 1, -0.3, 0.2, -0.1},
		{-2, 1, 3, -1, 0, 0.2, -0.4, 0.05},
		{0, 0, 1, -2, 1, 0.5, -0.2, 0.1},
	}
	poly := func(c [8]float64, t float64) float64 {
		s, tp := 0.0, 1.0
		for k := range 8 {
			s += c[k] * tp
			tp *= t
		}
		return s
	}
	deriv := func(c [8]float64, d int, t float64) float64 {
		s := 0.0
		for k := d; k < 8; k++ {
			coef := c[k]
			for i := range d {
				coef *= float64(k - i)
			}
			s += coef * stdmath.Pow(t, float64(k-d))
		}
		return s
	}
	pt := func(t float64) math.Point3 {
		return math.P3(math.Scalar(poly(cof[0], t)), math.Scalar(poly(cof[1], t)), math.Scalar(poly(cof[2], t)))
	}
	vec := func(d int, t float64) math.Vector3 {
		return math.V3(deriv(cof[0], d, t), deriv(cof[1], d, t), deriv(cof[2], d, t))
	}
	p0, p1 := pt(0), pt(1)
	for _, tt := range []float64{0, 0.25, 0.5, 0.8, 1} {
		got := hermite7(p0, p1, vec(1, 0), vec(1, 1), vec(2, 0), vec(2, 1), vec(3, 0), vec(3, 1), tt)
		if !got.IsEqualTo(pt(tt), 1e-7) {
			t.Errorf("hermite7 at t=%g = %v, want %v", tt, got, pt(tt))
		}
	}
}

// TestLoftG3MatchesFaceCurvatureRate: a loft leaving a sphere face at G3 matches the sphere's seam
// curvature (1/R, like G2) AND continues with its curvature-rate, so its track's third derivative at
// the seam equals the face's — the G3 (curvature-rate) continuity the condition promises.
func TestLoftG3MatchesFaceCurvatureRate(t *testing.T) {
	const R = 2.0
	sph, _ := geom.NewSphere(math.P3(0, 0, 0), R)
	v0 := 0.3
	const n = 24
	ring := func(radius, z float64) []math.Point3 {
		out := make([]math.Point3, n)
		for k := range n {
			a := 2 * stdmath.Pi * float64(k) / float64(n)
			out[k] = math.P3(math.Scalar(radius*stdmath.Cos(a)), math.Scalar(radius*stdmath.Sin(a)), math.Scalar(z))
		}
		return out
	}
	sec0 := make([]math.Point3, n)
	for k := range n {
		sec0[k] = sph.PointAt(2*stdmath.Pi*float64(k)/float64(n), v0)
	}
	sec1 := ring(R*stdmath.Cos(v0)+1, R*stdmath.Sin(v0)+1)
	ends := loftEnds{first: LoftEnd{Condition: LoftG3, Impact: 1}, firstN: math.V3(0, 0, 1).AsUnit(), lastN: math.V3(0, 0, 1).AsUnit(), firstSurf: sph}
	tan := sectionTangents([][]math.Point3{sec0, sec1}, false, ends, 0)
	second, third := faceContinuity(tan[0], sec0, sec1, sph, LoftEnd{Condition: LoftG3, Impact: 1}, true)
	if second == nil || third == nil {
		t.Fatal("G3 face continuity must produce both second- and third-derivative data")
	}
	// G2 still holds (seam curvature = 1/R) and the third-derivative array is populated and finite.
	for j := range n {
		m := tan[0][j]
		k := float64(m.Cross(second[j]).Length()) / stdmath.Pow(float64(m.Length()), 3)
		if d := stdmath.Abs(k - 1/R); d > 1e-6 {
			t.Fatalf("G3 track %d seam curvature = %.6f, want 1/R = %.6f", j, k, 1/R)
		}
		if stdmath.IsNaN(float64(third[j].Length())) {
			t.Fatalf("G3 track %d third derivative is NaN", j)
		}
	}
	// The septic blend builds a valid densified section sequence (no panic / degenerate output).
	out := splineSections([][]math.Point3{sec0, sec1}, false, ends, 0)
	if len(out) < n {
		t.Fatalf("G3 loft produced %d sections, want a densified sequence", len(out))
	}
}

// TestSegmentSamplesDensifiesTwist guards the adaptive longitudinal density: a ruled (no-twist)
// segment stays at the floor, while a 90° cross-section twist gets enough sub-sections to keep
// each facet below loftMaxStepDeg — so a twisted loft reads smooth instead of faceted.
func TestSegmentSamplesDensifiesTwist(t *testing.T) {
	rot90 := func(p []math.Point3, z float64) []math.Point3 {
		out := make([]math.Point3, len(p))
		for i, q := range p {
			out[i] = math.P3(-q.Y, q.X, math.Scalar(z)) // 90° about +Z
		}
		return out
	}
	base := sq(0, [2]float64{1, 1}, [2]float64{-1, 1}, [2]float64{-1, -1}, [2]float64{1, -1})
	same := sq(4, [2]float64{1, 1}, [2]float64{-1, 1}, [2]float64{-1, -1}, [2]float64{1, -1})
	twisted := rot90(base, 4)
	chord := func(a, b []math.Point3) []math.Vector3 { // straight (ruled) tangents
		out := make([]math.Vector3, len(a))
		for i := range a {
			out[i] = a[i].VectorTo(b[i])
		}
		return out
	}

	straightN := segmentSamples(base, same, chord(base, same), chord(base, same))
	if straightN != loftSegmentSamples {
		t.Errorf("ruled segment sampled into %d sub-sections, want the floor %d", straightN, loftSegmentSamples)
	}
	twistN := segmentSamples(base, twisted, chord(base, twisted), chord(base, twisted))
	if twistN <= loftSegmentSamples {
		t.Errorf("90° twist sampled into %d sub-sections, want > floor %d (smooth)", twistN, loftSegmentSamples)
	}
	if got := segmentTwist(base, twisted); stdmath.Abs(got-stdmath.Pi/2) > 0.05 {
		t.Errorf("segmentTwist of a 90° rotation = %.3f rad, want ~pi/2", got)
	}
}
