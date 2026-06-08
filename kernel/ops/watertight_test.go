// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"os"
	"testing"

	"oblikovati/kernel/brep"
	"oblikovati/math"
)

// freeEdgeCount welds coincident vertices, then counts edges not shared by exactly two triangles —
// the watertightness metric (0 = closed manifold). Welding is needed because TessellateBody copies
// shared-edge vertices per face (mergeMesh offsets indices).
func freeEdgeCount(m *Mesh) int {
	q := func(x float64) int64 {
		if x < 0 {
			return int64(x*1e6 - 0.5)
		}
		return int64(x*1e6 + 0.5)
	}
	canon := map[[3]int64]int{}
	weld := make([]int, len(m.Positions))
	for i, p := range m.Positions {
		k := [3]int64{q(float64(p.X)), q(float64(p.Y)), q(float64(p.Z))}
		if c, ok := canon[k]; ok {
			weld[i] = c
		} else {
			canon[k], weld[i] = i, i
		}
	}
	type edge struct{ a, b int }
	deg := map[edge]int{}
	for t := 0; 3*t+2 < len(m.Indices); t++ {
		v := [3]int{weld[m.Indices[3*t]], weld[m.Indices[3*t+1]], weld[m.Indices[3*t+2]]}
		for k := 0; k < 3; k++ {
			a, b := v[k], v[(k+1)%3]
			if a > b {
				a, b = b, a
			}
			deg[edge{a, b}]++
		}
	}
	free := 0
	for _, d := range deg {
		if d != 2 {
			free++
		}
	}
	return free
}

// TestCleanCurvedSolidIsWatertight pins the invariant that a clean curved solid tessellates watertight
// — every interior edge shared by exactly two triangles. The curved-face meshers (structuredGridMesh,
// gridPatchMesh) produce matching boundaries because the edges lie on their surfaces, so a face's
// PointAt(ParamAt(edge)) equals the shared edge point. On IMPORTED geometry the edges sit ~mm off the
// surfaces, breaking this — the watertightness M25 PBI-324 (edge snapping) / PBI-330 targets. This is
// the regression guard for the clean case.
func TestCleanCurvedSolidIsWatertight(t *testing.T) {
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 10)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	mesh, _ := TessellateBody(cyl, DefaultQuality())
	if free := freeEdgeCount(mesh); free != 0 {
		t.Errorf("clean cylinder solid tessellated with %d free edges; want 0 (watertight)", free)
	}
}

// TestImportedNurbsDuctWatertight guards the M25 fix that made imported NURBS faces conform to their
// neighbours: concatLoopPcurve projects each boundary loop's pcurve in ONE continuous march, so a
// self-proximal NURBS face (the EDF bell-mouth lip) no longer lands edge-starts on the wrong sheet,
// self-intersect in (u,v) and make the CDT silently drop boundary constraints — which used to crack
// the duct body (28 free edges) against its cone/cylinder/plane neighbours. Env-gated on the heavy
// model (not a committed fixture; same OBK_PERF_STEP convention as TestHeavyModelBudget). It also guards
// the cross-face conformance repair (conformCylConeFaces): a trimmed cyl/cone whose plane ear-clip
// absorbed a rim-arc point is re-meshed with the metric-(u,v) CDT so it conforms to its neighbour.
// Per-body free edges across the M25 fixes: pcurve continuity 4/0/4/28/0/0 → conformance repair
// 4/0/0/0/0/0 (total 4). The remaining 4 are body0's plane-side absorber (a separate future pass). Assert
// the TOTAL stays ≤4 — it regresses hard (to 8, then 36) if either fix is lost.
func TestImportedNurbsDuctWatertight(t *testing.T) {
	path := os.Getenv("OBK_PERF_STEP")
	if path == "" {
		t.Skip("set OBK_PERF_STEP=/path/EDF.STEP to run the imported-NURBS watertightness guard")
	}
	total := 0
	for i, body := range stepBodies(t, path) {
		mesh, _ := TessellateBody(body, DefaultQuality())
		free := freeEdgeCount(mesh)
		total += free
		if free > 4 {
			t.Errorf("imported body %d tessellated with %d free edges; want ≤4 "+
				"(a pcurve-continuity or cyl/cone conformance regression cracks a body)", i, free)
		}
	}
	if total > 4 {
		t.Errorf("imported model total free edges = %d; want ≤4 (watertightness regression)", total)
	}
}
