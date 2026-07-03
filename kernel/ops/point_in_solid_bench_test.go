// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdrand "math/rand"
	"testing"

	"oblikovati.org/math"
)

// windingBenchQueries builds the fixed batch a winding benchmark classifies — the
// multi-vertex-against-one-mesh workload of allVerticesInside, mixing near-mesh points (which
// must run the exact loop) with far-field points (which the cached early-out certifies): a
// tool body that barely overlaps the target has most of its vertices in the far field.
func windingBenchQueries(n int) []math.Point3 {
	rng := stdrand.New(stdrand.NewSource(1607))
	pts := make([]math.Point3, n)
	for i := range pts {
		span := 1.5
		if i%2 == 0 {
			span = 40 // half the batch sits deep in the far field
		}
		pts[i] = math.P3(rng.Float64()*2*span-span, rng.Float64()*2*span-span, rng.Float64()*2*span-span)
	}
	return pts
}

// windingBenchTiers quadruples the triangle count per tier (1×/4×/16×) to expose the scaling
// of the point-in-solid query batch (#1607). The winding number is kept EXACT (see
// winding_farfield.go for why hierarchical approximation was rejected), so near-mesh queries
// stay linear in T by design; the certified far-field early-out removes the far half of the
// batch entirely.
var windingBenchTiers = []struct{ nLat, nLon int }{
	{12, 16}, // ~352 triangles
	{24, 32}, // ~1472
	{48, 64}, // ~6016
}

// BenchmarkPointInMeshBruteBatch is the baseline: 64 exact brute-loop queries per iteration.
func BenchmarkPointInMeshBruteBatch(b *testing.B) {
	for _, tier := range windingBenchTiers {
		mesh := uvSphereMesh(tier.nLat, tier.nLon, 1)
		pts := windingBenchQueries(64)
		b.Run(fmt.Sprintf("tris_%d", len(mesh.Indices)/3), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				for _, p := range pts {
					pointInMesh(mesh, p)
				}
			}
		})
	}
}

// BenchmarkPointInMeshFarFieldBatch is the accelerated path as allVerticesInside uses it:
// build the certified far-field cache, then run the same 64 queries through it.
func BenchmarkPointInMeshFarFieldBatch(b *testing.B) {
	for _, tier := range windingBenchTiers {
		mesh := uvSphereMesh(tier.nLat, tier.nLon, 1)
		pts := windingBenchQueries(64)
		b.Run(fmt.Sprintf("tris_%d", len(mesh.Indices)/3), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				inMesh := insideMeshQuerier(mesh, len(pts))
				for _, p := range pts {
					inMesh(p)
				}
			}
		})
	}
}
