// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"testing"

	"oblikovati.org/api/types"
)

// TestDrawingViewLookupMissErrors covers the "no view named" branches of the view editors and
// removal — each rejects an unknown view name.
func TestDrawingViewLookupMissErrors(t *testing.T) {
	vs := NewContent().Sheets().Active().Views()
	if err := vs.EditBase("ghost", types.BaseViewFront, types.HiddenLineViewStyle, 1, 0, 0); err == nil {
		t.Error("EditBase(ghost) should error")
	}
	if err := vs.EditProjected("ghost", types.ProjectRight, 0, 0); err == nil {
		t.Error("EditProjected(ghost) should error")
	}
	if err := vs.Remove("ghost"); err == nil {
		t.Error("Remove(ghost) should error")
	}
}
