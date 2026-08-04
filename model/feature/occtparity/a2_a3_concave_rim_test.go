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

// A2 and A3 are OCCT bfuseblend cases whose fused-solid boolean section rounds into a CONCAVE closed
// sphere/cone cap rim — the concave dual of J1's convex closed-rim torus band. They are greened by the
// concave closed-rim cove band (kernel/ops/fillet_curved_closed_rim_concave*.go): an exact external-
// tangency torus arm (ρ = R+r, cap plane pushed into the void), the concave contact-circle branch, a
// winding-flipped cove band, and an OUTWARD-growing plate hole. Here the plate is wide enough that the
// plane-contact rail does NOT spill (unlike blend/simple S2/S5, whose narrow ±15 plate makes the cove
// climb the side walls — a documented follow-on slice). This gate pins, WITHOUT relying only on the
// whole-body area, that each result is a watertight fold-free positive-volume solid carrying exactly one
// torus cove band, and that the summed mesh area equals OCCT's oracle within the corpus deps (0.01).

// concaveRimCase captures the oracle facts for one greened concave-rim bfuseblend case.
type concaveRimCase struct {
	name  string
	faces int
	area  float64 // OCCT whole-result area (corpus.json expectedArea)
}

func TestA2A3ConcaveClosedRimWatertight(t *testing.T) {
	t.Parallel()
	for _, tc := range []concaveRimCase{
		{"A2", 9, 432086},
		{"A3", 8, 389033},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := bfuseblendResultBody(t, tc.name)
			assertWatertight(t, tc.name, body, tc.faces)
			assertPositiveVolume(t, tc.name, body)
			assertOneToriusCoveBand(t, tc.name, body)
			assertConcaveRimMeshArea(t, tc.name, body, tc.area)
		})
	}
}

// bfuseblendResultBody imports a bfuseblend-grid STEP fixture and runs the real fillet feature, returning
// the single result solid. Fails (not skips) on an import/locate/fillet gap for these GREEN cases.
func bfuseblendResultBody(t *testing.T, name string) *topo.Body {
	t.Helper()
	var rec Record
	for _, r := range Corpus() {
		if r.Grid == "bfuseblend" && r.Case == name {
			rec = r
		}
	}
	body, err := importInput(filepath.Join(CorpusFixtureDir(), rec.InputStep))
	if err != nil {
		t.Fatalf("%s import-divergence: %v", name, err)
	}
	sets, ok := scoreLocate(rec, body)
	if !ok {
		t.Fatalf("%s picks could not be located on the imported body", name)
	}
	res, okFillet, reason := runFillet(body, sets)
	if !okFillet || len(res) != 1 || res[0] == nil {
		t.Fatalf("%s fillet unhealthy: ok=%v reason=%q results=%d", name, okFillet, reason, len(res))
	}
	return res[0]
}

// assertPositiveVolume fails if the result's signed volume is not positive — the exact regression the
// concave cove band's winding flip guards (an un-flipped band would fill the material and read ≤ 0).
func assertPositiveVolume(t *testing.T, name string, body *topo.Body) {
	t.Helper()
	if vol := ops.BodyGeometryProperties(body, ops.PropertyQuality()).Volume; vol <= 0 {
		t.Fatalf("%s result volume %.4f, want positive (an un-flipped concave band would read ≤ 0)", name, vol)
	}
}

// assertOneToriusCoveBand fails unless exactly one face is the exact geom.Torus cove band (the arm
// surface, not a fallback) and every face meshes fold-free to a finite positive area.
func assertOneToriusCoveBand(t *testing.T, name string, body *topo.Body) {
	t.Helper()
	bands := 0
	for _, f := range body.Faces() {
		m := ops.TessellateFace(f, ops.PropertyQuality())
		area := ops.MeshArea(m)
		if area <= 0 || stdmath.IsInf(area, 0) || stdmath.IsNaN(area) {
			t.Fatalf("%s %T face meshed to %.4f, want a finite positive area", name, f.Geometry(), area)
		}
		assertFaceFoldFreeAtEveryQuality(t, name, f, m)
		if _, ok := f.Geometry().(geom.Torus); ok {
			bands++
		}
	}
	if bands != 1 {
		t.Fatalf("%s has %d torus cove-band faces, want exactly 1", name, bands)
	}
}

// assertConcaveRimMeshArea asserts the summed per-face mesh area equals OCCT's whole-result area within
// the corpus deps (0.01 relative) — a mis-receded host or a wrong-region cove band would inflate it.
func assertConcaveRimMeshArea(t *testing.T, name string, body *topo.Body, want float64) {
	t.Helper()
	total := 0.0
	for _, f := range body.Faces() {
		total += ops.MeshArea(ops.TessellateFace(f, ops.PropertyQuality()))
	}
	if rel := stdmath.Abs(total-want) / want; rel > 0.01 {
		t.Fatalf("%s total mesh area %.2f, want OCCT %.0f within deps 0.01 (rel %.5f)", name, total, want, rel)
	}
}
