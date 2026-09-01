// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Regression for Oblikovati/Oblikovati#1323 L3: a non-rectangular B-spline trim was meshed boundary-
// only (trimmedPatchMesh), chording flat across the surface's interior curvature. B-spline trims now
// go through metricPatchMesh, which adds curvature-adaptive interior Steiner points, so the meshed
// area converges to the true surface area instead of under-filling.

// curvedBumpPatch is a biquadratic B-spline patch over the unit (u,v) domain bulging in +Z — curved
// enough that a boundary-only triangulation chords visibly across the interior.
func curvedBumpPatch(t *testing.T) geom.BSplineSurface {
	t.Helper()
	ctrl := [][]math.Point3{
		{math.P3(0, 0, 0), math.P3(0, 0.5, 0.3), math.P3(0, 1, 0)},
		{math.P3(0.5, 0, 0.3), math.P3(0.5, 0.5, 1.0), math.P3(0.5, 1, 0.3)},
		{math.P3(1, 0, 0), math.P3(1, 0.5, 0.3), math.P3(1, 1, 0)},
	}
	w := [][]float64{{1, 1, 1}, {1, 1, 1}, {1, 1, 1}}
	s, err := geom.NewBSplineSurface(2, 2, ctrl, w, []float64{0, 0, 0, 1, 1, 1}, []float64{0, 0, 0, 1, 1, 1})
	if err != nil {
		t.Fatalf("NewBSplineSurface: %v", err)
	}
	return s
}

// referencePatchArea integrates |∂P/∂u × ∂P/∂v| over [0,1]² on a fine grid — the true surface area.
func referencePatchArea(s geom.Surface) float64 {
	const n = 200
	h := 1.0 / n
	var area float64
	for i := range n {
		for j := range n {
			u, v := (float64(i)+0.5)*h, (float64(j)+0.5)*h
			du, dv := s.DerivativesAt(u, v)
			area += float64(du.Cross(dv).Length()) * h * h
		}
	}
	return area
}

// denseBoundary samples the patch's four edges at m points each → the trim loop (3D + uv), so the
// boundary itself is accurate and the only meshing difference is interior refinement.
func denseBoundary(s geom.Surface, m int) (p3 []math.Point3, uv []math.Point2) {
	add := func(u, v float64) {
		p3 = append(p3, s.PointAt(u, v))
		uv = append(uv, math.P2(math.Scalar(u), math.Scalar(v)))
	}
	for i := range m {
		add(float64(i)/float64(m), 0)
	}
	for i := range m {
		add(1, float64(i)/float64(m))
	}
	for i := range m {
		add(1-float64(i)/float64(m), 1)
	}
	for i := range m {
		add(0, 1-float64(i)/float64(m))
	}
	return p3, uv
}

// TestBSplineTrimInteriorRefinementArea is the core L3 assertion: on a curved B-spline trim, the
// interior-refined mesh (metricPatchMesh, the new B-spline path) is closer to the true area than the
// boundary-only mesh (trimmedPatchMesh, the old path), and within the chord tolerance.
func TestBSplineTrimInteriorRefinementArea(t *testing.T) {
	t.Parallel()
	s := curvedBumpPatch(t)
	ref := referencePatchArea(s)
	p3, uv := denseBoundary(s, 24)

	boundaryOnly := trimmedPatchMesh(s, p3, nil)
	q := Quality{ChordTolerance: 0.01, AngleTolerance: 5 * stdmath.Pi / 180}
	refined := MetricPatchMesh(s, q, p3, nil, uv, nil)

	if refined.TriangleCount() <= boundaryOnly.TriangleCount() {
		t.Errorf("refined mesh has %d triangles, boundary-only %d — no interior Steiner points added",
			refined.TriangleCount(), boundaryOnly.TriangleCount())
	}
	errBoundary := stdmath.Abs(boundaryOnly.Area() - ref)
	errRefined := stdmath.Abs(refined.Area() - ref)
	if errRefined >= errBoundary {
		t.Errorf("interior refinement did not reduce area error: refined %g vs boundary-only %g (ref %g)",
			errRefined, errBoundary, ref)
	}
	// At chord 0.01 the interior-refined area is within ~1% of truth (it tightens further with the
	// chord — see TestBSplineTrimAreaConvergesWithQuality); the boundary-only mesh is far worse.
	if rel := errRefined / ref; rel > 1.5e-2 {
		t.Errorf("refined area rel error %g exceeds tolerance (area %g, ref %g)", rel, refined.Area(), ref)
	}
}

// TestBSplineTrimAreaConvergesWithQuality asserts the refined B-spline trim area converges toward the
// true area as the chord tolerance tightens (monotone, no spike).
func TestBSplineTrimAreaConvergesWithQuality(t *testing.T) {
	t.Parallel()
	s := curvedBumpPatch(t)
	ref := referencePatchArea(s)
	p3, uv := denseBoundary(s, 24)
	prev := stdmath.Inf(1)
	for _, chord := range []float64{0.05, 0.01, 0.002} {
		q := Quality{ChordTolerance: chord, AngleTolerance: 5 * stdmath.Pi / 180}
		m := MetricPatchMesh(s, q, p3, nil, uv, nil)
		relErr := stdmath.Abs(m.Area()-ref) / ref
		if relErr > prev+1e-9 {
			t.Errorf("chord %g: rel area error %g grew vs previous %g (non-monotone)", chord, relErr, prev)
		}
		prev = relErr
	}
	if prev > 2e-3 {
		t.Errorf("finest B-spline trim area rel error %g too large", prev)
	}
}
