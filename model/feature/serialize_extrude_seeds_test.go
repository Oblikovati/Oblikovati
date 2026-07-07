// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// A rectangle split by a vertical line into a small left region (area 2) and a large
// right region (area 6). Region *ordering* is a DCEL artifact, so an external author
// must select by an interior seed point, not an index.
func splitRectSketch() *sketch.Sketch {
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	l := sk.Lines()
	l.AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	l.AddByTwoPoints(math.P2(4, 0), math.P2(4, 2))
	l.AddByTwoPoints(math.P2(4, 2), math.P2(0, 2))
	l.AddByTwoPoints(math.P2(0, 2), math.P2(0, 0))
	l.AddByTwoPoints(math.P2(1, 0), math.P2(1, 2)) // split at x=1
	return sk
}

func TestResolveSeedsSelectsRegionByContainment(t *testing.T) {
	sk := splitRectSketch()

	// A seed in the large right region resolves to whichever index holds area 6.
	got := resolveSeeds(sk, [][]float64{{2.5, 1}}, []int{0})
	if len(got) != 1 {
		t.Fatalf("want one resolved index, got %v", got)
	}
	if a := sk.Profiles().Item(got[0]).Area(); stdmath.Abs(a-6) > 1e-6 {
		t.Errorf("seed (2.5,1) resolved to a region of area %v, want the area-6 region", a)
	}

	// A seed in the small left region resolves to the area-2 region.
	left := resolveSeeds(sk, [][]float64{{0.5, 1}}, []int{0})
	if a := sk.Profiles().Item(left[0]).Area(); stdmath.Abs(a-2) > 1e-6 {
		t.Errorf("seed (0.5,1) resolved to a region of area %v, want the area-2 region", a)
	}
}

func TestResolveSeedsFallsBack(t *testing.T) {
	sk := splitRectSketch()

	// No seeds -> the explicit index list is used unchanged.
	if got := resolveSeeds(sk, nil, []int{3}); len(got) != 1 || got[0] != 3 {
		t.Errorf("no seeds: want fallback [3], got %v", got)
	}
	// A seed that lands in no region -> fall back (never an empty, whole-body selection).
	if got := resolveSeeds(sk, [][]float64{{99, 99}}, []int{5}); len(got) != 1 || got[0] != 5 {
		t.Errorf("unmatched seed: want fallback [5], got %v", got)
	}
}

func TestResolveSeedSingleRegion(t *testing.T) {
	sk := splitRectSketch()
	// a seed in the large region resolves to whichever index holds area 6
	got := resolveSeed(sk, []float64{2.5, 1}, 0)
	if a := sk.Profiles().Item(got).Area(); stdmath.Abs(a-6) > 1e-6 {
		t.Errorf("seed (2.5,1) resolved to a region of area %v, want the area-6 region", a)
	}
	// no seed / unmatched seed -> the fallback index
	if got := resolveSeed(sk, nil, 3); got != 3 {
		t.Errorf("no seed: want fallback 3, got %d", got)
	}
	if got := resolveSeed(sk, []float64{99, 99}, 5); got != 5 {
		t.Errorf("unmatched seed: want fallback 5, got %d", got)
	}
}
