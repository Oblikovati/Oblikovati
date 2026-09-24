// SPDX-License-Identifier: GPL-2.0-only

package math

import (
	stdmath "math"
	"runtime"
	"testing"
)

// TestExplicitConversionBlocksContraction is the RUNTIME half of ADR-0064's FMA policy: the
// archguard walk proves every product in this package is written with its conversion, and this
// proves the conversion still does what the policy needs it to do.
//
// Go LICENSES the compiler to contract x*y+z into one unrounded FMA, and gc takes that licence on
// arm64. The policy rests on one spec guarantee — "an explicit floating-point type conversion
// rounds to the precision of the target type" — so if a future toolchain ever elided a
// Scalar-to-Scalar conversion before its SSA contraction pass, every conversion in this package
// would go quiet without a single test turning red. This one turns red, and it turns red on the
// macOS (arm64) CI leg where the contraction actually happens.
//
// 0.1*0.2 - 0.02 is the smallest witness: the true product 0.020000000000000004 rounds one ulp
// above 0.02, so the unfused difference is 2^-58 while the fused one keeps the exact tail.
func TestExplicitConversionBlocksContraction(t *testing.T) {
	t.Parallel()
	a, b, c := contractionWitness()
	fused := stdmath.FMA(a, b, c)
	unfused := roundedProductSum(a, b, c)
	if unfused == fused {
		t.Fatalf("Scalar(a*b)+c == math.FMA(a,b,c) == %v on %s/%s: the explicit conversion no "+
			"longer forces a rounding, so ADR-0064's whole policy is a no-op and kernel output is "+
			"not byte-identical across platforms", fused, runtime.GOOS, runtime.GOARCH)
	}
	if want := 3.4694469519536142e-18; unfused != want {
		t.Fatalf("unfused product-sum = %v, want %v (the separately rounded value)", unfused, want)
	}
}

// TestContractionIsWhatTheConversionPrevents pins the other side: WITHOUT the conversion the two
// forms agree on amd64 and disagree on arm64. It documents the platform split the policy exists to
// remove, and it is the reason the plain form may not be written in this package.
func TestContractionIsWhatTheConversionPrevents(t *testing.T) {
	t.Parallel()
	a, b, c := contractionWitness()
	plain := plainProductSum(a, b, c)
	switch runtime.GOARCH {
	case "amd64":
		if plain != roundedProductSum(a, b, c) {
			t.Fatalf("amd64 contracted a*b+c (%v); it never has, and every stored fingerprint "+
				"assumes it never will", plain)
		}
	case "arm64":
		if plain != stdmath.FMA(a, b, c) {
			t.Fatalf("arm64 did NOT contract a*b+c (%v): if the toolchain stopped contracting, "+
				"ADR-0064's cost argument and its conversions should be revisited", plain)
		}
	}
}

// contractionWitness returns 0.1, 0.2, -0.02 as RUNTIME values. Constants would defeat the test:
// Go evaluates a constant expression at compile time, where no contraction happens on any platform.
//
//go:noinline
func contractionWitness() (Scalar, Scalar, Scalar) { return 0.1, 0.2, -0.02 }

// roundedProductSum is the policy form.
//
//go:noinline
func roundedProductSum(a, b, c Scalar) Scalar { return Scalar(a*b) + c }

// plainProductSum is the form the policy forbids, kept here as the ONE deliberate witness of what
// it forbids — it is the only unrounded product-sum in this module, and it is in a test.
//
//go:noinline
func plainProductSum(a, b, c Scalar) Scalar { return a*b + c }
