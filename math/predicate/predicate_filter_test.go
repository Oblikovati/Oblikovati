// SPDX-License-Identifier: GPL-2.0-only

package predicate

import (
	stdmath "math"
	"math/rand"
	"testing"

	"oblikovati.org/math"
)

// Regression for Oblikovati/Oblikovati#1323 L4: the static float filter was tightened from a uniform
// 1e-14 to per-predicate constants just above Shewchuk's proven A-level bounds. These tests prove the
// tightened band is (a) still sign-exact — never trusts a wrong float sign — and (b) genuinely tighter
// (resolves cases the old 1e-14 sent to the exact path), the performance win.

const oldErrScale = 1e-14 // the previous uniform filter, for the before/after comparison

func sgn(x float64) int {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	default:
		return 0
	}
}

// nearCollinear2 returns a, b and a c that sits a tiny perpendicular distance off line a→b, so the
// orient2d determinant lands in the filter's uncertain band.
func nearCollinear2(rng *rand.Rand) (a, b, c math.Point2) {
	a = math.P2(math.Scalar(rng.Float64()*20-10), math.Scalar(rng.Float64()*20-10))
	b = math.P2(math.Scalar(rng.Float64()*20-10), math.Scalar(rng.Float64()*20-10))
	s := math.Scalar(rng.Float64()*2 - 0.5)
	perp := math.Scalar((rng.Float64()*2 - 1) * stdmath.Pow(10, -rng.Float64()*16))
	dir := a.VectorTo(b)
	n := math.V2(-float64(dir.Y), float64(dir.X))
	mid := a.TranslateBy(dir.Scale(s))
	c = mid.TranslateBy(n.Scale(perp))
	return a, b, c
}

// TestOrient2DTightenedFilterIsExactAndTighter checks both properties on orient2d.
func TestOrient2DTightenedFilterIsExactAndTighter(t *testing.T) {
	rng := rand.New(rand.NewSource(14))
	trustedNewMismatch, newResolvesOldExact := 0, 0
	for i := 0; i < 300000; i++ {
		a, b, c := nearCollinear2(rng)
		left := (a.X - c.X) * (b.Y - c.Y)
		right := (a.Y - c.Y) * (b.X - c.X)
		det := float64(left - right)
		mag := stdmath.Abs(float64(left)) + stdmath.Abs(float64(right))
		newBound := orient2DFilter * mag
		oldBound := oldErrScale * mag
		exact := orient2DExact(a, b, c)
		if stdmath.Abs(det) > newBound && sgn(det) != exact {
			trustedNewMismatch++ // the tightened filter trusted a WRONG sign — must never happen
		}
		if stdmath.Abs(det) > newBound && stdmath.Abs(det) <= oldBound {
			newResolvesOldExact++ // new filter resolves by float what old sent to exact
		}
		// The public predicate must always equal the exact sign.
		if sgn(Orient2D(a, b, c)) != exact {
			t.Fatalf("Orient2D sign %d != exact %d at case %d", sgn(Orient2D(a, b, c)), exact, i)
		}
	}
	if trustedNewMismatch != 0 {
		t.Errorf("tightened orient2d filter trusted a wrong float sign in %d cases", trustedNewMismatch)
	}
	if newResolvesOldExact == 0 {
		t.Error("tightened orient2d filter never beat the old 1e-14 bound — no perf win demonstrated")
	}
	t.Logf("orient2d: %d cases newly resolved by float (were exact under 1e-14)", newResolvesOldExact)
}

