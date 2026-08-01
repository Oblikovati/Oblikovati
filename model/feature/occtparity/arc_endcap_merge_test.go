// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// The ARC-fillet setback end-cap merge (kernel/ops/fillet_arc_endcap.go), gated on the CLOSED FORMS it
// buys rather than on "the retrace went away".
//
// WHAT WAS WRONG. rebuildWithArcFillet closed each end of the torus band with a flat setback triangle in
// the RADIAL plane through that end. On a sector solid the cut wall IS that radial plane, so the triangle
// was emitted as a second, coincident face while the wall kept its own un-receded corner — and the cap
// face's loop was routed out to that corner and straight back, a zero-area RETRACE of exactly the fillet
// radius. simple/B2 shipped EIGHT faces where DRAWEXE ships six, its two walls at 5000 against 4978.54,
// and 2·(21.46 + 21.46) = 85.84 of surplus, +0.401 % — comfortably inside the 1 % gate that called it PASS.
//
// WHY EVERY NUMBER BELOW IS A CLOSED FORM AND NOT A CAPTURE. Each is derived from the solid the fillet
// SHOULD produce and independently confirmed face-for-face against DRAWEXE 8.0.0 (`blend result s 10 s_1`
// on B2's own pcylinder script, `sprops` per exploded face): 7068.58 / 4978.54 / 4978.54 / 1963.50 /
// 1256.64 / 1144.04, summing to 21389.8 — OCCT's own expectedArea for the case. The MEASURING FUNCTION is
// ops.MeshArea over shippedFaceMesh at ops.PropertyQuality(), which is a POLYGONAL under-estimate of a
// curved face, hence the per-face budget below rather than exactness.
const arcEndCapPerFaceTol = 1e-4

// b2 closed forms. A 90° sector of a cylinder R=50, height 100, with r=10 on its TOP rim arc.
const (
	b2R    = 50.0
	b2H    = 100.0
	b2Fill = 10.0
)

// b2WallArea is one radial cut wall: the R × H rectangle LESS the corner the blend rounds off — the
// r × r square minus the quarter disc the terminal cross-section arc sweeps out of it. 4978.5398…
func b2WallArea() float64 {
	return b2R*b2H - (b2Fill*b2Fill - stdmath.Pi*b2Fill*b2Fill/4)
}

// b2BandArea is the quarter torus the blend sweeps: tube radius r about a centre circle of radius R−r,
// a quarter turn wide and a quarter tube tall. ∫∫ (R−r + r cos θ) r dθ dφ over [0,π/2]². 1144.0362…
func b2BandArea() float64 {
	return (stdmath.Pi / 2) * b2Fill * ((b2R-b2Fill)*(stdmath.Pi/2) + b2Fill)
}

// TestB2ArcFilletMergesItsSetbackCapsIntoTheSectorWalls pins the whole of simple/B2 on the six faces
// DRAWEXE ships. It is the regression guard for the merge: reinstate the separate end-cap face and the
// two walls go back to 5000, two surplus faces reappear, and the face count assertion fails first.
func TestB2ArcFilletMergesItsSetbackCapsIntoTheSectorWalls(t *testing.T) {
	t.Parallel()
	body := gridCaseBody(t, corpusRecord(t, "simple", "B2"))
	if n := len(body.Faces()); n != 6 {
		t.Errorf("simple/B2 ships %d faces, want the 6 DRAWEXE ships (a coplanar setback cap is part of "+
			"its sector wall, not a face of its own)", n)
	}
	if n := countLineagePrefix(body, "arcfillet:endcap"); n != 0 {
		t.Errorf("simple/B2 ships %d separate setback end-cap face(s), want 0 — both ends are radial cuts", n)
	}
	for _, tc := range []struct {
		lineage string
		want    float64
	}{
		{"import:step#16:face#0", stdmath.Pi * b2R / 2 * (b2H - b2Fill)},            // cylinder wall, receded to z=90
		{"import:step#16:face#1", stdmath.Pi * (b2R - b2Fill) * (b2R - b2Fill) / 4}, // top plane, receded to R−r
		{"import:step#16:face#2", stdmath.Pi * b2R * b2R / 4},                       // bottom plane, untouched
		{"import:step#16:face#3", b2WallArea()},
		{"import:step#16:face#4", b2WallArea()},
		{"arcfillet:torus#0", b2BandArea()},
	} {
		assertFaceMeshesToClosedForm(t, body, "simple/B2", tc.lineage, tc.want)
	}
	if bad := ops.RetracingFaceLoops(body, ops.PropertyQuality()); len(bad) != 0 {
		t.Errorf("simple/B2 still retraces on %d face loop(s): %s", len(bad), describeRetracing(bad))
	}
}

