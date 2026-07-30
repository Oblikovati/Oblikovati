// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The MID-SPAN OBSTACLE patch — the face bridging the fillet band across a boss that eats the host
// contact line — used to be a single Coons FillSurface through four right rails, and was the dominant
// per-face error in the whole obstacle class: +7.19 % (R9), +9.12 % (S3), +13.30 % (T6), +9.54 % (U3),
// +19.72 % (X3) against live DRAWEXE. Its straight wall rail also left the WALL face short by the entire
// tangency bulge (−0.21 % … −1.93 % of the wall face). It is now the exact SURF-RST rolling-ball canal: the
// ball tangent to the fillet wall, passing through the obstacle rim (kernel/ops/fillet_obstacle_canal.go).
//
// ★ THE ORACLE HERE IS A CLOSED FORM, not a recorded total. Each case's patch area and wall bulge are
// integrated analytically from the solid's own dimensions (the geometry is named per case below), and each
// closed form was checked against a LIVE DRAWEXE 8.0.0 run of the same case (`restore <rle> s ;
// tscale s 0 0 0 1 ; explode s E ; blend result s <r> s_<i> ; explode result F ; sprops result_i 1.e-6`).
// Every face in the table below re-reads IDENTICALLY at 1.e-9 and 1.e-12, so 1e-6 is sound for these ten;
// it is NOT sound in general — see perface_oracle_test.go's ★★ note on `sprops` quadrature tolerance
// before adding a row:
//
//	case  patch closed form   live DRAWEXE   wall bulge closed form   live DRAWEXE wall
//	R9      31.215583           31.2156        0.717167                340.717  (= 340  + 0.717)
//	S3     149.671704          149.672         4.624676                214.625  (= 210  + 4.625)
//	T6     156.364251          156.364         9.476511                569.476  (= 560  + 9.476)
//	U3      85.917424           42.9587 x 2    2.998261                302.998  (= 300  + 2.998)
//	X3    1471.701005         1471.7         147.353614               7647.35   (= 7500 + 147.35)
//
// Every one agrees to OCCT's own 6-figure printing precision, so these constants are an INDEPENDENT
// oracle, not a transcription of a golden master — which matters here, because two of these cases'
// whole-body totals do NOT reconcile with a live DRAWEXE run. T6's live `sprops result` reads 6845.4 at
// 1.e-6 against the corpus's recorded 6871.45, and the whole 26.05 gap sits on ONE face, its elliptical
// prism wall — but that face is exactly the tolerance-sensitive one: the same face reads 2355.61 / 2393.32
// / 2384.17 at 1e-6 / 1e-9 / 1e-12 against a closed form of 2381.677340, so the "gap" is DRAWEXE's own
// unconverged quadrature and NOT a defect of ours (ours is 2381.639771, −0.0016 %). The lesson stands
// either way: a whole-body pin would have been measuring that face instead of this patch.
//
// WHAT IT MEASURES. ops.MeshArea over ops.CalculateBodyFacets(body, ops.PropertyQuality()).FaceMeshes on
// the shipped body (ops-driven feature recompute via shippedCaseBody). The patch face is identified as the
// body's unique geom.BSplineSurface face; the wall face by its SUPPORTING PLANE (each case's closed-form
// wall plane) — never by DRAWEXE `bounding`, which returns a trimmed face's pole box (see
// perface_oracle_test.go).

// obstacleCanalCase is one case's closed-form expectations.
type obstacleCanalCase struct {
	name string
	// patchArea is the exact surf-rst canal area (see the table above).
	patchArea float64
	// wallPlainArea / wallBulge are the straight-seam wall area and the exact tangency-bulge gain, so the
	// gate asserts the bulge is PRESENT and right, not merely that the wall's total looks plausible.
	wallPlainArea, wallBulge float64
	// wallNormal/wallOrigin name the wall face's supporting plane (the identification key).
	wallNormal, wallOrigin math.Point3
}

// obstacleCanalPatchTol is the patch's relative budget. The measured residuals are +0.0029 % … +0.0283 %,
// all of one sign and all traceable to the RIM boundary being tiled by chords at the granularity the notch
// and the obstacle wall share (U3, the worst, has the coarsest dip: 13 samples across a 67 deg arc of a
// radius-12 rim). 0.1 % is ~3.5x the worst of those and ~200x inside the +19.7 % defect this replaces.
const obstacleCanalPatchTol = 0.001

// obstacleCanalBulgeTol is the wall bulge's relative budget. With the wall front tiled at full station
// density the shipped bulge reads only 0.0027 % … 0.0041 % short of its closed form, so 0.5 % separates
// that from any model regression by two orders — and losing the bulge entirely (the straight seam this
// replaces) is a 100 % miss.
const obstacleCanalBulgeTol = 0.005

