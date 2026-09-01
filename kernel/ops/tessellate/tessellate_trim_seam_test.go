// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"
)

// The seam invariant, in one sentence: a periodic loop's development is faithful only if the loop's
// TOTAL winding about the periodic axis — closing step included — is zero.
//
// The defect these tests pin (retrace-detector-report.md §7.1). Unwrap used to measure the span of the
// OPEN chain only, so a loop that STARTS on the seam passed by ε and then leapt a whole period on the
// step nothing looked at: samples 0..n−1 back to sample 0. simple/W1's corner sphere reads 6.2586 rad
// across its open chain — 0.024 short of 2π — and its closing step jumps 2π. Unwrap is production
// MESHER code (tessellateCurvedFace, conformingCylConeMesh both gate on it), so the only thing that
// kept a mis-developed loop off a triangulator was that spherePatchMesh happens to intercept every
// member of the measured population first. That is luck, not a guarantee.
//
// Both DIRECTIONS are pinned here, because the cheap fix (reject anything whose span is large) would
// be wrong: a loop may legitimately reach far around the period, run along the seam, or straddle it,
// and must still develop. Only the spurious full-period jump is a defect.

// seamStartFullWrapSamples is a loop that starts ON the seam and sweeps the whole period: n samples of
// u at k·2π/n. Its open chain spans (n−1)/n · 2π — always under the old guard's 2π − 1e-6 — and its
// closing step then leaps back to 0.
func seamStartFullWrapSamples(n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = 2 * stdmath.Pi * float64(i) / float64(n)
	}
	return out
}

// TestUnwrapRejectsSeamStartFullWrap is the RED half of the guard: a loop that encircles the periodic
// axis has no simple polygon in this chart, and Unwrap must say so however it is phased. Restoring the
// open-chain-only span guard turns this test red at every n.
func TestUnwrapRejectsSeamStartFullWrap(t *testing.T) {
	t.Parallel()
	for _, n := range []int{4, 24, 192, 320} {
		if _, ok := Unwrap(seamStartFullWrapSamples(n)); ok {
			t.Errorf("n=%d: a loop starting on the seam and winding a full period developed as a polygon "+
				"(open-chain span %.6g, closing step leaps 2π)", n, 2*stdmath.Pi*float64(n-1)/float64(n))
		}
	}
}

// TestUnwrapRejectsFullWrapWhereverItStarts checks the guard is phase-independent: the SAME ring
// rotated to start anywhere must be rejected identically. (The old guard rejected it only when the
// rotation happened to put its span over 2π.)
func TestUnwrapRejectsFullWrapWhereverItStarts(t *testing.T) {
	t.Parallel()
	const n = 24
	base := seamStartFullWrapSamples(n)
	for shift := range n {
		rot := make([]float64, n)
		for i := range rot {
			rot[i] = base[(i+shift)%n]
		}
		if _, ok := Unwrap(rot); ok {
			t.Errorf("shift=%d: the same full-period ring developed as a polygon", shift)
		}
	}
}

// TestUnwrapAcceptsWideOutAndBack is the first FALSE-POSITIVE direction: a loop that reaches almost
// all the way around and comes back covers a huge span but winds ZERO, so it develops. It is exactly
// the shape of a wide cylinder-wall trim, and rejecting it would drop that face onto the seam-crossing
// fallback for no reason.
func TestUnwrapAcceptsWideOutAndBack(t *testing.T) {
	t.Parallel()
	var a []float64
	const top = 6.0 // rad: 95.5% of the period, out and back
	for u := 0.0; u < top; u += 0.05 {
		a = append(a, u)
	}
	for u := top; u > 0; u -= 0.05 {
		a = append(a, u)
	}
	out, ok := Unwrap(a)
	if !ok {
		t.Fatalf("a zero-winding loop spanning %.6g rad was refused a development", top)
	}
	lo, hi := minMax(out)
	if hi-lo < top-1e-9 {
		t.Errorf("developed span %.6g collapsed below the loop's own %.6g", hi-lo, top)
	}
}

// TestUnwrapAcceptsSeamStraddlingLoop is the second FALSE-POSITIVE direction: a loop that CROSSES the
// seam transversally reads its samples on both sides of 0 (…, 2π−ε, ε, …) yet winds zero. It must
// develop continuously — the whole point of unwrapping — and the development must stay contiguous
// across the seam rather than tearing.
func TestUnwrapAcceptsSeamStraddlingLoop(t *testing.T) {
	t.Parallel()
	twoPi := 2 * stdmath.Pi
	a := []float64{twoPi - 0.2, twoPi - 0.1, 0.1, 0.2, 0.2, 0.1, twoPi - 0.1, twoPi - 0.2}
	out, ok := Unwrap(a)
	if !ok {
		t.Fatal("a loop straddling the seam with zero winding was refused a development")
	}
	lo, hi := minMax(out)
	if got := hi - lo; stdmath.Abs(got-0.4) > 1e-12 {
		t.Errorf("developed span %.12g, want 0.4 — the seam crossing tore instead of unwrapping", got)
	}
}

// TestUnwrapAcceptsSeamTangentialLoop is the third FALSE-POSITIVE direction, and the one a sloppy fix
// breaks: a loop that RUNS ALONG the seam inverts to u samples that flip between ~0 and ~2π on sign
// noise alone. Those flips are ±ε steps, not period leaps, and must not accumulate into a winding.
func TestUnwrapAcceptsSeamTangentialLoop(t *testing.T) {
	t.Parallel()
	twoPi := 2 * stdmath.Pi
	const eps = 1e-13
	a := []float64{0, twoPi - eps, 0, eps, twoPi - eps, 0, eps, twoPi - eps}
	out, ok := Unwrap(a)
	if !ok {
		t.Fatal("a loop running along the seam was refused a development (float sign noise read as winding)")
	}
	lo, hi := minMax(out)
	if hi-lo > 1e-9 {
		t.Errorf("developed span %.6g on a loop that never leaves the seam", hi-lo)
	}
}

// TestUnwrapAcceptsDegenerateAndTinyLoops guards the boundary cases the closing step is undefined or
// trivial on: a single sample, a two-sample loop, and an all-identical ring.
func TestUnwrapAcceptsDegenerateAndTinyLoops(t *testing.T) {
	t.Parallel()
	for _, a := range [][]float64{{1.0}, {1.0, 1.1}, {0.3, 0.3, 0.3, 0.3}} {
		if _, ok := Unwrap(a); !ok {
			t.Errorf("%v: a degenerate periodic sample set was refused a development", a)
		}
	}
}

// TestSeamWindingLeapMeasuresTheWinding pins the predicate's QUANTITY, not just its verdict: the
// residual it thresholds is the loop's total winding about the axis, which is 0 or ±2πk and nothing
// in between. A loop wound TWICE must be rejected just as a loop wound once is.
func TestSeamWindingLeapMeasuresTheWinding(t *testing.T) {
	t.Parallel()
	var twice []float64
	const n = 48
	for i := range n {
		twice = append(twice, stdmath.Mod(4*stdmath.Pi*float64(i)/float64(n), 2*stdmath.Pi))
	}
	if _, ok := Unwrap(twice); ok {
		t.Error("a loop winding the period TWICE developed as a polygon")
	}
}
