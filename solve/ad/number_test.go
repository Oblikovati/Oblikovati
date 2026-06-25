// SPDX-License-Identifier: GPL-2.0-only

package ad

import (
	stdmath "math"
	"testing"
)

// fdGrad approximates ∇f at x by central differences — the trusted oracle the
// analytic gradients are checked against (finite differencing is fine in a test;
// the point of #1417 is to keep it out of the solve path).
func fdGrad(f func([]float64) float64, x []float64) []float64 {
	const h = 1e-6
	g := make([]float64, len(x))
	for i := range x {
		xp := append([]float64(nil), x...)
		xm := append([]float64(nil), x...)
		xp[i] += h
		xm[i] -= h
		g[i] = (f(xp) - f(xm)) / (2 * h)
	}
	return g
}

func assertGradMatchesFD(t *testing.T, name string, f func([]Number) Number, x []float64) {
	t.Helper()
	got := f(Seed(x)).Grad()
	want := fdGrad(func(v []float64) float64 { return f(Consts(v)).Val() }, x)
	if len(got) != len(want) {
		t.Fatalf("%s: gradient length %d, want %d", name, len(got), len(want))
	}
	for i := range want {
		if stdmath.Abs(got[i]-want[i]) > 1e-5 {
			t.Errorf("%s: ∂/∂x%d = %.9f, FD says %.9f", name, i, got[i], want[i])
		}
	}
}

func TestArithmeticGradientsMatchFiniteDifference(t *testing.T) {
	cases := []struct {
		name string
		f    func([]Number) Number
	}{
		{"add", func(v []Number) Number { return v[0].Add(v[1]) }},
		{"sub", func(v []Number) Number { return v[0].Sub(v[1]) }},
		{"mul", func(v []Number) Number { return v[0].Mul(v[1]) }},
		{"div", func(v []Number) Number { return v[0].Div(v[1]) }},
		{"neg", func(v []Number) Number { return v[0].Neg() }},
		{"scale", func(v []Number) Number { return v[0].Scale(2.5) }},
		{"addconst", func(v []Number) Number { return v[0].AddConst(3) }},
		{"sqrt", func(v []Number) Number { return v[0].Mul(v[0]).Add(v[1]).Sqrt() }},
		{"hypot", func(v []Number) Number { return v[0].Hypot(v[1]) }},
		{"abs+", func(v []Number) Number { return v[0].Abs() }},
		{"abs-", func(v []Number) Number { return v[0].Neg().Abs() }},
		{"atan2", func(v []Number) Number { return v[0].Atan2(v[1]) }},
		{"sin", func(v []Number) Number { return v[0].Sin() }},
		{"cos", func(v []Number) Number { return v[0].Cos() }},
		{"composite", func(v []Number) Number {
			return v[0].Mul(v[1]).Sub(v[0].Hypot(v[1])).Div(v[1].AddConst(2))
		}},
	}
	x := []float64{1.3, 2.7}
	for _, c := range cases {
		assertGradMatchesFD(t, c.name, c.f, x)
	}
}

func TestVectorOpsGradientsMatchFiniteDifference(t *testing.T) {
	// v = [ax, ay, az, bx, by, bz]
	a := func(v []Number) Vec3 { return V3(v[0], v[1], v[2]) }
	b := func(v []Number) Vec3 { return V3(v[3], v[4], v[5]) }
	cases := []struct {
		name string
		f    func([]Number) Number
	}{
		{"dot3", func(v []Number) Number { return a(v).Dot(b(v)) }},
		{"cross3.x", func(v []Number) Number { return a(v).Cross(b(v)).X }},
		{"len3", func(v []Number) Number { return a(v).Length() }},
		{"sub.dot", func(v []Number) Number { return a(v).Sub(b(v)).Dot(a(v)) }},
		{"cross2", func(v []Number) Number { return V2(v[0], v[1]).Cross(V2(v[3], v[4])) }},
		{"len2", func(v []Number) Number { return V2(v[0], v[1]).Length() }},
	}
	x := []float64{1.1, -2.2, 3.3, 0.7, 1.9, -0.4}
	for _, c := range cases {
		assertGradMatchesFD(t, c.name, c.f, x)
	}
}

func TestConstHasNoGradient(t *testing.T) {
	if g := Const(5).Add(Const(2)).Grad(); g != nil {
		t.Errorf("constant arithmetic produced a gradient %v, want nil", g)
	}
}

func TestVarSeedsUnitGradient(t *testing.T) {
	v := Var(7, 1, 3)
	if v.Val() != 7 {
		t.Errorf("value = %v, want 7", v.Val())
	}
	want := []float64{0, 1, 0}
	for i, g := range v.Grad() {
		if g != want[i] {
			t.Fatalf("grad = %v, want %v", v.Grad(), want)
		}
	}
}

func TestVectorOpValues(t *testing.T) {
	// Spot-check the value (not gradient) of each vector helper at constants.
	a2, b2 := V2(Const(1), Const(2)), V2(Const(3), Const(5))
	if v := a2.Add(b2); v.X.Val() != 4 || v.Y.Val() != 7 {
		t.Errorf("Vec2.Add = (%v,%v), want (4,7)", v.X.Val(), v.Y.Val())
	}
	if v := b2.Sub(a2); v.X.Val() != 2 || v.Y.Val() != 3 {
		t.Errorf("Vec2.Sub = (%v,%v), want (2,3)", v.X.Val(), v.Y.Val())
	}
	if v := a2.Scale(2); v.X.Val() != 2 || v.Y.Val() != 4 {
		t.Errorf("Vec2.Scale = (%v,%v), want (2,4)", v.X.Val(), v.Y.Val())
	}
	if v := a2.MulN(Const(3)); v.X.Val() != 3 || v.Y.Val() != 6 {
		t.Errorf("Vec2.MulN = (%v,%v), want (3,6)", v.X.Val(), v.Y.Val())
	}
	if d := a2.Dot(b2).Val(); d != 13 {
		t.Errorf("Vec2.Dot = %v, want 13", d)
	}

	a3, b3 := V3(Const(1), Const(2), Const(3)), V3(Const(4), Const(5), Const(6))
	if v := a3.Add(b3); v.X.Val() != 5 || v.Z.Val() != 9 {
		t.Errorf("Vec3.Add = %v, want X=5,Z=9", v)
	}
	if v := b3.Scale(0.5); v.Y.Val() != 2.5 {
		t.Errorf("Vec3.Scale Y = %v, want 2.5", v.Y.Val())
	}
	if v := a3.MulN(Const(2)); v.Z.Val() != 6 {
		t.Errorf("Vec3.MulN Z = %v, want 6", v.Z.Val())
	}
}

func TestIndexErrorMessage(t *testing.T) {
	if got := adIndexError(5, 3).Error(); got != "ad: variable index 5 out of range for a 3-variable system" {
		t.Errorf("error message = %q", got)
	}
}

func TestVarPanicsOnBadIndex(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Var with out-of-range index did not panic")
		}
	}()
	Var(1, 5, 3)
}
