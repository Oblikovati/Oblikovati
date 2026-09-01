// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// The patchGridCap corpus gates (patchgridcap-report.md). spherePatchMesh's silent per-axis
// patchGridCap=80 floored the interior grid density of 25 shipped simple-grid cases at
// PropertyQuality (census: A2 A6 A8 B1 B9 C6 C8 D1–D9 E1–E4 F7 K6 L4 L9 N1, plus C2 at display
// quality), holding host-sphere areas up to −200.4 (simple/D6, vs live DRAWEXE `sprops … 1.e-12`
// 183647) FLAT at every swept chord tolerance — the same plateau-misread-as-converged failure the
// S6/S7 sphere slice documented. The grid now honours the chord tolerance up to an explicit,
// DIAGNOSED cell budget (ops.patchGridCellBudget); these two gates pin the corpus-visible halves:
//
//   - CONVERGENCE (A2's corner-ball octant, budget-honoured at every swept tolerance): the deficit
//     against the octant closed form 4π·10²/8 = 50π SHRINKS with the tolerance. With the clamp
//     restored it reads −0.0216 flat at 1e-3 AND 2.5e-4 and the shrink ratio fails.
//   - RESOLUTION CEILING (D2's R=150 host sphere, budget-SCALED at PropertyQuality — 801×327 of the
//     1341×547 steps asked, diagnosed): the mesh area lands within hostSphereD2AreaCeil of DRAWEXE's
//     61215.7 (`sprops result_4 1.e-12`, stable across quadrature tolerances; our lifted-clamp sweep
//     converges onto the same 61215.71). The clamped mesh read 61199.49 (−16.21) and fails loud.
const (
	cornerOctantShrink   = 0.6  // deficit(finer ct) must be < this × deficit(coarser ct): converging, not flat
	hostSphereD2AreaCeil = 0.26 // 1.2× the measured post-fix |Δ| 0.211 (incl. DRAWEXE's 6-s.f. print), abs units²
	hostSphereD2Drawexe  = 61215.7
)

func TestCornerOctantDeficitConverges(t *testing.T) {
	t.Parallel()
	body, ok := shippedCaseBody(caseRecord(t, "simple", "A2"), CorpusFixtureDir())
	if !ok {
		t.Fatal("A2: no shipped body")
	}
	f := singleSmallSphereFace(t, "A2", body)
	closed := 50 * stdmath.Pi // the r=10 corner-ball octant: 4πr²/8
	prev := stdmath.Inf(1)
	for _, ct := range []float64{1e-2, 1e-3, 2.5e-4} {
		m := tessellate.TessellateFace(f, ops.Quality{ChordTolerance: ct, AngleTolerance: stdmath.Pi / 180})
		if hasCapSaturated(m) {
			t.Fatalf("A2 octant saturated the cell budget at ct=%g — the convergence fixture must stay honoured", ct)
		}
		d := stdmath.Abs(ops.MeshArea(m) - closed)
		if d > cornerOctantShrink*prev {
			t.Errorf("A2 octant deficit plateaus: %.6f → %.6f across ct→%g (want < %.1f×) — a density floor is back",
				prev, d, ct, cornerOctantShrink)
		}
		prev = d
	}
}

func TestHostSphereD2MeshesWithinBudgetCeil(t *testing.T) {
	t.Parallel()
	body, ok := shippedCaseBody(caseRecord(t, "simple", "D2"), CorpusFixtureDir())
	if !ok {
		t.Fatal("D2: no shipped body")
	}
	for _, f := range body.Faces() {
		sph, isSph := f.Geometry().(geom.Sphere)
		if !isSph || sph.Radius < 100 {
			continue
		}
		got := ops.MeshArea(tessellate.TessellateFace(f, ops.PropertyQuality()))
		if d := stdmath.Abs(got - hostSphereD2Drawexe); d > hostSphereD2AreaCeil {
			t.Errorf("D2 host sphere meshes %.6f vs DRAWEXE %.1f (|Δ|=%.4f > %.2f) — the density floor is back",
				got, hostSphereD2Drawexe, d, hostSphereD2AreaCeil)
		}
		return
	}
	t.Fatal("D2: no R=150 host sphere face")
}

// singleSmallSphereFace returns the case's single fillet-radius (r ≤ 10) sphere face — the corner
// ball octant, identified by surface type + radius, never by mesh area (perface_oracle_test.go ★★).
func singleSmallSphereFace(t *testing.T, name string, body *topo.Body) *topo.Face {
	t.Helper()
	var found *topo.Face
	for _, f := range body.Faces() {
		if sph, ok := f.Geometry().(geom.Sphere); ok && sph.Radius <= 10 {
			if found != nil {
				t.Fatalf("%s: more than one corner-ball sphere face", name)
			}
			found = f
		}
	}
	if found == nil {
		t.Fatalf("%s: no corner-ball sphere face", name)
	}
	return found
}

// hasCapSaturated reports whether the mesh carries the tessellate.cap-saturated diagnostic.
func hasCapSaturated(m *ops.Mesh) bool {
	for _, d := range m.Diagnostics {
		if d.Code == tessellate.CodeTessellateCapSaturated {
			return true
		}
	}
	return false
}
