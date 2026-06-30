// SPDX-License-Identifier: GPL-2.0-only

package step_test

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/ops"
)

// TestRadialBoreThroughCurvedWallWatertight is the regression for Oblikovati#1510: a radial through-bore
// piercing a closed (periodic) B-spline barrel wall puts one bore mouth on the surface SEAM, which made
// the planar seam-cut trim mesher drop boundary constraints and leave the wall cracked and grossly
// under-enclosed (89 free edges, ~1.4k volume vs OCC's ~5.0k). The covering-space periodic mesher must now
// tessellate it watertight (0 free edges, 0 folds) with the volume converging to the getMass oracle.
func TestRadialBoreThroughCurvedWallWatertight(t *testing.T) {
	const occMass = 5023.5696
	data, err := os.ReadFile(filepath.Join("testdata", "occ", "cand_radial.step"))
	if err != nil {
		t.Fatalf("read cand_radial.step: %v", err)
	}
	bodies, _, err := step.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	if err != nil {
		t.Fatalf("import cand_radial: %v", err)
	}
	if len(bodies) != 1 {
		t.Fatalf("imported %d solids, want 1", len(bodies))
	}
	b := bodies[0]
	mesh, _ := ops.TessellateBody(b, ops.DefaultQuality())
	if free := openEdgeCount(mesh); free != 0 {
		t.Errorf("tessellation has %d free edges; want 0 (non-watertight curved wall)", free)
	}
	if folds := ops.FoldEdgeCount(mesh); folds != 0 {
		t.Errorf("tessellation has %d fold edges; want 0", folds)
	}
	vol := ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume
	if rel := (vol - occMass) / occMass; rel < -0.03 || rel > 0.03 {
		t.Errorf("volume %.2f vs OCC %.2f (rel %.4f); want within 3%%", vol, occMass, rel)
	}
}

// openEdgeCount welds coincident vertices, then counts mesh edges not shared by exactly two triangles —
// the watertightness metric (0 = closed manifold). It welds because TessellateBody copies shared-edge
// vertices per face (mergeMesh offsets indices), mirroring kernel/ops' own freeEdgeCount helper.
func openEdgeCount(m *ops.Mesh) int {
	q := func(x float64) int64 { return int64(x*1e6 + 0.5) }
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
	deg := map[[2]int]int{}
	for t := 0; 3*t+2 < len(m.Indices); t++ {
		v := [3]int{weld[m.Indices[3*t]], weld[m.Indices[3*t+1]], weld[m.Indices[3*t+2]]}
		for k := 0; k < 3; k++ {
			a, b := v[k], v[(k+1)%3]
			if a > b {
				a, b = b, a
			}
			deg[[2]int{a, b}]++
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
