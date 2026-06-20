// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"os"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/math"
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
// Per-body free edges across the fixes: pcurve continuity 4/0/4/28/0/0 → conformance repair
// 4/0/0/0/0/0 (total 4) → the planar-faithful CDT fallback (planarTris) + exact CDT predicates closed
// body0's plane-side crack to 0/0/0/0/0/0. The plane crack was an earcut hole-bridging defect on the
// top cap (a complex outer loop + two circular bores); detecting it by area mismatch and re-meshing
// with the now-cocircular-robust constrainedDelaunay makes it watertight. Assert the model is now FULLY
// watertight (total 0) — it regresses hard (to 4, 8, then 36) if any of those fixes is lost.
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
		if free != 0 {
			t.Errorf("imported body %d tessellated with %d free edges; want 0 "+
				"(a pcurve-continuity, conformance, or planar-faithful regression cracks a body)", i, free)
		}
	}
	if total != 0 {
		t.Errorf("imported model total free edges = %d; want 0 (watertightness regression)", total)
	}
}

// edfOCCVolume is OCC getMass for the EDF model (mm³); edfFoldCeiling is the residual fold count,
// which may only shrink. It is now 0: fold-driven interior refinement clears the B-spline lip folds
// (#585) and the sphere pole caps fall back to the fold-free boundary triangulation. See edfVolumeTol.
const (
	edfOCCVolume   = 207002.0
	edfVolumeTol   = 0.01 // 1% — the trim-local metric mesher lands EDF at ≈ -0.4% (was +35.6%)
	edfFoldCeiling = 0
)

// TestImportedNurbsDuctVolumeAndFolds guards the #585 fix: EDF's tessellation is not merely watertight
// (TestImportedNurbsDuctWatertight) but VOLUME-CORRECT and (almost) fold-free. A watertight mesh can
// still fold over itself and over-enclose — the EDF duct used to be +35.6% over OCC's getMass because
// its trimmed analytic faces (torus/cylinder/sphere/cone) folded. Routing them through metricPatchMesh
// (a TRIM-LOCAL metric-scaled (u,v) CDT) collapses the over-enclosure to ≈ -0.4% and the fold count
// from 139 to the residual ceiling (the remaining folds are the B-spline pcurve path + a few sphere
// pole facets, tracked separately). Env-gated like the watertightness guard.
func TestImportedNurbsDuctVolumeAndFolds(t *testing.T) {
	path := os.Getenv("OBK_PERF_STEP")
	if path == "" {
		t.Skip("set OBK_PERF_STEP=/path/EDF.STEP to run the imported-NURBS volume/fold guard")
	}
	var volume float64
	folds := 0
	for _, body := range stepBodies(t, path) {
		mesh, _ := TessellateBody(body, DefaultQuality())
		folds += FoldEdgeCount(mesh)
		volume += BodyGeometryProperties(body, DefaultQuality()).Volume
	}
	if rel := stdmath.Abs(volume-edfOCCVolume) / edfOCCVolume; rel > edfVolumeTol {
		t.Errorf("EDF total volume %.0f vs OCC %.0f (rel %.4f > %.4f) — an over-enclosure regression",
			volume, edfOCCVolume, rel, edfVolumeTol)
	}
	if folds > edfFoldCeiling {
		t.Errorf("EDF total fold edges = %d, want <= %d (the ceiling may only shrink as the residual "+
			"B-spline/sphere folds are fixed)", folds, edfFoldCeiling)
	}
}
