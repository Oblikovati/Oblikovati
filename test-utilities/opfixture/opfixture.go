// SPDX-License-Identifier: GPL-2.0-only

// Package opfixture builds the test operands that only an operation can produce.
//
// It is separate from brepfixture because brepfixture must stay a pure builder over geom/topo:
// kernel/ops/validate's own tests use it, and validate sits BELOW the operations, so a fixture
// package that imported one would close a cycle.
package opfixture

import (
	"testing"

	"oblikovati.org/kernel/ops/boolean"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/test-utilities/brepfixture"
)

// Cavity builds the canonical two-shell solid (#629): a 4×4×4 box with a centred 2×2×2 void,
// so the body has an outer skin and a void skin.
//
// Example: b := brepfixture.Cavity(t) // len(b.Shells()) == 2, the inner one a void
func Cavity(tb testing.TB) *topo.Body {
	tb.Helper()
	res, err := boolean.Boolean(boolean.Cut, brepfixture.Box(math.P3(0, 0, 0), 4, 4, 4), brepfixture.Box(math.P3(1, 1, 1), 2, 2, 2))
	if err != nil {
		tb.Fatalf("cavity cut: %v", err)
	}
	return res
}
