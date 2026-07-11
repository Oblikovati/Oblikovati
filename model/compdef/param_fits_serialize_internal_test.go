// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/param"
)

// TestFitsToleranceRoundTrip: an ISO fits tolerance survives the parameter recipe round-trip
// with its band and class strings intact (#1848). Without the recipe carrying HoleTolerance/
// ShaftTolerance the reopened parameter would keep the band but lose the fit annotation.
func TestFitsToleranceRoundTrip(t *testing.T) {
	src := param.NewParameters()
	p, err := src.AddUserParameter("bore", "5 cm")
	if err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}
	if err := p.SetToleranceFits("H7", "g6"); err != nil {
		t.Fatalf("SetToleranceFits: %v", err)
	}

	recs := parametersRecipeOf(src)
	dst := param.NewParameters()
	if err := applyParametersTo(dst, nil, recs); err != nil {
		t.Fatalf("applyParametersTo: %v", err)
	}

	got, ok := dst.ByName("bore")
	if !ok {
		t.Fatalf("bore missing after round-trip")
	}
	tol := got.Tolerance()
	if tol.Type != types.ToleranceLimitsFitsStacked || tol.HoleTolerance != "H7" || tol.ShaftTolerance != "g6" {
		t.Errorf("restored tolerance = %+v, want limitsFitsStacked H7/g6", tol)
	}
	if tol.Upper < 0.00249 || tol.Upper > 0.00251 {
		t.Errorf("restored band upper = %v cm, want ~0.0025 (50H7)", tol.Upper)
	}
}