// TestObstacleCanalMatchesItsClosedForm pins both faces the surf-rst model decides, on every corpus case
// that reaches a single-host obstacle rebuild.
func TestObstacleCanalMatchesItsClosedForm(t *testing.T) {
	for _, tc := range []obstacleCanalCase{
		// R9 — 20-cube, r=8 boss on z=10, fillet r=3 on y=−10 ∧ z=10. Wall = y=−10 (20x20 − 20x3 = 340).
		{"R9", 31.215583, 340, 0.717167, math.P3(0, 1, 0), math.P3(0, -10, 0)},
		// S3 — box z∈[−15,0], r=10 cone base on z=0, fillet r=8 on y=−15 ∧ z=0. Wall = y=−15 (30x15 − 30x8 = 210).
		{"S3", 149.671704, 210, 4.624676, math.P3(0, 1, 0), math.P3(0, -15, 0)},
		// T6 — box z∈[−20,0], 15x10 elliptic prism on z=0, fillet r=6 on y=−13 ∧ z=0. Wall = y=−13 (40x20 − 40x6 = 560).
		{"T6", 156.364251, 560, 9.476511, math.P3(0, 1, 0), math.P3(0, -13, 0)},
		// U3 — box x∈[−15,0], r=12 pipe boss out of x=0, fillet r=5 on x=0 ∧ z=15. Wall = z=15 (15x30 − 30x5 = 300).
		{"U3", 85.917424, 300, 2.998261, math.P3(0, 0, 1), math.P3(0, 0, 15)},
		// X3 — 100-cube z∈[0,100], r=30 boss on z=100 about (10,0), fillet r=25 on x=50 ∧ z=100.
		// Wall = x=50 (100x100 − 100x25 = 7500).
		{"X3", 1471.701005, 7500, 147.353614, math.P3(1, 0, 0), math.P3(50, 0, 0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, ok := shippedCaseBody(caseRecord(t, "simple", tc.name), CorpusFixtureDir())
			if !ok {
				t.Fatalf("simple/%s: no shipped body", tc.name)
			}
			facets := ops.CalculateBodyFacets(body, ops.PropertyQuality())
			assertObstaclePatchArea(t, tc, body, facets)
			assertObstacleWallBulge(t, tc, body, facets)
		})
	}
}

// assertObstaclePatchArea measures the unique free-form face — the obstacle patch — against its closed form.
func assertObstaclePatchArea(t *testing.T, tc obstacleCanalCase, body *topo.Body, facets *ops.BodyFacets) {
	t.Helper()
	area, found := 0.0, 0
	for i, f := range body.Faces() {
		if _, isBS := f.Geometry().(geom.BSplineSurface); isBS {
			area, found = ops.MeshArea(facets.FaceMeshes[i]), found+1
		}
	}
	if found != 1 {
		t.Fatalf("simple/%s: %d free-form faces, want exactly 1 (the obstacle patch)", tc.name, found)
	}
	rel := (area - tc.patchArea) / tc.patchArea
	if stdmath.Abs(rel) > obstacleCanalPatchTol {
		t.Errorf("simple/%s: obstacle patch area %.6f, closed form %.6f (%+.5f %%, budget %.3f %%) — the patch is not the surf-rst canal",
			tc.name, area, tc.patchArea, 100*rel, 100*obstacleCanalPatchTol)
	}
	t.Logf("simple/%s patch %.6f vs closed form %.6f (%+.5f %%)", tc.name, area, tc.patchArea, 100*rel)
}

// assertObstacleWallBulge measures the fillet WALL face and asserts the tangency bulge is present at its
// closed-form size. Losing the bulge is exactly what a straight wall seam does, so this is the gate that
// keeps the patch's shared front and the wall's own front describing one curve.
func assertObstacleWallBulge(t *testing.T, tc obstacleCanalCase, body *topo.Body, facets *ops.BodyFacets) {
	t.Helper()
	area, found := 0.0, 0
	for i, f := range body.Faces() {
		if isNamedPlane(f, tc.wallNormal, tc.wallOrigin) {
			area, found = ops.MeshArea(facets.FaceMeshes[i]), found+1
		}
	}
	if found != 1 {
		t.Fatalf("simple/%s: %d faces on the wall plane, want exactly 1", tc.name, found)
	}
	bulge := area - tc.wallPlainArea
	rel := (bulge - tc.wallBulge) / tc.wallBulge
	if stdmath.Abs(rel) > obstacleCanalBulgeTol {
		t.Errorf("simple/%s: wall face %.6f = %.0f + %.6f of bulge, closed form %.6f (%+.4f %%, budget %.2f %%) — the wall front is back on the straight seam",
			tc.name, area, tc.wallPlainArea, bulge, tc.wallBulge, 100*rel, 100*obstacleCanalBulgeTol)
	}
	t.Logf("simple/%s wall %.6f, bulge %.6f vs closed form %.6f (%+.4f %%)", tc.name, area, bulge, tc.wallBulge, 100*rel)
}

// isNamedPlane reports whether f is a plane with the given unit normal (either sense) through origin —
// identification by SUPPORTING GEOMETRY, the only key that survives a face-count or area change.
func isNamedPlane(f *topo.Face, normal, origin math.Point3) bool {
	p, isPlane := f.Geometry().(geom.Plane)
	if !isPlane {
		return false
	}
	n := math.V3(float64(normal.X), float64(normal.Y), float64(normal.Z))
	if stdmath.Abs(stdmath.Abs(float64(p.Normal().Dot(n)))-1) > 1e-9 {
		return false
	}
	return stdmath.Abs(float64(p.Normal().Dot(p.Origin.VectorTo(origin)))) < 1e-9
}
