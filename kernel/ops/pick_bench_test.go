// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// sphereBallBody is a single analytic-sphere body — one ball of the self-aligning thrust bearing,
// whose 16 copies the viewport hover-pick ray-tests EVERY frame during an orbit.
func sphereBallBody(t testing.TB) *topo.Body {
	body, err := brep.SolidSphere(math.V3(0, 0, 0).AsPoint(), 1.0, "ball")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	return body
}

// BenchmarkRayCastFacesSphere measures the per-ray cost of picking one analytic-sphere body. The
// hovered-plane pick runs RayCastFaces on every body every frame during an orbit; without the
// face-lifetime pick-tessellation memo this re-tessellated the sphere per ray (~151 µs, 151 allocs),
// and 16 balls per frame starved 60 fps. With the memo the repeat cost is ray-vs-cached-mesh only.
func BenchmarkRayCastFacesSphere(b *testing.B) {
	body := sphereBallBody(b)
	origin := math.V3(0, 0, 5).AsPoint()
	dir := math.V3(0, 0, -1)
	q := ops.DefaultQuality()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, ok := ops.RayCastFaces(body, origin, dir, q); !ok {
			b.Fatal("ray missed the sphere it is aimed at")
		}
	}
}

// TestRayCastFacesCurvedDoesNotRetessellatePerCall is the regression fence the recurring pick
// starvation keeps needing (planar #1913, then the analytic-sphere balls): once a curved face has
// been picked, subsequent rays MUST reuse the memoized tessellation, not rebuild it. A rebuild shows
// up as ~150 allocations/op (a fresh sphere Mesh); the memoized path is a handful. We assert a low
// per-call allocation ceiling so deleting or breaking the cache fails CI loudly.
func TestRayCastFacesCurvedDoesNotRetessellatePerCall(t *testing.T) {
	body := sphereBallBody(t)
	origin := math.V3(0, 0, 5).AsPoint()
	dir := math.V3(0, 0, -1)
	q := ops.DefaultQuality()
	// Warm the memo, then measure the steady-state (orbit) cost of repeated picks.
	if _, _, ok := ops.RayCastFaces(body, origin, dir, q); !ok {
		t.Fatal("warm-up ray missed the sphere")
	}
	const ceiling = 16 // memoized: a couple of small slices; re-tessellation would be ~150
	avg := testing.AllocsPerRun(200, func() {
		if _, _, ok := ops.RayCastFaces(body, origin, dir, q); !ok {
			t.Fatal("steady-state ray missed the sphere")
		}
	})
	if avg > ceiling {
		t.Fatalf("RayCastFaces on a curved face allocates %.0f/op after warm-up (ceiling %d): the "+
			"pick-tessellation memo is not being reused — every orbit frame re-tessellates the face", avg, ceiling)
	}
}

// TestPickTessMemoMatchesFreshTessellation guards correctness: the memoized pick must return the same
// hit as a cold pick. A stale or wrong memo would silently mispick — worse than being slow.
func TestPickTessMemoMatchesFreshTessellation(t *testing.T) {
	origin := math.V3(0, 0, 5).AsPoint()
	dir := math.V3(0, 0, -1)
	q := ops.DefaultQuality()

	cold := sphereBallBody(t) // never picked before → cold tessellation
	_, tCold, okCold := ops.RayCastFaces(cold, origin, dir, q)

	warm := sphereBallBody(t)
	ops.RayCastFaces(warm, origin, dir, q) // prime the memo
	_, tWarm, okWarm := ops.RayCastFaces(warm, origin, dir, q)

	if !okCold || !okWarm {
		t.Fatalf("expected both picks to hit the sphere (cold=%v warm=%v)", okCold, okWarm)
	}
	if diff := tCold - tWarm; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("memoized pick distance %v differs from cold %v (Δ=%g): the memo changed the hit", tWarm, tCold, diff)
	}
	// The near hit of a unit sphere centred at the origin, from z=5 toward −z, is at z=1 → distance 4.
	if tWarm < 3.9 || tWarm > 4.1 {
		t.Fatalf("sphere near-hit distance %v, want ≈4 (origin z=5, radius 1)", tWarm)
	}
}
