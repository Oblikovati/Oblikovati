// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati/addin/modelaccess"
	"oblikovati/api/types"
	"oblikovati/api/wire"
	"oblikovati/app"
)

func TestModelSelectionReportsReferences(t *testing.T) {
	r, s := emptyPartSession(t)
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatalf("active part: %v", err)
	}
	// Select the XY origin plane and the X origin axis; their references should come back
	// parallel to the kinds, ready to feed workPlanes.create.
	s.Selection().Add(app.WorkPlaneHandle{Plane: part.OriginPlanes()[0]})
	s.Selection().Add(app.WorkAxisHandle{Axis: part.WorkAxes().Item(0)})

	var sel wire.SelectionResult
	call(t, r, s, "model.selection", "{}", &sel)
	if sel.Count != 2 || len(sel.Refs) != 2 {
		t.Fatalf("selection = %+v, want 2 items with 2 refs", sel)
	}
	if sel.Refs[0] != types.WorkRefXYPlane || sel.Refs[1] != types.WorkRefXAxis {
		t.Errorf("refs = %v, want [%s %s]", sel.Refs, types.WorkRefXYPlane, types.WorkRefXAxis)
	}
}
