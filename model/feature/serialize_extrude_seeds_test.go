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

// regionIndexOfArea returns the profile index whose area matches want (within tol).
func regionIndexOfArea(sk *sketch.Sketch, want float64) int {
	ps := sk.Profiles()
	for i := 0; i < ps.Count(); i++ {
		if stdmath.Abs(ps.Item(i).Area()-want) < 1e-6 {
			return i
		}
	}
	return -1
}

// TestExtrudeSeedResolvesAtRecomputeNotStaleIndex is the #region-seed regression: a feature
// that carries a seed must re-resolve its region from the seed at recompute, so a stale
// ProfileIndices (as if the sketch was re-solved and the DCEL regions reordered after load)
// does NOT strand the extrude on the wrong cell. Here the index deliberately points at the
// large region while the seed points at the small one — the seed must win.
func TestExtrudeSeedResolvesAtRecomputeNotStaleIndex(t *testing.T) {
	sk := splitRectSketch()
	large := regionIndexOfArea(sk, 6) // the WRONG cell the stale index points at
	if large < 0 {
		t.Fatal("setup: no area-6 region")
	}

	e := &ExtrudeFeature{def: &ExtrudeDefinition{
		Sketch:         sk,
		ProfileIndices: []int{large},          // stale index → the area-6 region
		ProfileSeeds:   [][]float64{{0.5, 1}}, // seed → the area-2 region
	}}
	profs, err := e.resolveProfiles()
	if err != nil {
		t.Fatalf("resolveProfiles: %v", err)
	}
	if len(profs) != 1 {
		t.Fatalf("want 1 resolved profile, got %d", len(profs))
	}
	if a := profs[0].Area(); stdmath.Abs(a-2) > 1e-6 {
		t.Errorf("resolved region area %v, want the area-2 region the SEED selects (not the stale index's 6)", a)
	}
}

// TestResolveSeedsDropsMissingWhenSomeHit confirms a seed that hits no region is dropped as
// long as another seed resolves — a stray stale seed must not add a wrong (fallback) region.
func TestResolveSeedsDropsMissingWhenSomeHit(t *testing.T) {
	sk := splitRectSketch()
	// First seed hits the area-2 region; second seed is off the sheet (misses).
	got := resolveSeeds(sk, [][]float64{{0.5, 1}, {99, 99}}, []int{7, 7})
	if len(got) != 1 {
		t.Fatalf("want only the hit region (missed seed dropped), got %v", got)
	}
	if a := sk.Profiles().Item(got[0]).Area(); stdmath.Abs(a-2) > 1e-6 {
		t.Errorf("kept region area %v, want the area-2 region the first seed hit", a)
	}
}

// TestSeedFallbackAlignsToSeeds confirms seedFallback yields one load-time cell per seed, using
// the recipe's first index when a seed hits nothing, and the recipe list unchanged with no seeds.
func TestSeedFallbackAlignsToSeeds(t *testing.T) {
	sk := splitRectSketch()
	small, large := regionIndexOfArea(sk, 2), regionIndexOfArea(sk, 6)

	fb := seedFallback(sk, [][]float64{{0.5, 1}, {2.5, 1}}, []int{9})
	if len(fb) != 2 || fb[0] != small || fb[1] != large {
		t.Errorf("aligned fallback = %v, want [%d %d] (the two hit cells)", fb, small, large)
	}

	// A missing seed takes the recipe's first index (def0 = 9).
	miss := seedFallback(sk, [][]float64{{99, 99}}, []int{9})
	if len(miss) != 1 || miss[0] != 9 {
		t.Errorf("missed-seed fallback = %v, want [9] (recipe default)", miss)
	}

	// No seeds ⇒ the recipe index list unchanged.
	if none := seedFallback(sk, nil, []int{3, 4}); len(none) != 2 || none[0] != 3 || none[1] != 4 {
		t.Errorf("no-seed fallback = %v, want [3 4] (recipe unchanged)", none)
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
