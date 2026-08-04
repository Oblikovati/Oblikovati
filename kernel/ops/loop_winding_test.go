// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestLoopWindingIsExactlyQuantised pins the property the rim/lens classifier now rests on: a closed
// loop's winding is 2π·k EXACTLY, never a value in between, because the raw steps telescope to zero and
// each fold contributes a whole period. The target is exact (residual 0), not a captured number. It is
// swept over every start phase, since starting a rim ON the seam is exactly what broke the sibling
// classifier in tessellate_trim.go (TestUnwrapRejectsFullWrapWhereverItStarts).
func TestLoopWindingIsExactlyQuantised(t *testing.T) {
	for _, n := range []int{3, 5, 8, 17, 64} {
		for phase := 0; phase < n; phase++ {
			us := make([]float64, n)
			for i := range us {
				us[i] = stdmath.Mod(2*stdmath.Pi*float64((i+phase)%n)/float64(n), 2*stdmath.Pi)
			}
			w := loopWinding(us)
			k := stdmath.Round(w / (2 * stdmath.Pi))
			if res := stdmath.Abs(w - k*2*stdmath.Pi); res > 1e-12 {
				t.Fatalf("n=%d phase=%d: winding %.15f is %.3e off the nearest whole turn %.0f; want exactly quantised",
					n, phase, w, res, k)
			}
			if k != 1 {
				t.Fatalf("n=%d phase=%d: a rim sampled once round wound %.0f turns; want exactly 1", n, phase, k)
			}
		}
	}
}

// TestLoopWindingIsZeroForAContractibleLoop pins the other half: a loop that goes out and comes back —
// the shape of a lens hole, and of the seam-bridged wall rectangle wrappedWallUV builds — has winding
// exactly 0 however wide its angular excursion is. This is why wrappedWallUV keeps the u-RANGE test:
// its loop's winding is 0 by construction, so a winding test would be the wrong question there.
func TestLoopWindingIsZeroForAContractibleLoop(t *testing.T) {
	for _, span := range []float64{0.2, 1.5, 3.0, 6.0} { // up to 6 rad — nearly the whole period
		us := []float64{0, span / 3, 2 * span / 3, span, 2 * span / 3, span / 3}
		if w := loopWinding(us); stdmath.Abs(w) > 1e-12 {
			t.Fatalf("out-and-back loop of span %.1f wound %.3e turns; want exactly 0", span, w/(2*stdmath.Pi))
		}
	}
}

// TestHoleWrapsPeriodSeesACoarselySampledRim is the regression for the classifier this replaced. A rim
// sampled at 8 points leaves a 2π/8 = 0.785 rad closing gap, so its cumulative u-RANGE is only 5.498 —
// BELOW the old 2π − 0.5 = 5.783 threshold, which therefore called an axis-encircling rim a lens. That
// misclassification drops a two-rim band out of twoRimHoledBandMesh (len(rims) != 1) and onto the flat
// best-fit-plane CDT, which mangles a cylinder wall (#1724: inside-out band, ~140 free edges).
//
// The old expression is recomputed here rather than described, so this test states the defect it guards
// against instead of merely asserting the new answer.
func TestHoleWrapsPeriodSeesACoarselySampledRim(t *testing.T) {
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 5)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	const n = 8
	rim := make([]math.Point3, n)
	us := make([]float64, n)
	for i := range rim {
		th := 2 * stdmath.Pi * float64(i) / n
		rim[i] = math.P3(math.Scalar(5*stdmath.Cos(th)), math.Scalar(5*stdmath.Sin(th)), 3)
		us[i], _ = cyl.ParamAt(rim[i])
	}
	lo, hi := minMax(cumulativeUnwrap(us))
	if hi-lo > 2*stdmath.Pi-0.5 {
		t.Fatalf("fixture is not coarse enough: u-range %.6f already clears the old 2π−0.5 threshold %.6f",
			hi-lo, 2*stdmath.Pi-0.5)
	}
	if !holeWrapsPeriod(cyl, rim) {
		t.Errorf("an 8-point rim (u-range %.4f, winding %.4f turns) was classified as a LENS; want a RIM — "+
			"the verdict must follow the loop's topology, not how finely it happens to be faceted",
			hi-lo, loopWinding(us)/(2*stdmath.Pi))
	}
}

// TestHoleWrapsPeriodStillRejectsAWideLens is the false-positive direction: a contractible lens spanning
// 340° of the wall must stay a LENS. Its u-range (5.93) CLEARS the old 2π − 0.5 threshold, so the range
// classifier would have promoted it to a rim and sent a well-formed drilled wall down the two-rim path.
func TestHoleWrapsPeriodStillRejectsAWideLens(t *testing.T) {
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 5)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	const span = 340 * stdmath.Pi / 180
	var lens []math.Point3
	at := func(th, z float64) math.Point3 {
		return math.P3(math.Scalar(5*stdmath.Cos(th)), math.Scalar(5*stdmath.Sin(th)), math.Scalar(z))
	}
	for i := 0; i <= 40; i++ { // out along z=2 …
		lens = append(lens, at(span*float64(i)/40, 2))
	}
	for i := 39; i >= 1; i-- { // … and back along z=4
		lens = append(lens, at(span*float64(i)/40, 4))
	}
	us := make([]float64, len(lens))
	for i, p := range lens {
		us[i], _ = cyl.ParamAt(p)
	}
	if lo, hi := minMax(cumulativeUnwrap(us)); hi-lo <= 2*stdmath.Pi-0.5 {
		t.Fatalf("fixture is not wide enough: u-range %.6f does not reach the old threshold %.6f",
			hi-lo, 2*stdmath.Pi-0.5)
	}
	if holeWrapsPeriod(cyl, lens) {
		t.Errorf("a 340° contractible lens (winding %.4f turns) was classified as a RIM; want a LENS",
			loopWinding(us)/(2*stdmath.Pi))
	}
}
