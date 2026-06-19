// SPDX-License-Identifier: GPL-2.0-only

package sheetmetal

import "testing"

// TestOrientationLookupMissErrors covers the "orientation does not exist" branches: activating,
// copying or deleting an unknown orientation name is rejected.
func TestOrientationLookupMissErrors(t *testing.T) {
	o := NewOrientations()
	if err := o.Activate("ghost"); err == nil {
		t.Error("Activate(ghost) should error")
	}
	if _, err := o.Copy("ghost", "x"); err == nil {
		t.Error("Copy(ghost) should error")
	}
	if err := o.Delete("ghost"); err == nil {
		t.Error("Delete(ghost) should error")
	}
}
