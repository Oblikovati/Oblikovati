// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"slices"
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// rimHostSeamOnFaceTol is the budget for "the re-aimed host seam lies on its own host", relative to the
// bounding diagonal. Held here at the corpus-wide default 1e-6 with NO debt entry: the carried meridian
// measures 2.7e-14 (J2) / 1.6e-13 (J4) relative, while the chord it replaces measured 0.3035 / 0.1499 —
// eight decades outside, so this is a construction/arithmetic separator, not a tolerance to tune.
const rimHostSeamOnFaceTol = 1e-6

// rimHostSeamPerFaceTol is the per-face DRAWEXE budget for these two bodies. Measured worst 5.6e-5 (J2's
// 1948.84 fillet band, mesh quantization); the defect it protects against measures +317% on J4's torus host
// face, four decades out. 2e-4 leaves ~3.5× headroom.
const rimHostSeamPerFaceTol = 2e-4

// TestRimHostSeamStaysOnItsHost is the oracle gate for the rim-fillet HOST-SEAM carry
// (fillet_rim_build.go's wallSeamCurve → retainedHostSeamCurve).
//
// The rim rebuild recedes the curved host and re-aims its axial SEAM edge at the new contact vertex. It used
// to rebuild that seam as a straight LineSegment, which is the host's own meridian only on a cylinder, cone
// or elliptical cylinder — on a SPHERE (J2) or a TORUS (J4) the meridian is an ARC, so the shipped seam was
// a 90.38 / 61.24-long chord sitting 28.44 / 10.43 off the very face it bounds. J2 was the corpus's LARGEST
// off-surface residual (rel 0.303) and J4 tiled its torus host at 394781 against DRAWEXE's 94600.6, i.e.
// +70.2% whole-body area (FAIL(area)). Three assertions, each falsifiable on its own:
//
//	(a) the `rimfillet:wallseam` edge is NOT straight on a non-ruled host, and lies ON that host;
//	(b) every face's mesh area equals DRAWEXE's own, face by face, rank-paired;
//	(c) every boundary edge lies on the face it bounds, at the default budget with no debt entry.
//
// Fixtures are OCCT's own `psphere s 5 -90 45` (J2 — a spherical zone reaching an enclosed pole) and
// `ptorus s 20 5 0 90` (J4 — a quarter torus), both `tscale SCALE1=10` and blended r=10 on one rim.
func TestRimHostSeamStaysOnItsHost(t *testing.T) {
	t.Parallel()
	for _, tc := range rimHostSeamCases() {
		t.Run(tc.name, func(t *testing.T) {
			body := pinnedBody(t, "simple", tc.name)
			assertHostSeamIsItsHostMeridian(t, tc.name, body)
			assertPerFaceAgainstDrawexe(t, tc, body)
			assertLoopSegmentsOnFaces(t, Record{Grid: "simple", Case: tc.name}, body, rimHostSeamOnFaceTol)
		})
	}
}

// rimHostSeamCases is the swept population: the corpus's only two rim-fillet hosts whose meridian is not a
// straight ruling. DRAWEXE 8.0.0 per-face areas (`explode result F` + `sprops result_i 1e-6`), descending.
// Rank pairing is sound on both: J2's closest pair of face areas differs by 54%, J4's by 33%, so a
// size-ordered pairing cannot mis-associate two faces.
func rimHostSeamCases() []drawexeFaceCase {
	return []drawexeFaceCase{
		{name: "J2", totalArea: 30620.7, perFaceTol: rimHostSeamPerFaceTol,
			drawexe: []float64{25665, 3006.84, 1948.84}},
		{name: "J4", totalArea: 427447, perFaceTol: rimHostSeamPerFaceTol,
			drawexe: []float64{179045, 125664, 94600.6, 28137.3}},
	}
}

// assertHostSeamIsItsHostMeridian fails when the rebuild's re-aimed host seam is still a straight chord, or
// when it does not lie on the host face it bounds. It asserts "not straight" only for a host whose meridian
// genuinely is not a ruling (sphere / torus), because a cylinder, cone or elliptical-cylinder host's seam IS
// a straight ruling and must stay one — that is the byte-identity half of the same fix.
func assertHostSeamIsItsHostMeridian(t *testing.T, name string, b *topo.Body) {
	t.Helper()
	seen := 0
	for _, e := range b.Edges() {
		if !strings.Contains(e.Lineage().KeyString(), "wallseam") {
			continue
		}
		seen++
		assertSeamOnEveryHost(t, name, b, e)
	}
	if seen != 1 {
		t.Fatalf("%s: found %d rimfillet:wallseam edges, want exactly 1 (the rebuild re-aims one host seam)", name, seen)
	}
}

// assertSeamOnEveryHost checks the seam edge against each face it bounds: never straight on a non-ruled
// host, and always within rimHostSeamOnFaceTol of that face's own surface.
func assertSeamOnEveryHost(t *testing.T, name string, b *topo.Body, e *topo.Edge) {
	t.Helper()
	for _, f := range facesBoundedBy(b, e) {
		if _, straight := e.Geometry().(geom.LineSegment); straight && !surfaceIsRuled(f.Geometry()) {
			t.Errorf("%s: the re-aimed host seam (edge %d) is still a straight CHORD across a %T host (face %d) — the meridian was not carried",
				name, e.ID(), f.Geometry(), f.ID())
		}
		d, ok := curveOffSurface(e.Geometry(), f.Geometry())
		if ok && d/boundingDiag(b) > rimHostSeamOnFaceTol {
			t.Errorf("%s: the re-aimed host seam (edge %d, %T) leaves its own host (face %d, %T) by %.6g",
				name, e.ID(), e.Geometry(), f.ID(), f.Geometry(), d)
		}
	}
}

// surfaceIsRuled reports whether the surface's meridian through a point is a straight line, so a straight
// re-aimed seam is CORRECT on it (a plane, and the three extrusion-like quadrics the corpus's rim hosts use).
func surfaceIsRuled(s geom.Surface) bool {
	switch s.(type) {
	case geom.Plane, geom.Cylinder, geom.Cone, geom.EllipticalCylinder:
		return true
	}
	return false
}

// facesBoundedBy returns every face of b whose loops use e.
func facesBoundedBy(b *topo.Body, e *topo.Edge) []*topo.Face {
	var out []*topo.Face
	for _, f := range b.Faces() {
		if f.Geometry() == nil {
			continue
		}
		if slices.Contains(boundaryEdgesOf(f), e) {
			out = append(out, f)
		}
	}
	return out
}
