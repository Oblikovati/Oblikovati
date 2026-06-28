// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"os"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
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

// nurbsDuctSubject returns the imported-NURBS bodies the duct guards run against, the reference total
// volume (OCC getMass), and the allowed fold ceiling. By default it is the committed strongly-curved
// bulged_duct fixture, so the guard runs in CI on every machine (#1498). Setting OBK_PERF_STEP to the
// heavy developer-local EDF STEP file overrides it with that model and its EDF reference values.
func nurbsDuctSubject(t *testing.T) (bodies []*topo.Body, wantVolume float64, foldCeiling int) {
	if path := os.Getenv("OBK_PERF_STEP"); path != "" {
		return stepBodies(t, path), edfOCCVolume, edfFoldCeiling
	}
	return occBodies(t, "bulged_duct"), bulgedDuctVolume, 0
}

// TestImportedNurbsDuctWatertight guards the M25 fix that made imported NURBS faces conform to their
// neighbours: concatLoopPcurve projects each boundary loop's pcurve in ONE continuous march, so a
// self-proximal NURBS face (the EDF bell-mouth lip) no longer lands edge-starts on the wrong sheet,
// self-intersect in (u,v) and make the CDT silently drop boundary constraints — which used to crack
// the duct body (28 free edges) against its cone/cylinder/plane neighbours. It also guards the
// cross-face conformance repair (conformCylConeFaces): a trimmed cyl/cone whose plane ear-clip absorbed
// a rim-arc point is re-meshed with the metric-(u,v) CDT so it conforms to its neighbour. By default
// this runs against the committed bulged_duct fixture (CI on every run); OBK_PERF_STEP overrides to the
// heavy EDF model, where per-body free edges across the fixes went 4/0/4/28/0/0 → conformance repair
// 4/0/0/0/0/0 → planar-faithful CDT (planarTris) + exact CDT predicates → 0/0/0/0/0/0. Assert FULLY
// watertight (total 0) — it regresses hard if any of those fixes is lost.
func TestImportedNurbsDuctWatertight(t *testing.T) {
	bodies, _, _ := nurbsDuctSubject(t)
	total := 0
	for i, body := range bodies {
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
	// bulgedDuctVolume is OCC getMass (mm³) for the committed bulged_duct fixture — a strongly curved
	// B-spline barrel (radii 6→13→6) with an axial bore (test-utilities/step-oracle/generate.py). Our
	// metric-mesher tessellation lands it at ≈ -0.37%, well inside edfVolumeTol.
	bulgedDuctVolume = 6060.0844
)

// TestImportedNurbsDuctVolumeAndFolds guards the #585 fix: an imported NURBS duct's tessellation is not
// merely watertight (TestImportedNurbsDuctWatertight) but VOLUME-CORRECT and fold-free. A watertight
// mesh can still fold over itself and over-enclose — the EDF duct used to be +35.6% over OCC's getMass
// because its trimmed faces folded over a plain anisotropic (u,v) CDT. Routing them through
// metricPatchMesh (a TRIM-LOCAL metric-scaled (u,v) CDT) collapses the over-enclosure and the folds to
// zero. By default this runs against the committed strongly-curved bulged_duct fixture so the volume +
// fold-free invariants are checked in CI on every run. OBK_PERF_STEP overrides to the heavy EDF model
// (the definitive #585 over-enclosure guard, whose emergent failure does not fit a small fixture) and
// its reference volume.
func TestImportedNurbsDuctVolumeAndFolds(t *testing.T) {
	bodies, wantVolume, foldCeiling := nurbsDuctSubject(t)
	var volume float64
	folds := 0
	for _, body := range bodies {
		mesh, _ := TessellateBody(body, DefaultQuality())
		folds += FoldEdgeCount(mesh)
		volume += BodyGeometryProperties(body, DefaultQuality()).Volume
	}
	if rel := stdmath.Abs(volume-wantVolume) / wantVolume; rel > edfVolumeTol {
		t.Errorf("imported duct total volume %.4f vs OCC %.4f (rel %.4f > %.4f) — an over-enclosure regression",
			volume, wantVolume, rel, edfVolumeTol)
	}
	if folds > foldCeiling {
		t.Errorf("imported duct total fold edges = %d, want <= %d (the ceiling may only shrink as residual "+
			"folds are fixed)", folds, foldCeiling)
	}
}
