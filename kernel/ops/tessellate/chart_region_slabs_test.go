// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"math/rand"
	"testing"

	"oblikovati.org/math"
)

// TestSlabIndexAnswersExactlyAsTheEvenOddRule holds the whole claim the index rests on: for every query
// it returns what pointInUVPoly returns, bit for bit. The queries are chosen to be where an index would
// go wrong — rows passing exactly through vertices, horizontal edges, the top and bottom of the range,
// points outside it, and non-finite values — as well as random ones.
func TestSlabIndexAnswersExactlyAsTheEvenOddRule(t *testing.T) {
	rng := rand.New(rand.NewSource(3558))
	for _, poly := range slabTestPolygons(rng) {
		s, ok := newContourSlabs(poly)
		if !ok {
			t.Fatalf("a finite %d-point contour could not be indexed", len(poly))
		}
		for _, q := range slabTestQueries(rng, poly) {
			if got, want := s.contains(poly, q), pointInUVPoly(poly, q); got != want {
				t.Errorf("query %v on a %d-point contour: indexed %v, even-odd rule %v", q, len(poly), got, want)
			}
		}
	}
}

// TestSlabIndexRefusesWhatItCannotIndexExactly: floor(·) has no slab for a non-finite coordinate, so such
// a contour is not indexed at all rather than indexed wrongly.
func TestSlabIndexRefusesWhatItCannotIndexExactly(t *testing.T) {
	for _, poly := range [][]math.Point2{
		{math.P2(0, 0), math.P2(1, 0)},
		{math.P2(0, 0), math.P2(1, stdmath.NaN()), math.P2(0, 1)},
		{math.P2(0, 0), math.P2(stdmath.Inf(1), 0), math.P2(0, 1)},
	} {
		if _, ok := newContourSlabs(poly); ok {
			t.Errorf("indexed %v, which has no exact slab", poly)
		}
	}
}

// slabTestPolygons: a square, a star with horizontal runs, a zero-height sliver, and random polygons.
func slabTestPolygons(rng *rand.Rand) [][]math.Point2 {
	polys := [][]math.Point2{
		{math.P2(0, 0), math.P2(2, 0), math.P2(2, 2), math.P2(0, 2)},
		{math.P2(0, 0), math.P2(3, 0), math.P2(3, 1), math.P2(2, 1), math.P2(2, 3), math.P2(1, 3), math.P2(1, 1), math.P2(0, 1)},
		{math.P2(0, 1), math.P2(5, 1), math.P2(2, 1)},
	}
	for range 30 {
		n := 3 + rng.Intn(60)
		poly := make([]math.Point2, n)
		for i := range poly {
			a := 2 * stdmath.Pi * float64(i) / float64(n)
			r := 0.5 + rng.Float64()
			poly[i] = math.P2(r*stdmath.Cos(a), r*stdmath.Sin(a))
		}
		polys = append(polys, poly)
	}
	return polys
}

// slabTestQueries: every vertex's own row at several x, the range's ends and beyond, non-finite values,
// and random points.
func slabTestQueries(rng *rand.Rand, poly []math.Point2) [][2]float64 {
	var qs [][2]float64
	for _, v := range poly {
		for _, x := range []float64{-10, float64(v.X) - 1e-9, float64(v.X), float64(v.X) + 1e-9, 10} {
			qs = append(qs, [2]float64{x, float64(v.Y)})
		}
	}
	for range 400 {
		qs = append(qs, [2]float64{4*rng.Float64() - 2, 4*rng.Float64() - 2})
	}
	return append(qs,
		[2]float64{0, stdmath.NaN()}, [2]float64{0, stdmath.Inf(1)}, [2]float64{0, stdmath.Inf(-1)},
		[2]float64{stdmath.NaN(), 0.5}, [2]float64{0, -1e300}, [2]float64{0, 1e300})
}
