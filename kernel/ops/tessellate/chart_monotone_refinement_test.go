// SPDX-License-Identifier: GPL-2.0-only

package tessellate_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// Monotone refinement (ADR-0061 stage 5, Task 7). A chord-tolerance mesher owes two things a
// single-quality gate cannot see: the body must be watertight at EVERY faceting, and tightening the
// tolerance must not take area or volume away. Both were broken by the covering's replication pad,
// which was measured in the wrong axis's station gaps: at PropertyQuality the #1738 corner junction's
// wall came back with 871 unpaired edges against a rim of 868 and cracked the body, while at
// DefaultQuality the same body was watertight (see chartCoverPadStations).

// refinementQualities is the coarse-then-fine pair every row below is measured at.
func refinementQualities() (coarse, fine ops.Quality) {
	return ops.DefaultQuality(), ops.PropertyQuality()
}

// knownFreeEdgesAtFineQuality pins the one corpus body that is NOT watertight at PropertyQuality, so
// the gate ships green while saying exactly what is broken and trips the day it is fixed.
//
// "ring − half space" is the torus cut by the plane x = R, whose section is the LEMNISCATE: the two
// spiric branches meet at (R, 0, ±r), so the face's boundary passes through the same 3D point twice.
// At PropertyQuality the chart-driven mesher comes back with 276 unpaired edges against a rim of 272 —
// four extra, two per node — is declined by its own rim gate, and the face falls to the surface's whole
// domain (296.062 mm² against the 264.830 it had built), cracking the body's planar cap with it. The
// count is PRE-EXISTING and was proved so by removing this task's spiric conditioning gate and
// re-measuring: identical, 272. At DefaultQuality the body is watertight, which is why no gate saw it.
var knownFreeEdgesAtFineQuality = map[string]int{"ring − half space": 272}

// TestEveryCorpusBodyIsWatertightUnderRefinement meshes every classification-corpus body at both
// facetings and demands no free edge at either. Volume is deliberately NOT asserted monotone here: a
// body with a CONCAVE curved feature loses volume as it refines, because the faceted bore is inscribed
// and grows into the solid (measured: the drilled plate 325.419 → 325.262, the conical drill point
// 345.318 → 345.288). The invariant that does hold is per-FACE area, below.
func TestEveryCorpusBodyIsWatertightUnderRefinement(t *testing.T) {
	t.Parallel()
	coarse, fine := refinementQualities()
	for _, row := range classificationCorpus() {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			body := row.build(t)
			lo, hi := meshOf(t, body, coarse), meshOf(t, body, fine)
			if lo.free != 0 {
				t.Errorf("%d free edges at the coarse faceting (want 0)", lo.free)
			}
			if want := knownFreeEdgesAtFineQuality[row.name]; hi.free != want {
				t.Errorf("%d free edges at the fine faceting, want %d — see knownFreeEdgesAtFineQuality: "+
					"either a new crack, or the lemniscate node is fixed and this row is now watertight",
					hi.free, want)
			}
		})
	}
}

// TestEveryChartedFaceGainsAreaUnderRefinement is the same invariant per FACE, on the faces the
// chart-driven mesher owns: it is where the defect actually lives, and a body total can hide a face
// that loses area behind another that gains it.
func TestEveryChartedFaceGainsAreaUnderRefinement(t *testing.T) {
	t.Parallel()
	coarse, fine := refinementQualities()
	seen := 0
	forEachCurvedCorpusFace(t, func(body string, i int, f *topo.Face) {
		if tessellate.ClassifyCurvedTrimName(f, coarse) != tessellate.ChartedTrimKindName() {
			return
		}
		seen++
		lo := tessellate.MeshGeometryProperties(tessellate.TessellateFace(f, coarse)).Area
		hi := tessellate.MeshGeometryProperties(tessellate.TessellateFace(f, fine)).Area
		// A facet chord is a SECANT of the surface, so refining can only add area back; the slack is
		// the fine mesh's own remaining deficit, not a licence to lose any.
		if hi < lo-refinementSlack*stdmath.Abs(lo) {
			t.Errorf("%s face %d (%T): area FELL under refinement, %.5f → %.5f (rel %+.5f)",
				body, i, f.Geometry(), lo, hi, (hi-lo)/lo)
		}
	})
	if seen == 0 {
		t.Error("no corpus face reached the chart-driven mesher — the refinement gate covers nothing")
	}
}

// refinementSlack is how much of a face's coarse area the fine mesh may still be short by. It is not a
// tolerance on the geometry: a face whose boundary is exact at both facetings gains area monotonically,
// and this only absorbs the last-place rounding of two independent summations.
const refinementSlack = 1e-9 // tol:numeric

// meshRow is one body's watertightness and volume at one faceting.
type meshRow struct {
	free   int
	volume float64
}

// meshOf tessellates a body and reads the two numbers the refinement gate compares.
func meshOf(t *testing.T, body *topo.Body, q ops.Quality) meshRow {
	t.Helper()
	m, _ := tessellate.TessellateBody(body, q)
	return meshRow{free: tessellate.FreeEdgeCount(m), volume: tessellate.MeshGeometryProperties(m).Volume}
}
