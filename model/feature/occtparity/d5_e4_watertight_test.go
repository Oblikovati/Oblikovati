// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// FR4 whole-body gate on the REAL D5/E4 STEP bodies — the first corpus corners whose curved-arm weld
// rolls an OBLIQUE torus arm on a sphere HOST. It complements the tessellation-area gate
// TestOCCTBlendSimple/{D5,E4}: it asserts, WITHOUT tessellating, that the oblique-aware corner-host retrim
// welds a watertight solid (every edge 2-incident, valid closed orientable, oracle face count), and — the
// crux of the S7 host-sphere region — that the trimmed geom.Sphere HOST face carries the small corner
// bite, not its ~5×-larger complement. A regression that mis-winds the host sphere (the pre-FR4 decline,
// or the sphere-patch mesher filling the complement) fails the host-sphere-area assertion loud and fast.
func TestD5E4WholeBodyWatertight(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		faces        int
		hostSphere   float64 // expected tessellated area of the large host-sphere face (~sphere minus the bite)
		cornerSphere float64 // expected tessellated area of the SMALL corner-blend sphere patch (0 = only assert finite positive)
		torusArms    int     // expected number of trimmed torus ARM faces (each must mesh to a finite positive area)
	}{
		// cornerSphere 55.7891 is the brief-mandated analytic area of the D5 corner-blend spherical
		// triangle; the tessellation lands ~55.784 (a convex-patch under-estimate, ~0.01% < the 2% tol).
		{"D5", 9, 57815, 55.7891, 2},
		{"E4", 8, 57609, 0, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := caseResultBody(t, tc.name)
			assertWatertight(t, tc.name, body, tc.faces)
			assertHostSphereRegion(t, tc.name, body, tc.hostSphere)
			assertCornerSphereArea(t, tc.name, body, tc.cornerSphere)
			assertTorusArmFaces(t, tc.name, body, tc.torusArms)
		})
	}
}

// caseResultBody imports a simple-grid STEP fixture and runs the real fillet feature, returning the single
// result solid. Skips (not fails) when the case does not import/locate on this build.
func caseResultBody(t *testing.T, name string) *topo.Body {
	t.Helper()
	var rec Record
	for _, r := range Corpus() {
		if r.Grid == "simple" && r.Case == name {
			rec = r
		}
	}
	body, err := importInput(filepath.Join(CorpusFixtureDir(), rec.InputStep))
	if err != nil {
		t.Skipf("%s import-divergence (not a fillet defect): %v", name, err)
	}
	sets, ok := scoreLocate(rec, body)
	if !ok {
		t.Skipf("%s picks could not be located on the imported body", name)
	}
	res, okFillet, reason := runFillet(body, sets)
	if !okFillet || len(res) != 1 || res[0] == nil {
		t.Fatalf("%s fillet unhealthy: ok=%v reason=%q results=%d", name, okFillet, reason, len(res))
	}
	return res[0]
}

// assertWatertight checks the body is a closed manifold solid with the oracle face count.
func assertWatertight(t *testing.T, name string, body *topo.Body, wantFaces int) {
	t.Helper()
	if got := len(body.Faces()); got != wantFaces {
		t.Fatalf("%s result has %d faces, want the oracle's %d", name, got, wantFaces)
	}
	for _, e := range body.Edges() {
		if n := len(e.Uses()); n != 2 {
			t.Fatalf("%s edge %d is %d-incident (%v→%v), want exactly 2 (a watertight manifold solid)",
				name, e.ID(), n, e.StartVertex().Point(), e.EndVertex().Point())
		}
	}
	rep := ops.Validate(body)
	if !rep.Valid || !rep.Closed || !rep.HolesContained || !body.IsSolid() {
		t.Fatalf("%s result not a watertight solid: valid=%v closed=%v holes=%v solid=%v issues=%v",
			name, rep.Valid, rep.Closed, rep.HolesContained, body.IsSolid(), rep.Issues)
	}
}

