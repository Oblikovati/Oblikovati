// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"os"
	"testing"

	"oblikovati.org/kernel/exchange/dwg"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// piracicabaData loads the real 70 MB georeferenced survey DWG from #1549. Set PIRACICABA_DWG
// to its path to run these benchmarks/guards; they skip when it is absent (the file is not
// committed). The file is the stress case for large-drawing import performance.
func piracicabaData(tb testing.TB) []byte {
	tb.Helper()
	path := os.Getenv("PIRACICABA_DWG")
	if path == "" {
		tb.Skip("set PIRACICABA_DWG to the 70MB survey file to run")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("read %q: %v", path, err)
	}
	return data
}

// piracicabaPlane builds the world XY plane (benchXYPlane takes *testing.B; this takes
// testing.TB so the benchmark and the guard test can share it).
func piracicabaPlane(tb testing.TB) sketch.Plane {
	tb.Helper()
	p, err := sketch.NewPlane(gmath.P3(0, 0, 0), gmath.V3(1, 0, 0).AsUnit(), gmath.V3(0, 1, 0).AsUnit())
	if err != nil {
		tb.Fatalf("NewPlane: %v", err)
	}
	return p
}

// BenchmarkPiracicabaDecode times only dwg.Decode (container + object map + collect + INSERT
// expand) on the real file.
func BenchmarkPiracicabaDecode(b *testing.B) {
	data := piracicabaData(b)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	var ents int
	for i := 0; i < b.N; i++ {
		dr, _, err := dwg.Decode(data)
		if err != nil {
			b.Fatal(err)
		}
		ents = len(dr.Entities)
	}
	b.ReportMetric(float64(ents), "entities")
}

// BenchmarkPiracicabaImport times the whole pipeline: decode + scale + recenter + sketch build.
func BenchmarkPiracicabaImport(b *testing.B) {
	data := piracicabaData(b)
	plane := piracicabaPlane(b)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	var ents int
	for i := 0; i < b.N; i++ {
		part := compdef.NewPartComponentDefinition()
		res, err := ImportDWG(part, data, plane)
		if err != nil {
			b.Fatal(err)
		}
		ents = res.EntityCount
	}
	b.ReportMetric(float64(ents), "sketch_ents")
}

// TestPiracicabaImportsAs2D guards the #1549 follow-up fix: this flat survey street map (94% of
// its Z samples at exactly Z=0, ~50 off-sheet strays at ±1.8e7) must import as a single 2D
// sketch, not a Sketch3D. A robustness regression in drawing.Planar — letting the outliers flip
// the classification — would route it to the 3D path: ~3.4x slower to build and the wrong
// representation. See kernel/exchange/drawing.planarInlierFraction.
func TestPiracicabaImportsAs2D(t *testing.T) {
	data := piracicabaData(t)
	part := compdef.NewPartComponentDefinition()
	res, err := ImportDWG(part, data, piracicabaPlane(t))
	if err != nil {
		t.Fatal(err)
	}
	if res.Is3D {
		t.Errorf("piracicaba.dwg imported as Sketch3D; want a single 2D sketch (%d entities)", res.EntityCount)
	}
	if got := part.Sketches().Count(); got != 1 {
		t.Errorf("2D sketch count = %d, want 1", got)
	}
}
