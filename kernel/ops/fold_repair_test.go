// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

func countFolds(m *Mesh) int {
	c := 0
	for _, ts := range edgeTriMap(m) {
		if len(ts) == 2 && trianglesFold(m, ts[0], ts[1]) {
			c++
		}
	}
	return c
}

func TestRepairFoldsFixesAFold(t *testing.T) {
	t.Parallel()
	// A non-planar quad triangulated on the diagonal that folds (normals oppose); flipping to the
	// other diagonal removes the fold.
	m := &Mesh{
		Positions: []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 1), math.P3(2, 0, 0), math.P3(1, 1, 1)},
		Normals:   []math.Vector3{math.V3(0, 1, 0), math.V3(0, 1, 0), math.V3(0, 1, 0), math.V3(0, 1, 0)},
		Indices:   []int{0, 1, 2, 0, 2, 3},
	}
	if countFolds(m) == 0 {
		t.Fatal("test setup should contain a fold")
	}
	folded := totalTriArea(m)
	repairFolds(m, 4)
	if c := countFolds(m); c != 0 {
		t.Errorf("repairFolds left %d fold edges; want 0", c)
	}
	if m.TriangleCount() != 2 {
		t.Errorf("a flip must preserve the triangle count, got %d", m.TriangleCount())
	}
	// The fold's diagonal spans the crease, so its triangles over-cover; the repaired diagonal must
	// not increase total area (no overlap introduced).
	if repaired := totalTriArea(m); repaired > folded+1e-9 {
		t.Errorf("repair increased area: folded=%.4f repaired=%.4f", folded, repaired)
	}
}

func totalTriArea(m *Mesh) float64 {
	var a float64
	for t := 0; 3*t+2 < len(m.Indices); t++ {
		n := triGeomNormal(m, t)
		a += stdmath.Sqrt(float64(n.Dot(n))) / 2
	}
	return a
}

func TestRepairFoldsPreservesCleanMesh(t *testing.T) {
	t.Parallel()
	// A flat, fold-free quad: repairFolds must do nothing.
	m := &Mesh{
		Positions: []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0), math.P3(0, 1, 0)},
		Normals:   []math.Vector3{math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1)},
		Indices:   []int{0, 1, 2, 0, 2, 3},
	}
	before := append([]int(nil), m.Indices...)
	if n := repairFolds(m, 4); n != 0 {
		t.Errorf("clean mesh needed %d flips; want 0", n)
	}
	for i := range before {
		if m.Indices[i] != before[i] {
			t.Errorf("clean mesh was modified at index %d", i)
		}
	}
}