// nearCoplanar3 returns a,b,c and a d a tiny distance off their plane.
func nearCoplanar3(rng *rand.Rand) (a, b, c, d math.Point3) {
	rp := func() math.Point3 {
		return math.P3(math.Scalar(rng.Float64()*20-10), math.Scalar(rng.Float64()*20-10), math.Scalar(rng.Float64()*20-10))
	}
	a, b, c = rp(), rp(), rp()
	n := a.VectorTo(b).Cross(a.VectorTo(c))
	s := math.Scalar(rng.Float64())
	u := math.Scalar(rng.Float64())
	base := a.TranslateBy(a.VectorTo(b).Scale(s)).TranslateBy(a.VectorTo(c).Scale(u))
	off := math.Scalar((rng.Float64()*2 - 1) * stdmath.Pow(10, -rng.Float64()*16))
	d = base.TranslateBy(n.Scale(off))
	return a, b, c, d
}

// TestOrient3DTightenedFilterIsExactAndTighter checks both properties on orient3d.
func TestOrient3DTightenedFilterIsExactAndTighter(t *testing.T) {
	rng := rand.New(rand.NewSource(56))
	mismatch, tighter := 0, 0
	for i := 0; i < 300000; i++ {
		a, b, c, d := nearCoplanar3(rng)
		det, mag := orient3DFloat(a, b, c, d)
		newBound := orient3DFilter * mag
		oldBound := oldErrScale * mag
		exact := orient3DExact(a, b, c, d)
		if stdmath.Abs(det) > newBound && sgn(det) != exact {
			mismatch++
		}
		if stdmath.Abs(det) > newBound && stdmath.Abs(det) <= oldBound {
			tighter++
		}
		if sgn(Orient3D(a, b, c, d)) != exact {
			t.Fatalf("Orient3D sign != exact at case %d", i)
		}
	}
	if mismatch != 0 {
		t.Errorf("tightened orient3d filter trusted a wrong float sign in %d cases", mismatch)
	}
	if tighter == 0 {
		t.Error("tightened orient3d filter never beat the old 1e-14 bound")
	}
	t.Logf("orient3d: %d cases newly resolved by float", tighter)
}

// nearCocircular2 returns a,b,c and a d near their circumcircle.
func nearCocircular2(rng *rand.Rand) (a, b, c, d math.Point2) {
	cx, cy := rng.Float64()*10-5, rng.Float64()*10-5
	r := rng.Float64()*5 + 1
	at := func(ang float64) math.Point2 {
		return math.P2(math.Scalar(cx+r*stdmath.Cos(ang)), math.Scalar(cy+r*stdmath.Sin(ang)))
	}
	a, b, c = at(rng.Float64()*6.28), at(rng.Float64()*6.28), at(rng.Float64()*6.28)
	rr := r + (rng.Float64()*2-1)*stdmath.Pow(10, -rng.Float64()*15)
	ang := rng.Float64() * 6.28
	d = math.P2(math.Scalar(cx+rr*stdmath.Cos(ang)), math.Scalar(cy+rr*stdmath.Sin(ang)))
	return a, b, c, d
}

// TestInCircleTightenedFilterIsExactAndTighter checks both properties on incircle.
func TestInCircleTightenedFilterIsExactAndTighter(t *testing.T) {
	rng := rand.New(rand.NewSource(96))
	mismatch, tighter := 0, 0
	for i := 0; i < 300000; i++ {
		a, b, c, d := nearCocircular2(rng)
		det, mag := inCircleFloat(a, b, c, d)
		newBound := inCircleFilter * mag
		oldBound := oldErrScale * mag
		exact := inCircleExact(a, b, c, d)
		if stdmath.Abs(det) > newBound && sgn(det) != exact {
			mismatch++
		}
		if stdmath.Abs(det) > newBound && stdmath.Abs(det) <= oldBound {
			tighter++
		}
		if sgn(InCircle(a, b, c, d)) != exact {
			t.Fatalf("InCircle sign != exact at case %d", i)
		}
	}
	if mismatch != 0 {
		t.Errorf("tightened incircle filter trusted a wrong float sign in %d cases", mismatch)
	}
	if tighter == 0 {
		t.Error("tightened incircle filter never beat the old 1e-14 bound")
	}
	t.Logf("incircle: %d cases newly resolved by float", tighter)
}