// assertHostSphereRegion checks the LARGE host-sphere face (radius 150) tessellates to the corner-bite
// region, not its complement — the S7-gap regression the FR4 orient-for-sphere-host seed fixes. The
// complement would read ~224929 (nearly the whole 282743 sphere), 4× over.
func assertHostSphereRegion(t *testing.T, name string, body *topo.Body, want float64) {
	t.Helper()
	for _, f := range body.Faces() {
		sph, ok := f.Geometry().(geom.Sphere)
		if !ok || sph.Radius < 100 { // skip the small corner-blend sphere (radius = fillet r)
			continue
		}
		if a := faceMeshArea2(f); stdmath.Abs(a-want) > 0.02*want {
			t.Fatalf("%s host-sphere region area %.1f, want ~%.0f (a complement-fill reads ~224929)", name, a, want)
		}
		return
	}
	t.Fatalf("%s carries no host-sphere face (radius 150)", name)
}

// assertCornerSphereArea checks the SMALL corner-blend sphere patch (radius = fillet r, the analytic
// geom.Sphere corner triangle, distinct from the radius-150 host sphere) meshes to its brief-mandated
// area. want==0 asserts only a finite positive area (a corner sphere that meshed the wrong region or
// collapsed would read a wildly different or zero area). The relative tol mirrors assertHostSphereRegion.
func assertCornerSphereArea(t *testing.T, name string, body *topo.Body, want float64) {
	t.Helper()
	f := cornerBlendSphereFace(body)
	if f == nil {
		t.Fatalf("%s carries no corner-blend sphere face (radius < 100)", name)
	}
	a := faceMeshArea2(f)
	if a <= 0 || stdmath.IsInf(a, 0) || stdmath.IsNaN(a) {
		t.Fatalf("%s corner-blend sphere area %.4f is not finite positive", name, a)
	}
	if want > 0 && stdmath.Abs(a-want) > 0.02*want {
		t.Fatalf("%s corner-blend sphere area %.4f, want ~%.4f (a mis-wound/collapsed corner reads far off)", name, a, want)
	}
}

// cornerBlendSphereFace returns the corner-blend sphere face — the one geom.Sphere whose radius is the
// fillet r (< 100), located by the same radius cut assertHostSphereRegion uses to skip it.
func cornerBlendSphereFace(body *topo.Body) *topo.Face {
	for _, f := range body.Faces() {
		if sph, ok := f.Geometry().(geom.Sphere); ok && sph.Radius < 100 {
			return f
		}
	}
	return nil
}

// assertTorusArmFaces checks the trimmed torus ARM faces (every geom.Torus face — the host here is a
// sphere+planes, so no torus is a host) number the oracle count and each meshes to a finite positive
// area. The exact trimmed-torus area is not cheaply derivable from the result body alone (the arm's
// four bounding rails are not carried here), so this pins the COUNT and finite-positivity per the brief.
func assertTorusArmFaces(t *testing.T, name string, body *topo.Body, want int) {
	t.Helper()
	got := 0
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(geom.Torus); !ok {
			continue
		}
		got++
		if a := faceMeshArea2(f); a <= 0 || stdmath.IsInf(a, 0) || stdmath.IsNaN(a) {
			t.Fatalf("%s torus-arm face meshed to %.4f, want a finite positive area", name, a)
		}
	}
	if got != want {
		t.Fatalf("%s has %d torus-arm faces, want %d", name, got, want)
	}
}

// faceMeshArea2 sums the triangle areas of a face's Property-quality tessellation.
func faceMeshArea2(f *topo.Face) float64 {
	m := ops.TessellateFace(f, ops.PropertyQuality())
	if m == nil {
		return 0
	}
	area := 0.0
	for i := 0; i+2 < len(m.Indices); i += 3 {
		p, q, r := m.Positions[m.Indices[i]], m.Positions[m.Indices[i+1]], m.Positions[m.Indices[i+2]]
		area += 0.5 * float64(p.VectorTo(q).Cross(p.VectorTo(r)).Length())
	}
	return area
}
