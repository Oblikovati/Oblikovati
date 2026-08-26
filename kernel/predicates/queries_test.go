// SPDX-License-Identifier: GPL-2.0-only

package predicates

import (
	"math/big"
	"math/rand"
	"testing"
)

func TestInTriangleCoplanarBasic(t *testing.T) {
	a := [3]float64{0, 0, 0}
	b := [3]float64{4, 0, 0}
	c := [3]float64{0, 4, 0}
	cases := []struct {
		p    [3]float64
		want TriRegion
	}{
		{[3]float64{1, 1, 0}, Inside},
		{[3]float64{2, 0, 0}, OnBoundary}, // on edge ab
		{[3]float64{0, 0, 0}, OnBoundary}, // vertex a
		{[3]float64{2, 2, 0}, OnBoundary}, // on edge bc (x+y=4)
		{[3]float64{5, 5, 0}, Outside},    // beyond
		{[3]float64{-1, 1, 0}, Outside},   // left of edge ca
	}
	for i, tc := range cases {
		if got := InTriangleCoplanar(a, b, c, tc.p); got != tc.want {
			t.Fatalf("case %d p=%v: got %d, want %d", i, tc.p, got, tc.want)
		}
	}
}

// TestInTriangleCoplanarVsOracle uses integer coordinates (exactly coplanar in a
// common tilted plane, so the Orient3D==0 precondition holds without rounding) and
// checks the classification against a rational barycentric oracle over a dense
// affine grid that spans interior, edges, vertices, and exterior.
func TestInTriangleCoplanarVsOracle(t *testing.T) {
	r := rand.New(rand.NewSource(0x1539))
	for trial := range 400 {
		a := randIntPt(r)
		b := randIntPt(r)
		c := randIntPt(r)
		axis := dominantNormalAxis(a, b, c)
		au, av := drop(a, axis)
		bu, bv := drop(b, axis)
		cu, cv := drop(c, axis)
		if Orient2D(au, av, bu, bv, cu, cv) == 0 {
			continue // degenerate (a,b,c collinear) — outside the non-degenerate contract
		}
		for mi := -2; mi <= 4; mi++ {
			for ni := -2; ni <= 4; ni++ {
				m, n := float64(mi)/2, float64(ni)/2 // half-integer combos stay exact
				var p [3]float64
				for k := range 3 {
					p[k] = a[k] + m*(b[k]-a[k]) + n*(c[k]-a[k])
				}
				if orient3(a, b, c, p) != 0 {
					continue // p not exactly coplanar (rounding) — outside the contract
				}
				want := oracleTriRegion(a, b, c, p)
				if got := InTriangleCoplanar(a, b, c, p); got != want {
					t.Fatalf("trial %d m=%g n=%g: got %d want %d (a=%v b=%v c=%v p=%v)", trial, m, n, got, want, a, b, c, p)
				}
			}
		}
	}
}

// TestSegmentPiercesTriangleVsOracle compares the transversal-pierce test against
// an independent oracle that constructs the plane-crossing point as an exact
// rational and tests strict containment, over random segments plus segments aimed
// through the centroid so genuine pierces are exercised.
func TestSegmentPiercesTriangleVsOracle(t *testing.T) {
	r := rand.New(rand.NewSource(0x2072))
	pierces := 0
	const n = 20000
	for i := range n {
		a, b, c := randPt(r), randPt(r), randPt(r)
		var p, q [3]float64
		if i%2 == 0 {
			// Aim through the centroid, perpendicular-ish, to force genuine pierces.
			var g [3]float64
			for k := range 3 {
				g[k] = (a[k] + b[k] + c[k]) / 3
			}
			d := randPt(r)
			for k := range 3 {
				p[k] = g[k] - d[k]
				q[k] = g[k] + d[k]
			}
		} else {
			p, q = randPt(r), randPt(r)
		}
		want := oraclePierces(a, b, c, p, q)
		if got := SegmentPiercesTriangle(p, q, a, b, c); got != want {
			t.Fatalf("case %d: got %v want %v (a=%v b=%v c=%v p=%v q=%v)", i, got, want, a, b, c, p, q)
		}
		if want {
			pierces++
		}
	}
	if pierces == 0 {
		t.Fatalf("pierce test never produced a true pierce in %d cases; not exercising the true branch", n)
	}
	t.Logf("exact transversal pierces: %d/%d", pierces, n)
}

