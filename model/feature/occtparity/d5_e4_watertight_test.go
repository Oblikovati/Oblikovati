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
	for _, tc := range []struct {
		name       string
		faces      int
		hostSphere float64 // expected tessellated area of the large host-sphere face (~sphere minus the bite)
	}{
		{"D5", 9, 57815},
		{"E4", 8, 57609},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := caseResultBody(t, tc.name)
			assertWatertight(t, tc.name, body, tc.faces)
			assertHostSphereRegion(t, tc.name, body, tc.hostSphere)
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
