// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	m "oblikovati.org/math"
)

// finPlateCuts hollow out a 10×10×2 plate so that one wall of the result stands 3 mm of material away
// from open space on one side and 4 mm of gap from the rest of the plate on the other. Both cuts stop at
// y = 9, so everything stays ONE shell.
//
// Those two lengths are what the fixture is for: the orientation pass probes a face by stepping a
// stand-off scaled to the whole SHELL — 1e-3 of an 11.8 mm diagonal, so ~12 mm — either side of it. On
// this wall the outward step lands past the gap, inside the plate, and the inward step lands past the
// fin, in open space, so the probe reads the wall exactly backwards.
func finPlateCuts() [][2]m.Point3 {
	return [][2]m.Point3{
		{m.P3(-1, -1, -1), m.P3(3.993, 9, 3)},
		{m.P3(3.996, -1, -1), m.P3(4.0, 9, 3)},
	}
}

// finPlate applies finPlateCuts to the 10×10×2 plate, one at a time.
func finPlate(t *testing.T) *topo.Body {
	t.Helper()
	body, err := brep.SolidBlock(m.P3(0, 0, 0), m.P3(10, 10, 2), "plate")
	if err != nil {
		t.Fatalf("plate: %v", err)
	}
	for i, c := range finPlateCuts() {
		tool, err := brep.SolidBlock(c[0], c[1], "cut")
		if err != nil {
			t.Fatalf("cut tool %d: %v", i, err)
		}
		if body, err = brep.Boolean(brep.Difference, body, tool); err != nil {
			t.Fatalf("cut %d: %v", i, err)
		}
	}
	return body
}

// TestThinFinWallKeepsTheSenseItsLoopsCarry is the corpus row for the second half of
// Oblikovati/Oblikovati#3512: the geometric probe must not override a face's own loop handedness.
//
// A shell whose loops are traversal-consistent carries ONE orientation, up to a single global bit. The
// probe's job is to settle that bit, and orientFaceSigns used to let it decide each face outright
// instead: where its shell-scaled stand-off cannot resolve a local feature — the wall below, and the
// 0.6 µm slot wall the Inventor multipoint disk's Extrusion5 hands it — it wrote a sense contradicting
// that face's own loops, and the boolean's emission gate (brep.FaceWindingConsistent) then refused a
// correct body with boolean.winding-reject.
//
// Requicha-exact: 10·10·2 − 3.993·9·2 − 0.004·9·2 = 128.054.
func TestThinFinWallKeepsTheSenseItsLoopsCarry(t *testing.T) {
	body := finPlate(t)
	if !Validate(body).ValidSolid() {
		t.Fatalf("fin plate is not a valid closed solid: %v", Validate(body).Issues)
	}
	if got, want := len(body.Shells()), 1; got != want {
		t.Errorf("shell count = %d, want %d — the cuts stop short of y = 10, so nothing is severed", got, want)
	}
	assertEveryFaceWoundOutward(t, body)
	assertFinPlateVolume(t, body)
}

// finPlateVolume is the plate less the two through-cuts, which do not overlap.
const finPlateVolume = 10*10*2 - 3.993*9*2 - 0.004*9*2

// assertFinPlateVolume checks the Requicha volume against the ANALYTIC B-rep, which is the oracle: an
// oracle that gates a result has to be more exact than the result it gates, and a mesh is not. A face
// whose stored sense contradicts its loops integrates with the wrong sign here.
func assertFinPlateVolume(t *testing.T, body *topo.Body) {
	t.Helper()
	terms, ok := query.AnalyticBodyTerms(body)
	if !ok {
		t.Fatalf("the fin plate has no analytic volume")
	}
	if stdmath.Abs(terms.Vol-finPlateVolume) > 1e-9 { // tol:calibrated — exact planar faces, only rounding
		t.Errorf("analytic volume = %g, want %g (Requicha: plate less two through-cuts)", terms.Vol, finPlateVolume)
	}
	assertFinPlateMesh(t, body)
}

// assertFinPlateMesh is the secondary check: the same volume at both facetings, plus watertightness. A
// gate that holds at one faceting is measuring the tessellation, not the geometry — and an inside-out
// face shows in the mesh through a different code path than the analytic integrator's.
func assertFinPlateMesh(t *testing.T, body *topo.Body) {
	t.Helper()
	for _, gq := range gateQualities() {
		mesh, _ := tessellate.TessellateBody(body, gq.q)
		if n := freeEdgeCount(mesh); n != 0 {
			t.Errorf("%s faceting: %d free edges, want a watertight mesh", gq.name, n)
		}
		v := tessellate.MeshGeometryProperties(mesh).Volume
		if stdmath.Abs(v-finPlateVolume) > 1e-6 { // tol:calibrated — planar facets are exact here
			t.Errorf("%s faceting: mesh volume = %g, want %g", gq.name, v, finPlateVolume)
		}
	}
}
