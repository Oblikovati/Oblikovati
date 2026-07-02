// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"testing"

	"oblikovati.org/model/benchgen"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/persistence"
)

// BenchmarkCollectPlacedBodies measures the transform-propagation flatten over the 30k
// automotive DAG (model/compdef/assembly_derive.go). After M34-F2 it walks with a reused DFS
// path stack (one path allocation per emitted leaf, ~half the pre-F2 allocs) and fans the
// top-level subtrees across workers (~1.8× faster wall time at 30k). allocs/op is the
// GC-pressure signal the trim targets.
func BenchmarkCollectPlacedBodies(b *testing.B) {
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
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
