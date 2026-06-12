// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"testing"

	"oblikovati.org/api/types"
)

func TestCollectionSettingsDefaults(t *testing.T) {
	ps := NewParameters()
	s := ps.Settings()
	if s.LinearDimensionPrecision != 3 || s.AngularDimensionPrecision != 2 {
		t.Errorf("default precisions = %d/%d, want 3/2", s.LinearDimensionPrecision, s.AngularDimensionPrecision)
	}
	if s.DimensionDisplayType != types.DimensionDisplayValue || s.UseStandardTolerances {
		t.Errorf("defaults = %+v, want value display and standard tolerances off", s)
	}
	// Settings edit in place through the accessor.
	s.UseStandardTolerances = true
	if !ps.Settings().UseStandardTolerances {
		t.Error("Settings() must expose the live settings, not a copy")
	}
}

func TestSetAllModelValueTypeSweepsTolerancedOnly(t *testing.T) {
	ps := NewParameters()
	plain, _ := ps.AddUserParameter("plain", "10 mm")
	tolA, _ := ps.AddUserParameter("a", "10 mm")
	tolB, _ := ps.AddUserParameter("b", "20 mm")
	txt, _ := ps.AddTextUserParameter("label", "lid")
	_ = tolA.SetToleranceDeviation(0.02, -0.01)
	_ = tolB.SetToleranceSymmetric(0.05)

	affected, err := ps.SetAllModelValueType(Upper)
	if err != nil {
		t.Fatalf("SetAllModelValueType: %v", err)
	}
	if affected != 2 {
		t.Errorf("affected = %d, want the 2 toleranced parameters", affected)
	}
	if tolA.ModelValueType() != Upper || tolB.ModelValueType() != Upper {
		t.Errorf("sweep missed a toleranced parameter: %s/%s", tolA.ModelValueType(), tolB.ModelValueType())
	}
	if !approxScalar(tolA.ModelValue(), 1.02) || !approxScalar(tolB.ModelValue(), 2.05) {
		t.Errorf("model values = %v/%v, want 1.02/2.05 (nominal + upper)", tolA.ModelValue(), tolB.ModelValue())
	}
	// Untoleranced and non-numeric parameters are left alone.
	if plain.ModelValueType() != Nominal || txt.ModelValueType() != Nominal {
		t.Error("sweep must not touch untoleranced parameters")
	}

	if _, err := ps.SetAllModelValueType(ModelValueType(9)); err == nil {
		t.Error("unknown sweep selection must be rejected")
	}
}