// TestN6ArcFilletTerminatesBothEndsOnTheirWalls is the DECLINE half of the coplanar merge — on the one
// corpus case that has one end of each kind — and, since the run-out landed, the guard that a declined
// merge is NOT the same thing as a shipped setback triangle.
//
// simple/N6's r=5 band starts on a radial cut wall (x=50 — the setback cap's own plane, so it merges) and
// stops on a wall the radial plane crosses at 0.6435 rad (x=80, which declines). This test used to assert
// that the declined end ships a setback cap of its own. It does not, and DRAWEXE says so: `blend result s
// 5 s_5` ships NINE faces, not ten, and the tenth we used to ship was drawn into the void — the setback
// cap's tip lands at (77,14,10), inside the box the cut REMOVED (x∈[50,80], y∈[0,30], z∈[10,80]). The
// band is instead terminated on that wall's own spiric section (kernel/ops/fillet_arc_runout.go), which
// reproduces OCCT face for face. Reinstate the setback triangle and the face count, the wall, the pocket
// floor and the band areas all move at once.
func TestN6ArcFilletTerminatesBothEndsOnTheirWalls(t *testing.T) {
	t.Parallel()
	body := gridCaseBody(t, corpusRecord(t, "simple", "N6"))
	if n := len(body.Faces()); n != 9 {
		t.Errorf("simple/N6 ships %d faces, want the 9 DRAWEXE ships (neither end is a face of its own: "+
			"one merges into its radial wall, the other runs out on x=80)", n)
	}
	if n := countLineagePrefix(body, "arcfillet:endcap"); n != 0 {
		t.Errorf("simple/N6 ships %d separate setback end-cap face(s), want 0 — the declined end RUNS OUT "+
			"on its wall; a setback triangle there reaches (77,14,10), inside the removed pocket", n)
	}
	const r = 5.0
	// The MERGED end's wall: 30 × 70 plus the corner the blend adds back. 2105.3650…, DRAWEXE 2105.37.
	assertFaceMeshesToClosedForm(t, body, "simple/N6", "import:step#16:face#3", 30*70+(r*r-stdmath.Pi*r*r/4))
	// The RUN-OUT end's three faces, against DRAWEXE 8.0.0 `explode result F` + `sprops _ 1.e-9`. They have
	// no closed form (the run-out boundary is a spiric quartic), so the oracle is OCCT's own number.
	for _, tc := range []struct {
		lineage string
		drawexe float64
	}{
		{"import:step#16:face#6", 1406.8},  // the x=80 wall the band runs out on (was 1400 with the cap)
		{"import:step#16:face#4", 641.965}, // the pocket floor, receded by the run-out (was 651.53)
		{"arcfillet:torus#0", 254.441},     // the band itself (was 243.51 over the un-terminated span)
	} {
		assertFaceMeshesToDrawexe(t, body, "simple/N6", tc.lineage, tc.drawexe, 1e-4)
	}
	if bad := ops.RetracingFaceLoops(body, ops.PropertyQuality()); len(bad) != 0 {
		t.Errorf("simple/N6 still retraces on %d face loop(s): %s", len(bad), describeRetracing(bad))
	}
}

// assertFaceMeshesToDrawexe compares one face's SHIPPED mesh area against DRAWEXE's own sprops number at
// a relative budget — the oracle to use where no closed form exists.
func assertFaceMeshesToDrawexe(t *testing.T, body *topo.Body, name, lineage string, drawexe, tol float64) {
	t.Helper()
	got := ops.MeshArea(shippedFaceMesh(t, body, faceByLineage(t, body, lineage)))
	if rel := stdmath.Abs(got-drawexe) / drawexe; rel > tol {
		t.Errorf("%s face %s meshes %.6g, DRAWEXE %.6g (rel %+.4f%%, tol %.1g)",
			name, lineage, got, drawexe, (got-drawexe)/drawexe*100, tol)
	}
}

// assertFaceMeshesToClosedForm compares one face's SHIPPED mesh area against its closed form.
func assertFaceMeshesToClosedForm(t *testing.T, body *topo.Body, name, lineage string, want float64) {
	t.Helper()
	got := ops.MeshArea(shippedFaceMesh(t, body, faceByLineage(t, body, lineage)))
	if rel := stdmath.Abs(got-want) / want; rel > arcEndCapPerFaceTol {
		t.Errorf("%s %s meshes %.8g, closed form %.8g (rel %+.6f%%, budget %.1g)",
			name, lineage, got, want, (got-want)/want*100, arcEndCapPerFaceTol)
	}
}

// countLineagePrefix counts the body's faces whose lineage key starts with prefix.
func countLineagePrefix(b *topo.Body, prefix string) int {
	n := 0
	for _, f := range b.Faces() {
		if strings.HasPrefix(f.Lineage().KeyString(), prefix) {
			n++
		}
	}
	return n
}
