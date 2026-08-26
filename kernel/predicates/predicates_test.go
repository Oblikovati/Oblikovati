// SPDX-License-Identifier: GPL-2.0-only

package predicates

import (
	"math/big"
	"math/rand"
	"testing"
)

// oracleOrient3D is an INDEPENDENT exact oracle: it expands the same 3x3
// determinant along the first column, a different grouping from exactOrient3D's
// (which groups by the z-terms). Both equal the true determinant, so a sign
// disagreement between Orient3D and this oracle exposes a transcription bug.
func oracleOrient3D(a, b, c, d [3]float64) int {
	adx, ady, adz := ratDiff(a[0], d[0]), ratDiff(a[1], d[1]), ratDiff(a[2], d[2])
	bdx, bdy, bdz := ratDiff(b[0], d[0]), ratDiff(b[1], d[1]), ratDiff(b[2], d[2])
	cdx, cdy, cdz := ratDiff(c[0], d[0]), ratDiff(c[1], d[1]), ratDiff(c[2], d[2])
	// det = adx*(bdy*cdz - bdz*cdy) - ady*(bdx*cdz - bdz*cdx) + adz*(bdx*cdy - bdy*cdx)
	t1 := new(big.Rat).Mul(adx, crossDiff(bdy, cdz, bdz, cdy))
	t2 := new(big.Rat).Mul(ady, crossDiff(bdx, cdz, bdz, cdx))
	t3 := new(big.Rat).Mul(adz, crossDiff(bdx, cdy, bdy, cdx))
	det := t1.Sub(t1, t2)
	det.Add(det, t3)
	return det.Sign()
}

// naiveOrient3D is the plain floating-point determinant with no filter and no
// exact fallback — the compromise these predicates replace. It is here only to
// prove the robustness tests actually reach the regime where it fails.
func naiveOrient3D(a, b, c, d [3]float64) int {
	adx, ady, adz := a[0]-d[0], a[1]-d[1], a[2]-d[2]
	bdx, bdy, bdz := b[0]-d[0], b[1]-d[1], b[2]-d[2]
	cdx, cdy, cdz := c[0]-d[0], c[1]-d[1], c[2]-d[2]
	det := adx*(bdy*cdz-bdz*cdy) - ady*(bdx*cdz-bdz*cdx) + adz*(bdx*cdy-bdy*cdx)
	return signOf(det)
}

func TestOrient2DBasicSigns(t *testing.T) {
	if got := Orient2D(0, 0, 1, 0, 0, 1); got != 1 {
		t.Fatalf("CCW triangle: got %d, want +1", got)
	}
	if got := Orient2D(0, 0, 0, 1, 1, 0); got != -1 {
		t.Fatalf("CW triangle: got %d, want -1", got)
	}
	if got := Orient2D(0, 0, 2, 2, 5, 5); got != 0 {
		t.Fatalf("collinear points: got %d, want 0", got)
	}
}

func TestOrient3DBasicSigns(t *testing.T) {
	// d=(0,0,1) is above the xy-plane through (0,0,0),(1,0,0),(0,1,0) → -1.
	if got := Orient3D(0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1); got != -1 {
		t.Fatalf("point above plane: got %d, want -1", got)
	}
	if got := Orient3D(0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, -1); got != 1 {
		t.Fatalf("point below plane: got %d, want +1", got)
	}
	if got := Orient3D(0, 0, 0, 1, 0, 0, 0, 1, 0, 2, 3, 0); got != 0 {
		t.Fatalf("coplanar point: got %d, want 0", got)
	}
}

// TestOrient3DExactMatchesOracleUnderStress drives the predicate through 20000
// near-coplanar configurations (d placed on the float-rounded plane of a,b,c) —
// the noise floor where a naive determinant misclassifies — and asserts Orient3D
// equals the independent exact oracle every time. It also requires that the naive
// predicate disagrees on a non-trivial number of them, so the suite proves it is
// actually exercising the hard regime rather than easy inputs.
func TestOrient3DExactMatchesOracleUnderStress(t *testing.T) {
	r := rand.New(rand.NewSource(0x2084))
	teeth := 0
	const n = 20000
	for i := range n {
		a, b, c := randPt(r), randPt(r), randPt(r)
		s, u := r.Float64(), r.Float64()
		var d [3]float64
		for k := range 3 {
			d[k] = a[k] + s*(b[k]-a[k]) + u*(c[k]-a[k]) // on plane(a,b,c) in real arithmetic
		}
		want := oracleOrient3D(a, b, c, d)
		got := Orient3D(a[0], a[1], a[2], b[0], b[1], b[2], c[0], c[1], c[2], d[0], d[1], d[2])
		if got != want {
			t.Fatalf("case %d: Orient3D=%d, oracle=%d (a=%v b=%v c=%v d=%v)", i, got, want, a, b, c, d)
		}
		if naiveOrient3D(a, b, c, d) != want {
			teeth++
		}
	}
	if teeth == 0 {
		t.Fatalf("robustness test never reached a case the naive predicate gets wrong (%d cases); it is not exercising the degenerate regime", n)
	}
	t.Logf("naive float predicate disagreed with the exact sign on %d/%d near-coplanar cases", teeth, n)
}

