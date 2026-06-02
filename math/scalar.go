// SPDX-License-Identifier: GPL-2.0-only

package math

import stdmath "math"

// Scalar is the precision used throughout the kernel and math layer. CAD work
// needs double precision; a single alias makes the choice explicit and
// flippable (architecture/core/01-module-layout.md, realtime-3d §13).
type Scalar = float64

// DefaultTolerance is the comparison tolerance used when a caller does not
// supply one. It is expressed in database units (cm) and is tight enough to
// distinguish distinct model features yet loose enough to absorb the rounding
// of a chain of float64 transforms.
const DefaultTolerance Scalar = 1e-9

// AngleTolerance is the default tolerance, in radians, for parallel/
// perpendicular and angular-equality tests.
const AngleTolerance Scalar = 1e-9

// approxEqual reports whether a and b are within tol of each other. tol is
// treated as an absolute tolerance; callers in database units want that.
func approxEqual(a, b, tol Scalar) bool {
	return stdmath.Abs(a-b) <= tol
}

// resolveTolerance returns tol when it is positive, otherwise [DefaultTolerance].
// It lets methods accept 0 to mean "use the default", mirroring the optional
// Tolerance argument on the COM contracts.
func resolveTolerance(tol Scalar) Scalar {
	if tol > 0 {
		return tol
	}
	return DefaultTolerance
}

// resolveAngleTolerance is [resolveTolerance] for angular tests: it falls back
// to [AngleTolerance] (radians) rather than the linear default.
func resolveAngleTolerance(tol Scalar) Scalar {
	if tol > 0 {
		return tol
	}
	return AngleTolerance
}

// IsNearZero reports whether x is within tol of zero. Pass tol <= 0 to use
// [DefaultTolerance]. It centralizes the "is this denominator/cross-product
// effectively zero?" guard that geometric queries branch on.
func IsNearZero(x, tol Scalar) bool {
	return approxEqual(x, 0, resolveTolerance(tol))
}

// clampUnit constrains x to [-1, 1]; used to guard math.Acos against inputs
// that float64 rounding pushed just outside the valid cosine range.
func clampUnit(x Scalar) Scalar {
	if x < -1 {
		return -1
	}
	if x > 1 {
		return 1
	}
	return x
}