// TestSegmentPiercesTriangleBoundaryContract pins the deliberate exclusions: only
// a strict straddle through the strict interior counts. Endpoint-on-plane,
// edge-grazing, and clean misses all return false; a clean interior pierce is true.
func TestSegmentPiercesTriangleBoundaryContract(t *testing.T) {
	a := [3]float64{0, 0, 0}
	b := [3]float64{4, 0, 0}
	c := [3]float64{0, 4, 0}
	cases := []struct {
		name string
		p, q [3]float64
		want bool
	}{
		{"clean interior pierce", [3]float64{1, 1, -1}, [3]float64{1, 1, 1}, true},
		{"endpoint on plane", [3]float64{1, 1, 0}, [3]float64{1, 1, 1}, false},
		{"pierce on edge ab", [3]float64{2, 0, -1}, [3]float64{2, 0, 1}, false},
		{"straddle but miss triangle", [3]float64{5, 5, -1}, [3]float64{5, 5, 1}, false},
		{"both above plane", [3]float64{1, 1, 1}, [3]float64{1, 1, 2}, false},
	}
	for _, tc := range cases {
		if got := SegmentPiercesTriangle(tc.p, tc.q, a, b, c); got != tc.want {
			t.Fatalf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

// --- oracles (independent, exact via math/big.Rat) ---

// oracleTriRegion classifies p (coplanar with a,b,c) via rational barycentric-sign
// areas, projected onto the dominant axis, entirely in exact arithmetic.
func oracleTriRegion(a, b, c, p [3]float64) TriRegion {
	axis := dominantNormalAxis(a, b, c)
	o1 := ratOrient2(a, b, p, axis)
	o2 := ratOrient2(b, c, p, axis)
	o3 := ratOrient2(c, a, p, axis)
	hasNeg := o1 < 0 || o2 < 0 || o3 < 0
	hasPos := o1 > 0 || o2 > 0 || o3 > 0
	if hasNeg && hasPos {
		return Outside
	}
	if o1 == 0 || o2 == 0 || o3 == 0 {
		return OnBoundary
	}
	return Inside
}

// oraclePierces builds the exact rational point where line pq meets plane(a,b,c)
// and reports whether the endpoints strictly straddle the plane and that point is
// strictly inside the triangle.
func oraclePierces(a, b, c, p, q [3]float64) bool {
	dp := ratOrient3Val(a, b, c, p)
	dq := ratOrient3Val(a, b, c, q)
	if dp.Sign() == 0 || dq.Sign() == 0 || dp.Sign() == dq.Sign() {
		return false
	}
	// x = p + t(q-p), t = dp/(dp-dq): the exact plane-crossing point.
	tt := new(big.Rat).Quo(dp, new(big.Rat).Sub(dp, dq))
	var x [3]*big.Rat
	for k := range 3 {
		diff := new(big.Rat).Sub(ratOf(q[k]), ratOf(p[k]))
		x[k] = new(big.Rat).Add(ratOf(p[k]), new(big.Rat).Mul(tt, diff))
	}
	axis := dominantNormalAxis(a, b, c)
	s1 := ratOrient2Pt(ratPt(a), ratPt(b), x, axis)
	s2 := ratOrient2Pt(ratPt(b), ratPt(c), x, axis)
	s3 := ratOrient2Pt(ratPt(c), ratPt(a), x, axis)
	return s1 != 0 && s1 == s2 && s2 == s3
}

// ratOrient2 is the exact 2D orientation sign of (a,b,p) projected on axis.
func ratOrient2(a, b, p [3]float64, axis int) int {
	return ratOrient2Pt(ratPt(a), ratPt(b), ratPt(p), axis)
}

// ratOrient2Pt is the exact 2D orientation sign of rational points a,b,p on axis.
func ratOrient2Pt(a, b, p [3]*big.Rat, axis int) int {
	au, av := ratDrop(a, axis)
	bu, bv := ratDrop(b, axis)
	pu, pv := ratDrop(p, axis)
	left := new(big.Rat).Mul(new(big.Rat).Sub(au, pu), new(big.Rat).Sub(bv, pv))
	right := new(big.Rat).Mul(new(big.Rat).Sub(av, pv), new(big.Rat).Sub(bu, pu))
	return left.Sub(left, right).Sign()
}

// ratOrient3Val returns the exact 3D orientation determinant value of a,b,c,d.
func ratOrient3Val(a, b, c, d [3]float64) *big.Rat {
	adx, ady, adz := ratDiff(a[0], d[0]), ratDiff(a[1], d[1]), ratDiff(a[2], d[2])
	bdx, bdy, bdz := ratDiff(b[0], d[0]), ratDiff(b[1], d[1]), ratDiff(b[2], d[2])
	cdx, cdy, cdz := ratDiff(c[0], d[0]), ratDiff(c[1], d[1]), ratDiff(c[2], d[2])
	t1 := new(big.Rat).Mul(adz, crossDiff(bdx, cdy, cdx, bdy))
	t2 := new(big.Rat).Mul(bdz, crossDiff(cdx, ady, adx, cdy))
	t3 := new(big.Rat).Mul(cdz, crossDiff(adx, bdy, bdx, ady))
	return t1.Add(t1, t2).Add(t1, t3)
}

func ratPt(p [3]float64) [3]*big.Rat {
	return [3]*big.Rat{ratOf(p[0]), ratOf(p[1]), ratOf(p[2])}
}

func ratDrop(p [3]*big.Rat, axis int) (u, v *big.Rat) {
	switch axis {
	case 0:
		return p[1], p[2]
	case 1:
		return p[0], p[2]
	default:
		return p[0], p[1]
	}
}

// randIntPt returns a point with small integer coordinates, so affine combinations
// with half-integer weights stay exactly representable and exactly coplanar.
func randIntPt(r *rand.Rand) [3]float64 {
	return [3]float64{
		float64(r.Intn(21) - 10),
		float64(r.Intn(21) - 10),
		float64(r.Intn(21) - 10),
	}
}
