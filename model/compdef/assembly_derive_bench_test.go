// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"testing"

	"oblikovati.org/model/benchgen"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/persistence"
)

// BenchmarkCollectPlacedBodies measures the transform-propagation flatten over the 30k
// automotive DAG (M34-B4): the recursive, single-threaded walk that composes a world
// matrix for every leaf occurrence (model/compdef/assembly_derive.go). allocs/op is the
// GC-pressure signal behind the F2 parallel/cached-traversal work — it allocates a fresh
// PlacedBody slice and per-node OccurrencePath on every call.
func BenchmarkCollectPlacedBodies(b *testing.B) {
	ws := doc.NewWorkspace(persistence.NewPackageStore())
	root, stats, err := benchgen.Generate(ws, "bench", benchgen.Auto30k())
	if err != nil {
		b.Fatal(err)
	}
	asm := root.Content().(*compdef.AssemblyComponentDefinition)
	b.Logf("benchmark fixture: %d leaf placements over %d unique meshes", stats.LeafPlacements, stats.UniqueMeshes)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = asm.PlacedBodies()
	}
}
