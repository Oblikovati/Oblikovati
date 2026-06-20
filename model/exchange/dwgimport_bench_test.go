// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"os"
	"path/filepath"
	"testing"

	gmath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// benchCorpus loads a real .dwg corpus file for a benchmark, skipping when the git-ignored
// experiments tree is absent (mirrors corpusFile, which takes *testing.T).
func benchCorpus(b *testing.B, name string) []byte {
	b.Helper()
	dir := os.Getenv("DWG_TESTFILES_DIR")
	if dir == "" {
		dir = filepath.Join("..", "..", "..", "experiments", "dwg-reverse-engineering")
	}
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		b.Skipf("corpus unavailable: %v", err)
	}
	return data
}

// benchXYPlane is the world XY plane for benchmark imports (mirrors xyPlane).
func benchXYPlane(b *testing.B) sketch.Plane {
	b.Helper()
	p, err := sketch.NewPlane(gmath.P3(0, 0, 0), gmath.V3(1, 0, 0).AsUnit(), gmath.V3(0, 1, 0).AsUnit())
	if err != nil {
		b.Fatalf("NewPlane: %v", err)
	}
	return p
}

// BenchmarkImportDWGFull times the whole import pipeline — dwg.Decode plus the sketch
// conversion (add2DEntities / add3DEntities) into a fresh part — so a regression in either
// the decoder or the converter shows up. tf-7 is planar (2D sketch), tf-2 is 3D.
func BenchmarkImportDWGFull(b *testing.B) {
	cases := []struct{ file string }{{"testfile-7.dwg"}, {"testfile-2.dwg"}, {"testfile-1.dwg"}}
	for _, tc := range cases {
		data := benchCorpus(b, tc.file)
		plane := benchXYPlane(b)
		b.Run(tc.file, func(b *testing.B) {
			b.ReportAllocs()
			var ents int
			for i := 0; i < b.N; i++ {
				part := compdef.NewPartComponentDefinition()
				res, err := ImportDWG(part, data, plane)
				if err != nil {
					b.Fatalf("ImportDWG %s: %v", tc.file, err)
				}
				ents = res.EntityCount
			}
			b.ReportMetric(float64(ents), "sketch_ents")
		})
	}
}
