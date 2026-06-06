// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/math"
)

// TestDraftTapersSideFace drafts the +X face of a 2×2×2 box inward by atan(0.25) about +Z:
// the face hinges at its base (z=0) and leans in, removing a 4·tan(angle)=1 wedge ⇒ vol 7.
func TestDraftTapersSideFace(t *testing.T) {
	box := shellBox(2, 2, 2)
	var side []byte
	for _, f := range box.Faces() {
		if f.Geometry().NormalAt(0, 0).X > 0.99 {
			side = f.ReferenceKey()
		}
	}
	res, err := ops.DraftFaces(box, [][]byte{side}, math.V3(0, 0, 1), -stdmath.Atan(0.25))
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("drafted body not a valid solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; stdmath.Abs(got-7) > 1e-6 {
		t.Errorf("draft volume = %g, want 7", got)
	}
}

// TestDraftLostKeyErrors reports a vanished face key so the feature can go Sick.
func TestDraftLostKeyErrors(t *testing.T) {
	box := shellBox(2, 2, 2)
	if _, err := ops.DraftFaces(box, [][]byte{[]byte("ghost")}, math.V3(0, 0, 1), 0.1); err == nil {
		t.Error("draft with a lost key should error")
	}
}
