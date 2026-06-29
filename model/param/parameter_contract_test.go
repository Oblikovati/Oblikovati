// SPDX-License-Identifier: GPL-2.0-only

package param_test

import (
	"testing"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/model/param"
)

// TestParameterContractProjections checks the M39-F06 (#1562) additions to contract.Parameter:
// a host parameter, read through the contract interface, reports its nominal value, unit
// category, tolerance band, and health.
func TestParameterContractProjections(t *testing.T) {
	ps := param.NewParameters()
	p, err := ps.AddUserParameter("width", "40 mm") // 40 mm == 4 cm (database unit)
	if err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}
	var c contract.Parameter = p // compile- and run-time: *Parameter satisfies the contract

	if c.NominalValue() < 3.999 || c.NominalValue() > 4.001 {
		t.Errorf("NominalValue = %v, want ~4 (40 mm in db units)", c.NominalValue())
	}
	if c.UnitName() != "length" {
		t.Errorf("UnitName = %q, want \"length\"", c.UnitName())
	}
	if !c.IsHealthy() || c.HealthReason() != "" {
		t.Errorf("fresh parameter health = (healthy=%v reason=%q), want healthy/none", c.IsHealthy(), c.HealthReason())
	}
	if c.Tolerance() != (types.Tolerance{}) {
		t.Errorf("default tolerance = %+v, want the zero band", c.Tolerance())
	}

	// A symmetric tolerance surfaces on the contract.
	if err := p.SetToleranceSymmetric(0.05); err != nil {
		t.Fatalf("SetToleranceSymmetric: %v", err)
	}
	tol := c.Tolerance()
	if tol.Upper != 0.05 || tol.Lower != -0.05 {
		t.Errorf("tolerance band = %+v, want ±0.05", tol)
	}
}

// TestUnhealthyParameterReportsReason checks a failed parameter surfaces a non-empty reason
// through the contract's lean health projection (the status enum stays host-internal, #1501).
func TestUnhealthyParameterReportsReason(t *testing.T) {
	ps := param.NewParameters()
	p, err := ps.AddUserParameter("bad", "missing + 1") // references an undefined parameter
	if err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}
	var c contract.Parameter = p
	if c.IsHealthy() || c.HealthReason() == "" {
		t.Errorf("undefined-reference parameter = (healthy=%v reason=%q), want unhealthy with a reason", c.IsHealthy(), c.HealthReason())
	}
}