// naiveOrient2D is the plain floating-point 2D determinant, present only to prove
// the 2D stress test reaches the regime where it fails.
func naiveOrient2D(a, b, c [2]float64) int {
	det := (a[0]-c[0])*(b[1]-c[1]) - (a[1]-c[1])*(b[0]-c[0])
	return signOf(det)
}

// oracleOrient2D is the exact 2D orientation sign, computed straightforwardly.
func oracleOrient2D(a, b, c [2]float64) int {
	return exactOrient2D(a[0], a[1], b[0], b[1], c[0], c[1])
}

// TestOrient2DExactMatchesOracleUnderStress drives Orient2D through near-collinear
// configurations (c placed on the float-rounded line a→b) and asserts it equals
// the exact sign every time, with the naive predicate proven to fail on some.
func TestOrient2DExactMatchesOracleUnderStress(t *testing.T) {
	r := rand.New(rand.NewSource(0x2081))
	teeth := 0
	const n = 20000
	for i := range n {
		a := [2]float64{(r.Float64() - 0.5) * 2e5, (r.Float64() - 0.5) * 2e5}
		b := [2]float64{(r.Float64() - 0.5) * 2e5, (r.Float64() - 0.5) * 2e5}
		s := r.Float64()
		c := [2]float64{a[0] + s*(b[0]-a[0]), a[1] + s*(b[1]-a[1])} // on line a→b
		want := oracleOrient2D(a, b, c)
		if got := Orient2D(a[0], a[1], b[0], b[1], c[0], c[1]); got != want {
			t.Fatalf("case %d: Orient2D=%d, oracle=%d", i, got, want)
		}
		if naiveOrient2D(a, b, c) != want {
			teeth++
		}
	}
	if teeth == 0 {
		t.Fatalf("2D robustness test never reached a case the naive predicate gets wrong (%d cases)", n)
	}
	t.Logf("naive float predicate disagreed with the exact sign on %d/%d near-collinear cases", teeth, n)
}

// TestFilterNeverCertifiesWrongSign is the safety property of the fast path: on
// every input where the static filter reports "certified", its sign MUST equal
// the exact sign. A filter that ever certifies a wrong sign silently defeats the
// whole design, so this asserts it directly across the stress corpus.
func TestFilterNeverCertifiesWrongSign(t *testing.T) {
	r := rand.New(rand.NewSource(0x1822))
	for i := range 20000 {
		a, b, c := randPt(r), randPt(r), randPt(r)
		s, u := r.Float64(), r.Float64()
		var d [3]float64
		for k := range 3 {
			d[k] = a[k] + s*(b[k]-a[k]) + u*(c[k]-a[k])
		}
		det, certified := filterOrient3D(a[0], a[1], a[2], b[0], b[1], b[2], c[0], c[1], c[2], d[0], d[1], d[2])
		if !certified {
			continue
		}
		if signOf(det) != oracleOrient3D(a, b, c, d) {
			t.Fatalf("case %d: filter certified sign %d but exact sign is %d", i, signOf(det), oracleOrient3D(a, b, c, d))
		}
	}
}

// TestOrient3DAntisymmetry checks an invariant a correct predicate satisfies
// exactly and a naive float one violates near degeneracy: swapping two of the
// first three points negates the orientation.
func TestOrient3DAntisymmetry(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	for i := range 5000 {
		a, b, c, d := randPt(r), randPt(r), randPt(r), randPt(r)
		abcd := Orient3D(a[0], a[1], a[2], b[0], b[1], b[2], c[0], c[1], c[2], d[0], d[1], d[2])
		bacd := Orient3D(b[0], b[1], b[2], a[0], a[1], a[2], c[0], c[1], c[2], d[0], d[1], d[2])
		if abcd != -bacd {
			t.Fatalf("case %d: orient3d(a,b,c,d)=%d but orient3d(b,a,c,d)=%d (must be negatives)", i, abcd, bacd)
		}
	}
}

// randPt returns a point with coordinates in [-1e5, 1e5]. Large magnitudes make
// the pairwise products lose low-order bits, so the on-plane construction in the
// stress tests lands squarely at the floating-point noise floor.
func randPt(r *rand.Rand) [3]float64 {
	return [3]float64{
		(r.Float64() - 0.5) * 2e5,
		(r.Float64() - 0.5) * 2e5,
		(r.Float64() - 0.5) * 2e5,
	}
}
