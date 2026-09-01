// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	"testing"

	"oblikovati.org/kernel/ops/boolean"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/test-utilities/brepfixture"
)

// TestFacetReturnsAValidSolid pins the post-condition #3329 added: the cage Facet returns becomes
// an operand the exact planar boolean trusts, so it must be a valid solid or be refused. Measured
// over the whole tier-2 corpus, Facet never produced an invalid cage — this keeps that true.
func TestFacetReturnsAValidSolid(t *testing.T) {
	t.Parallel()
	box := brepfixture.Box(math.P3(0, 0, 0), 2, 3, 4)
	faceted := boolean.Facet(box, "facet-postcondition")
	if faceted == nil {
		t.Fatal("Facet refused a plain box: it is already a valid closed solid")
	}
	if r := validate.Validate(faceted); !r.ValidSolid() {
		t.Fatalf("Facet returned a body that is not a valid solid: %+v", r)
	}
}

// TestFacetRefusesAnUnfacetableBody documents the refusal contract its callers rely on: nil means
// "I could not produce a trustworthy cage", and model/feature.planarized then keeps the analytic
// body rather than proceeding with a bad operand. A body with no faces has no triangles to weld,
// so it reaches the refusal deterministically.
//
// Note Facet is NOT nil-safe — a nil body is a caller bug, not an unsupported configuration, and
// panicking surfaces it where returning nil would hide it. planarized guards for nil itself.
func TestFacetRefusesAnUnfacetableBody(t *testing.T) {
	t.Parallel()
	empty := topo.NewBuilder(true, topo.NewLineage(topo.Tok("facet-empty", "body", 0))).Build()
	if got := boolean.Facet(empty, "facet-empty"); got != nil {
		t.Fatalf("Facet of a faceless body = %v, want nil — an unfacetable input must be refused", got)
	}
}
