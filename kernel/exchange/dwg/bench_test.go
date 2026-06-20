// SPDX-License-Identifier: GPL-2.0-only

package dwg

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// benchCorpus lists the real DWG files (largest first) used to profile decode. They live in
// the git-ignored experiments tree, so the benchmark skips when the corpus is absent.
var benchCorpus = []string{
	"testfile-1.dwg", "testfile-2.dwg", "testfile-3.dwg", "testfile-4.dwg",
	"testfile-5.dwg", "testfile-6.dwg", "testfile-7.dwg",
}

// BenchmarkDecodeCorpus times a full Decode of each corpus file (container + object map +
// geometry + insert expansion), reporting entities/op so a regression shows up as ns/entity.
func BenchmarkDecodeCorpus(b *testing.B) {
	for _, name := range benchCorpus {
		path := filepath.Join(testFilesDir(), name)
		data, err := os.ReadFile(path)
		if err != nil {
			b.Skipf("corpus %s unavailable; set %s to run", name, testFilesEnv)
			return
		}
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			var ents int
			for i := 0; i < b.N; i++ {
				dr, _, err := Decode(data)
				if err != nil {
					b.Fatalf("decode %s: %v", name, err)
				}
				ents = len(dr.Entities)
			}
			b.ReportMetric(float64(ents), "entities")
		})
	}
}

// TestReportDecodeTimings prints a one-shot decode time + entity count for every corpus file,
// a quick human-readable baseline (run with -run TestReportDecodeTimings -v).
func TestReportDecodeTimings(t *testing.T) {
	if os.Getenv("DWG_BENCH_REPORT") == "" {
		t.Skip("set DWG_BENCH_REPORT=1 to print decode timings")
	}
	for _, name := range benchCorpus {
		data := loadTestFile(t, name)
		dr, warns, err := Decode(data)
		if err != nil {
			t.Fatalf("decode %s: %v", name, err)
		}
		fmt.Printf("%-16s %8d bytes -> %7d entities, %3d warnings\n", name, len(data), len(dr.Entities), len(warns))
	}
}
