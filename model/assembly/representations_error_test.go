// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"testing"

	"oblikovati.org/math"
)

// TestRepresentationLookupMissErrors covers the "no representation with id" branches: every
// mutator and activation rejects an unknown representation id.
func TestRepresentationLookupMissErrors(t *testing.T) {
	reps, occs, _, _ := newReps()
	occ := place(occs, "a:1", math.Identity4())
	const missing = uint64(404)

	if _, err := reps.ActivateDesignView(missing); err == nil {
		t.Error("ActivateDesignView(missing) should error")
	}
	if _, err := reps.ActivatePositional(missing); err == nil {
		t.Error("ActivatePositional(missing) should error")
	}
	wantErr := func(name string, err error) {
		t.Helper()
		if err == nil {
			t.Errorf("%s(missing) should error", name)
		}
	}
	wantErr("SetVisibility", reps.SetVisibility(missing, occ, true))
	wantErr("SetAppearance", reps.SetAppearance(missing, occ, "x"))
	wantErr("SetFlexible", reps.SetFlexible(missing, occ, true))
	wantErr("SetSuppressed", reps.SetSuppressed(missing, occ, true))
	wantErr("SetPositionalOverride", reps.SetPositionalOverride(missing, 1, false, 0))
}
