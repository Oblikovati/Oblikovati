// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"

	"oblikovati.org/kernel/mesh"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
)

// Test-only window onto the chart-driven mesher for the external corpus rows. It lives in a _test.go
// file, so it is compiled into the package's tests and never ships.

// ChartCoveringLocationCount is the covering's vertex count for one face and how many of those
// vertices share a LOCATION with an earlier one — the number that must be zero, because a constrained
// triangulation cannot recover a constraint incident to a vertex another vertex sits on (#3551).
//
// It counts what the triangulation would actually see: the covering as newChartCover lays it, boundary
// chains and interior nodes together. ok=false for a face this mesher never owned.
//
// Example: dup, n, ok := ChartCoveringLocationCount(f, PropertyQuality())
func ChartCoveringLocationCount(f *topo.Face, q Quality) (duplicates, vertices int, ok bool) {
	s := f.Geometry()
	r, isChart := newChartRegion(f, s)
	if !isChart {
		return 0, 0, false
	}
	chains := chartBoundaryChains(f, s, r, q)
	b := newChartCover(s, r, q)
	b.addChains(chains)
	b.addInterior(chains)
	seen := map[[2]int64]int{}
	for i := range b.xy {
		k := [2]int64{mesh.Quantize(b.xy[i][0], b.weld), mesh.Quantize(b.xy[i][1], b.weld)}
		if j, dup := seen[k]; dup && stdmath.Hypot(b.xy[i][0]-b.xy[j][0], b.xy[i][1]-b.xy[j][1]) <= b.weld {
			duplicates++
		}
		seen[k] = i
	}
	return duplicates, len(b.xy), true
}

// ChartFaceRimMismatch drives the chart-driven mesher on one face and reports the GATE's own two
// numbers: how many of the mesh's unpaired edges are no rim segment, and how many rim segments the
// mesh does not bound. It reports them EVEN WHEN THE GATE REFUSES, which chartFaceMesh cannot: that
// call answers only "declined", and a corpus row has to say by how much and in which direction.
//
// meshed is false only when the mesher never built a mesh at all — the face records no chart, or the
// covering kept nothing — and declined then says why. It reads the numbers through chartRimMismatch,
// the one place a rim is keyed, so a row asserts what the mesher itself decides on (#3520).
//
// Example: extra, missing, ok, why := ChartFaceRimMismatch(f, PropertyQuality())
func ChartFaceRimMismatch(f *topo.Face, q Quality) (extra, missing int, meshed bool, declined string) {
	s := f.Geometry()
	r, ok := newChartRegion(f, s)
	if !ok {
		return 0, 0, false, "the face records no chart"
	}
	chains := chartBoundaryChains(f, s, r, q)
	b := newChartCover(s, r, q)
	loops := b.addChains(chains)
	b.addInterior(chains)
	kept, split := b.keptWithoutRimEars(loops)
	if !split || len(kept) == 0 {
		return 0, 0, false, "the covering kept no triangle it could ship"
	}
	pos, nrm, idx := weldCoverTriangles(b.pos, b.nrm, kept)
	m := patchMeshFrom(pos, nrm, idx)
	validate.RepairFolds(m, 8)
	e, mi := chartRimMismatch(m, chains, b.weld)
	return e, mi, true, ""
}
