// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	stdmath "math"
	"math/big"
	"math/rand"
	"testing"

	"oblikovati.org/kernel/predicates"
	"oblikovati.org/math"
)

func TestPointRoundTripAndEqual(t *testing.T) {
	p := FromCoords(1.5, -2.25, 3.0) // exactly representable
	got := p.Round()
	if got != (math.P3(1.5, -2.25, 3.0)) {
		t.Fatalf("round-trip: got %v, want (1.5,-2.25,3)", got)
	}
	if !p.Equal(FromPoint3(math.P3(1.5, -2.25, 3.0))) {
		t.Fatal("Equal: identical coordinates compared unequal")
	}
	if p.Equal(FromCoords(1.5, -2.25, 3.0000001)) {
		t.Fatal("Equal: distinct coordinates compared equal")
	}
}

func TestFromCoordsRejectsNonFinite(t *testing.T) {
	for _, bad := range []float64{stdmath.Inf(1), stdmath.Inf(-1), stdmath.NaN()} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatalf("FromCoords(%v,...) did not panic on a non-finite coordinate", bad)
				}
			}()
			_ = FromCoords(bad, 0, 0)
		}()
	}
}

// TestOrient3DMatchesPredicates is the critical cross-check: the rational Orient3D
// on Points must give the identical sign to predicates.Orient3D on the same
// float64 coordinates, so the co-refinement can mix original and constructed
// vertices with no sign-convention seam. It runs both the general regime and the
// near-coplanar noise floor (where predicates takes its exact path).
func TestOrient3DMatchesPredicates(t *testing.T) {
	r := rand.New(rand.NewSource(0x1c01))
	for i := range 20000 {
		a, b, c := rc(r), rc(r), rc(r)
		var d [3]float64
		if i%2 == 0 {
			// on plane(a,b,c) in real arithmetic → forces the exact path
			s, u := r.Float64(), r.Float64()
			for k := range 3 {
				d[k] = a[k] + s*(b[k]-a[k]) + u*(c[k]-a[k])
			}
		} else {
			d = rc(r)
		}
		want := predicates.Orient3D(a[0], a[1], a[2], b[0], b[1], b[2], c[0], c[1], c[2], d[0], d[1], d[2])
		got := Orient3D(pt(a), pt(b), pt(c), pt(d))
		if got != want {
			t.Fatalf("case %d: meshbool.Orient3D=%d, predicates.Orient3D=%d", i, got, want)
		}
	}
}

// TestEdgePlaneCrossExact checks that a constructed crossing lies EXACTLY on both
// the plane and the segment (zero orientation determinant; zero cross product with
// the edge direction; each coordinate between the endpoints), over random
// straddling edges. This is the conforming-vertex guarantee at the atom level.
func TestEdgePlaneCrossExact(t *testing.T) {
	r := rand.New(rand.NewSource(0x1c02))
	built := 0
	for i := range 20000 {
		a, b, c := pt(rc(r)), pt(rc(r)), pt(rc(r))
		e0, e1 := pt(rc(r)), pt(rc(r))
		s0, s1 := Orient3D(a, b, c, e0), Orient3D(a, b, c, e1)
		if s0 == 0 || s1 == 0 || s0 == s1 {
			continue // not strictly straddling — outside the precondition
		}
		x := EdgePlaneCross(e0, e1, a, b, c)
		if Orient3D(a, b, c, x) != 0 {
			t.Fatalf("case %d: crossing not on plane (orient=%d)", i, Orient3D(a, b, c, x))
		}
		if !collinear(e0, e1, x) {
			t.Fatalf("case %d: crossing not collinear with the edge", i)
		}
		if !between(e0, e1, x) {
			t.Fatalf("case %d: crossing not between the endpoints", i)
		}
		built++
	}
	if built == 0 {
		t.Fatal("no straddling edge was constructed; test exercised nothing")
	}
	t.Logf("exact edge-plane crossings verified: %d", built)
}

// --- helpers ---

func rc(r *rand.Rand) [3]float64 {
	return [3]float64{(r.Float64() - 0.5) * 2e5, (r.Float64() - 0.5) * 2e5, (r.Float64() - 0.5) * 2e5}
}

func pt(p [3]float64) Point { return FromCoords(p[0], p[1], p[2]) }

// collinear reports whether x lies on line e0-e1 exactly ((x-e0)×(e1-e0)==0).
func collinear(e0, e1, x Point) bool {
	u := x.sub(e0)
	v := e1.sub(e0)
	cx := crossDiff(u[1], v[2], u[2], v[1])
	cy := crossDiff(u[2], v[0], u[0], v[2])
	cz := crossDiff(u[0], v[1], u[1], v[0])
	return cx.Sign() == 0 && cy.Sign() == 0 && cz.Sign() == 0
}

// between reports whether each coordinate of x lies within [min,max] of the
// endpoints — necessary for a point on the closed segment.
func between(e0, e1, x Point) bool {
	return inRange(e0.X, e1.X, x.X) && inRange(e0.Y, e1.Y, x.Y) && inRange(e0.Z, e1.Z, x.Z)
}

func inRange(a, b, v *big.Rat) bool {
	lo, hi := a, b
	if lo.Cmp(hi) > 0 {
		lo, hi = hi, lo
	}
	return v.Cmp(lo) >= 0 && v.Cmp(hi) <= 0
}
