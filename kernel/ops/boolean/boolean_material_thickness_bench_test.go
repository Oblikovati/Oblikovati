// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"testing"

	"oblikovati.org/kernel/geom"
)

// BenchmarkSolidThickness is the cost of the size classification's measurement, against the
// bounding-box read it replaces. It is committed so the numbers in a report are reproducible from
// the tree (#3524 review I4):
//
//	go test ./kernel/ops/boolean/ -run XXX -bench SolidThickness -benchtime 1s -count 2
//
// The three arms are the same n-gon prism measured three ways. `weld` is the PRODUCTION path — the
// classification passes the pair's weld as a ceiling, so a direction that already reaches it is not
// measured to the end. `exact` is the same measurement with nothing cut short, which a caller gets by
// passing 0. `aabb` is what the measure used to be: one read of a memoized box.
//
// Measured on an AMD Ryzen AI MAX+ 395, -benchtime 1s -count 2 (the two runs bracket each cell):
//
//	faces    aabb       weld (production)   exact
//	   28    18-22 ns   10.1-11.0 us        9.7-10.1 us
//	  100    19-21 ns   31-34 us            49-50 us
//	  388    17-20 ns   106-130 us          275-318 us
//	 1004    15-16 ns   141-150 us          1.44-1.51 ms
//	 2004    13-15 ns   300-308 us          5.04-5.15 ms
//
// The exact arm is QUADRATIC — 2x the faces costs 3.4-4.7x the time — because every one of the F/2
// distinct side directions is then scanned over all F support points. The production arm is not: the
// ceiling stops each direction's scan after two or three points, so it tracks the O(F log F) of the
// direction sort. The ceiling is the whole reason the cost is acceptable, and a caller that passes 0
// on a large body pays the quadratic form.
//
// Against the bounding-box read it replaces this is three to four orders of magnitude, and it is paid
// on BOTH operands of every boolean, before classify() — so even a disjoint pair, which returns at
// once, now pays it. Three end-to-end measurements (kernel/brep A/B, a CPU profile of the NopSCADlib
// corpus, and the Inventor batch-translate suite) put the effect below the run-to-run noise of each.
func BenchmarkSolidThickness(b *testing.B) {
	for _, sides := range []int{26, 98, 386, 1002, 2002} {
		prism := ngonPrism(sides, 3, 2)
		weld := geom.ResolutionForBox(prism.RangeBox()).Weld()
		b.Run(benchName("faces", len(prism.Faces()), "aabb"), func(b *testing.B) {
			for b.Loop() {
				box := prism.RangeBox()
				d := box.Diagonal()
				_ = float64(min(min(d.X, d.Y), d.Z))
			}
		})
		b.Run(benchName("faces", len(prism.Faces()), "weld"), func(b *testing.B) {
			for b.Loop() {
				solidThickness(prism, weld)
			}
		})
		b.Run(benchName("faces", len(prism.Faces()), "exact"), func(b *testing.B) {
			for b.Loop() {
				solidThickness(prism, 0)
			}
		})
	}
}

// benchName labels one arm so the sub-benchmarks sort by size and read as a table.
func benchName(key string, n int, arm string) string {
	return key + itoa(n) + "/" + arm
}

// itoa is strconv.Itoa without the import, kept tiny so the benchmark file adds no dependency.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}
