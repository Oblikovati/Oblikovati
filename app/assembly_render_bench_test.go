// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"sync"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/benchgen"
)

// benchAssembly builds the committed 30k automotive benchmark assembly once and shares
// it across the large-assembly benchmarks (M34-B4). It is an in-memory session whose
// active document is the generated root assembly, so VisibleInstances/VisibleBodies/
// BuildBrowser run against the same scene the head renders.
var (
	benchAssemblyOnce    sync.Once
	benchAssemblySession *Session
)

func largeAssemblySession(tb testing.TB) *Session {
	benchAssemblyOnce.Do(func() {
		s := NewSession()
		root, stats, err := benchgen.Generate(s.Workspace(), "bench", benchgen.Auto30k())
		if err != nil {
			panic(err)
		}
		if err := s.Workspace().SetActiveDocument(root); err != nil {
			panic(err)
		}
		tb.Logf("benchmark fixture: %d leaf placements, %d unique meshes", stats.LeafPlacements, stats.UniqueMeshes)
		benchAssemblySession = s
	})
	return benchAssemblySession
}

// BenchmarkVisibleInstances measures the per-frame instancing assembly — flatten the
// occurrence tree and group placements by shared source body (ADR-0038). This is the
// render-side hot path; allocs/op exposes the per-frame garbage it produces.
func BenchmarkVisibleInstances(b *testing.B) {
	s := largeAssemblySession(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.VisibleInstances()
	}
}

// BenchmarkWorldAssemblyBodies measures the picking/mass-props path: one ops.TransformBody
// per occurrence into world space (the F5 bottleneck, O(total placements)). The cache is
// reset each iteration so the rebuild — not the cache hit — is timed.
func BenchmarkWorldAssemblyBodies(b *testing.B) {
	s := largeAssemblySession(b)
	asm, err := activeAssembly(s)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.asmBodies = assemblyBodyCache{} // bust the revision cache to measure the rebuild
		_ = s.worldAssemblyBodies(asm)
	}
}

// BenchmarkRayPickBodies measures the F5 fix: one pick ray through the 30k assembly answered by
// the BVH (only ray-crossed placements transformed) instead of materializing all N world bodies.
// Compare allocs/op and ns/op against BenchmarkWorldAssemblyBodies — that is the before/after of
// M34-F5. The index build is warmed out of the loop (revision cache), so the loop times the query.
func BenchmarkRayPickBodies(b *testing.B) {
	s := largeAssemblySession(b)
	asm, err := activeAssembly(s)
	if err != nil {
		b.Fatal(err)
	}
	// Aim straight down through the assembly's center so the ray actually crosses geometry —
	// a representative pick, not an empty miss. Building the index here also warms it (revision
	// cache), so the loop times only the query.
	root := s.assemblyPickIndexFor(asm).nodes[0].box
	c := root.Center()
	origin, dir := math.P3(c.X, c.Y, root.Max.Z+1000), math.V3(0, 0, -1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.RayPickBodies(origin, dir)
	}
}

// BenchmarkBuildBrowser measures the model-browser tree the UI rebuilds every frame
// (app/browser.go) — full node allocation + timeline sort, the F3 bottleneck. allocs/op
// is the per-frame cost an ImGuiListClipper / cached tree would remove.
func BenchmarkBuildBrowser(b *testing.B) {
	s := largeAssemblySession(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildBrowser(s)
	}
}
