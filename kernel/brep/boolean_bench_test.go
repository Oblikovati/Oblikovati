// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"fmt"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
)

// BenchmarkBooleanPatternedPocketChain locks the boolean pipeline's asymptotics on the
// pattern-blowup shape the audit flagged (#1607): a plate with an n×n pocket grid cut one
// boolean at a time, so the target's face count grows with every cut (6+5k faces after k
// pockets) exactly like a patterned-feature recompute. Tiers quadruple the pocket count
// (1×/4×/16×); with the AABB face-pair culling the cost per tier must grow sub-quadratically
// in the total face count where the brute O(Fa·Fb) pairing grew quadratically.
func BenchmarkBooleanPatternedPocketChain(b *testing.B) {
	for _, n := range []int{2, 4, 8} {
		b.Run(fmt.Sprintf("pockets_%dx%d", n, n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				pocketChain(b, n)
			}
		})
	}
}

// pocketChain cuts an n×n grid of blind pockets into a 4n×4n×2 plate, one Difference at a
// time, and returns the final body (face count 6+5n²).
func pocketChain(tb testing.TB, n int) *topo.Body {
	tb.Helper()
	res := box(0, 0, 0, float64(4*n), float64(4*n), 2)
	for k := 0; k < n*n; k++ {
		tool := box(float64(4*(k/n))+1, float64(4*(k%n))+1, 1, 2, 2, 2)
		out, err := brep.Boolean(brep.Difference, res, tool)
		if err != nil {
			tb.Fatalf("pocket %d/%d: %v", k, n*n, err)
		}
		res = out
	}
	return res
}

// TestPocketChainFixtureIsSound guards the benchmark fixture itself: the chained cuts must
// yield a valid solid with the analytic volume (plate minus n² pockets of 2×2×1), so the
// benchmark keeps measuring real boolean work rather than an error path.
func TestPocketChainFixtureIsSound(t *testing.T) {
	t.Parallel()
	n := 3
	res := pocketChain(t, n)
	if !res.IsSolid() {
		t.Fatal("pocket-chain result is not a solid")
	}
	want := float64(4*n*4*n*2 - n*n*4)
	if got := vol(res); stdAbs(got-want) > 1e-6*want {
		t.Fatalf("pocket-chain volume = %g, want %g", got, want)
	}
}

func stdAbs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
