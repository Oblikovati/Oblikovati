// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"sync"
	"testing"

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
