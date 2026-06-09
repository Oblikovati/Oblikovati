// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// sphereGrid builds the full-domain (u,v) breakpoints a bare sphere face tessellates over, matching
// fullDomainGridMesh's sampling.
func sphereGrid(s geom.Sphere, q Quality) (us, vs []float64) {
	uLo, uHi := s.UDomain()
	vLo, vHi := s.VDomain()
	us = adaptiveParams(func(u float64) math.Point3 { return s.PointAt(u, (vLo+vHi)/2) }, uLo, uHi, q.tol(), q.angleTol())
	vs = adaptiveParams(func(v float64) math.Point3 { return s.PointAt((uLo+uHi)/2, v) }, vLo, vHi, q.tol(), q.angleTol())
	return us, vs
}

// TestClosedDomainMeshSphereWatertight pins the M25 PBI-330 fix: a whole sphere (a closed, periodic,
// pole-capped surface) meshes WATERTIGHT — the seam wraps and each pole is one shared vertex, so no
// edge is left unpaired. A naive UV grid duplicated the seam and degenerated the poles (66 free edges).
func TestClosedDomainMeshSphereWatertight(t *testing.T) {
	s, err := geom.NewSphere(math.P3(0, 0, 0), 5)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	us, vs := sphereGrid(s, DefaultQuality())
	m := closedDomainMesh(s, us, vs)
	if free := freeEdgeCount(m); free != 0 {
		t.Errorf("sphere meshed with %d free edges; want 0 (watertight)", free)
	}
}

// TestClosedDomainMeshSphereArea pins that the watertight sphere mesh has ~the analytic surface area
// (a chord-faceted inscribed mesh is slightly under 4πr²) — i.e. wrapping the seam and collapsing the
// poles did not drop or double-count any area.
func TestClosedDomainMeshSphereArea(t *testing.T) {
	const r = 5.0
	s, err := geom.NewSphere(math.P3(0, 0, 0), r)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	us, vs := sphereGrid(s, DefaultQuality())
	got := meshGeometryProperties(closedDomainMesh(s, us, vs)).Area
	want := 4 * stdmath.Pi * r * r
	if rel := stdmath.Abs(got-want) / want; rel > 0.03 {
		t.Errorf("sphere mesh area %.3f; want ~%.3f (4πr²), off by %.1f%%", got, want, rel*100)
	}
}

// TestImportedAnalyticPrimitivesWatertight pins M25 PBI-330 Phases 1–2b: every imported solid bounded by
// closed/periodic analytic faces tessellates watertight (0 free edges). Each previously leaked — sphere
// 66 (seam+poles, Phase 1), cylinder 4 and cone_sharp 33 (the band dropped its seam cell, Phase 2a),
// filleted_box 36 (pole-degenerate sphere corner-fillet caps the CDT tore, Phase 2b) — and the caps must
// still pair with their neighbours. Boundary-trimmed primitives (partial_*, drilled/chamfered box) guard
// that the fixes did not break the non-closed paths.
func TestImportedAnalyticPrimitivesWatertight(t *testing.T) {
	for _, name := range []string{
		"sphere", "cylinder", "cone_frustum", "cone_sharp", "torus",
		"partial_cylinder", "drilled_box", "chamfered_box", "box_with_boss",
		"filleted_box", "partial_sphere",
	} {
		data, err := os.ReadFile(filepath.Join("..", "exchange", "step", "testdata", "occ", name+".step"))
		if err != nil {
			t.Fatalf("read %s.step: %v", name, err)
		}
		bodies, _, err := step.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
		if err != nil || len(bodies) == 0 {
			t.Fatalf("import %s.step: %v (n=%d)", name, err, len(bodies))
		}
		total := 0
		for _, b := range bodies {
			mesh, _ := TessellateBody(b, DefaultQuality())
			total += freeEdgeCount(mesh)
		}
		if total != 0 {
			t.Errorf("%s tessellated with %d free edges; want 0 (watertight)", name, total)
		}
	}
}

// TestClosedDomainMeshOpenSurfaceIsPlainGrid pins the no-op case: a non-closed surface (a clamped plane
// patch — no periodic seam, no pole) tessellates as a plain (cols-1)×(rows-1)×2 grid, unchanged.
func TestClosedDomainMeshOpenSurfaceIsPlainGrid(t *testing.T) {
	p, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	us := []float64{0, 1, 2, 3}
	vs := []float64{0, 1, 2}
	m := closedDomainMesh(p, us, vs)
	wantTris := (len(us) - 1) * (len(vs) - 1) * 2
	if got := len(m.Indices) / 3; got != wantTris {
		t.Errorf("open plane grid produced %d triangles; want %d (plain grid, no wrap/pole)", got, wantTris)
	}
}
