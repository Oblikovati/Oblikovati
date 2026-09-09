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
//	faces    aabb          weld (production)   exact
//	   28    11.5-11.6 ns  2.12-2.15 us        2.75-2.77 us
//	  100    11.5 ns       7.85-8.18 us        20.4 us
//	  388    11.4 ns       35.6-36.0 us        194-195 us
//	 1004    11.3-11.4 ns  101-102 us          1.20-1.21 ms
//	 2004    11.3 ns       203-211 us          4.21-4.24 ms
//
// The exact arm is QUADRATIC — 2x the faces costs 3.5-6.2x the time — because every one of the F/2
// distinct side directions is then scanned over all F support points. The production arm is not: the
// ceiling stops each direction's scan after two or three points, so it tracks the O(F log F) of the
// direction sort. The ceiling is the whole reason the cost is acceptable, and a caller that passes 0
// on a large body pays the quadratic form.
//
// The unconditional principal frame (one scatter matrix and one 3x3 Jacobi over the support, plus
// three more extents) is inside these numbers: the whole package measured 61.83 s wall / 209.78 s CPU
// on the wave base and 60.40 s / 210.55 s with it, run one at a time on the same box.
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
