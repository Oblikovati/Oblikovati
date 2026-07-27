// SPDX-License-Identifier: GPL-2.0-only

package step_test

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// TestRadialBoreThroughCurvedWallWatertight is the regression for Oblikovati#1510: a radial through-bore
// piercing a closed (periodic) B-spline barrel wall puts one bore mouth on the surface SEAM, which made
// the planar seam-cut trim mesher drop boundary constraints and leave the wall cracked and grossly
// under-enclosed (89 free edges, ~1.4k volume vs OCC's ~5.0k). The covering-space periodic mesher must now
// tessellate it watertight (0 free edges, 0 folds) with the volume converging to the getMass oracle.
// It runs at BOTH qualities. That is not redundancy: the defect that made the covering mesh fold was
// invisible at DefaultQuality and only appeared at PropertyQuality — see
// TestRadialBoreWallIsAManifoldDiscAtPropertyQuality.
func TestRadialBoreThroughCurvedWallWatertight(t *testing.T) {
	const occMass = 5023.5696
	b := importCandRadial(t)
	for _, c := range []struct {
		name string
		q    ops.Quality
	}{{"default", ops.DefaultQuality()}, {"property", ops.PropertyQuality()}} {
		mesh, _ := ops.TessellateBody(b, c.q)
		if free := openEdgeCount(mesh); free != 0 {
			t.Errorf("%s quality: tessellation has %d free edges; want 0 (non-watertight curved wall)", c.name, free)
		}
		if folds := ops.FoldEdgeCount(mesh); folds != 0 {
			t.Errorf("%s quality: tessellation has %d fold edges; want 0", c.name, folds)
		}
		vol := ops.BodyGeometryProperties(b, c.q).Volume
		if rel := (vol - occMass) / occMass; rel < -0.03 || rel > 0.03 {
			t.Errorf("%s quality: volume %.4f vs OCC %.4f (rel %.5f); want within 3%%", c.name, vol, occMass, rel)
		}
	}
}

// TestRadialBoreWallIsAManifoldDiscAtPropertyQuality gates the covering-space periodic mesher on the
// ONE face in the tree that actually reaches it — this barrel wall — at the quality that exposed the
// bug. `ParamAt` used to invert every point of the surface's LAST knot span to u = ulo (a full period
// off its own foot), collapsing FOUR distinct 256-point rim samples onto a single chart node; the
// zero-length constraint segments that produced folded the CDT. Measured before → after:
// folds 3 → 0, unpaired boundary edges 984 → 992, face area 1403.712529 → 1403.409482, and the whole
// body 12 free edges → 0.
//
// The invariant asserted is EXACT, with target zero, not a captured number: a trimmed face meshes to
// a single-cover manifold patch, so it has no fold edges at all. The receipt that the RED mesh was
// over-covering rather than merely different: it carried FEWER triangles (8466 vs 8476) while
// measuring MORE area (1403.712529 vs 1403.409482) over the same trim — the surplus is overlap.
func TestRadialBoreWallIsAManifoldDiscAtPropertyQuality(t *testing.T) {
	b := importCandRadial(t)
	wall := 0
	for _, f := range b.Faces() {
		if _, isSpline := f.Geometry().(geom.BSplineSurface); !isSpline {
			continue
		}
		wall++
		m := ops.TessellateFace(f, ops.PropertyQuality())
		if m == nil {
			t.Fatalf("barrel wall face %d did not tessellate", f.ID())
		}
		if folds := ops.FoldEdgeCount(m); folds != 0 {
			t.Errorf("barrel wall has %d fold edges; want 0 (the covering mesh folded over its own seam)", folds)
		}
	}
	if wall != 1 {
		t.Fatalf("cand_radial has %d B-spline faces; the fixture is meant to carry exactly 1", wall)
	}
}

// importCandRadial loads the #1510 fixture as its single solid.
func importCandRadial(t *testing.T) *topo.Body {
	t.Helper()
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
	return bodies[0]
}

// openEdgeCount welds coincident vertices, then counts mesh edges not shared by exactly two triangles —
// the watertightness metric (0 = closed manifold). It welds because TessellateBody copies shared-edge
// vertices per face (mergeMesh offsets indices), mirroring kernel/ops' own freeEdgeCount helper.
func openEdgeCount(m *ops.Mesh) int {
	weld := weldIndices(m)
	deg := map[[2]int]int{}
	for t := 0; 3*t+2 < len(m.Indices); t++ {
		v := [3]int{weld[m.Indices[3*t]], weld[m.Indices[3*t+1]], weld[m.Indices[3*t+2]]}
		for k := 0; k < 3; k++ {
			deg[orderedPair(v[k], v[(k+1)%3])]++
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

// orderedPair canonicalises a mesh edge's endpoint pair so the two triangles that share it hash alike.
func orderedPair(a, b int) [2]int {
	if a > b {
		a, b = b, a
	}
	return [2]int{a, b}
}

// weldIndices maps each mesh vertex to a canonical index for coincident 3D positions.
func weldIndices(m *ops.Mesh) []int {
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
	return weld
}
