// SPDX-License-Identifier: GPL-2.0-only

// Package predicate provides exact geometric predicates — orientation and
// in-circle/in-sphere tests whose SIGN is always correct, even at near-degenerate
// configurations that fool naive float64 arithmetic. Boolean robustness lives or
// dies on these (architecture core/03): every branching decision in the kernel's
// operations ("which side of this plane", "do these intersect") must agree with a
// single consistent geometry, or topology becomes inconsistent.
//
// The implementation is adaptive: a fast float64 evaluation with a conservative
// error bound handles the common case; when the result is too close to zero to
// trust, it falls back to exact rational arithmetic (math/big.Rat — float64 values
// are exact rationals, so the rational determinant's sign is definitive). This is
// pure Go and cgo-free, simpler than Shewchuk's expansion arithmetic and exact for
// the sign, which is all the kernel needs.
package predicate
