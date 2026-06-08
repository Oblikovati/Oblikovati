// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
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
