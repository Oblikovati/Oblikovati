// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestFixedToFacePlaneRoundTrips pins the fix for a keyless to-face target. A to-face extrude whose
// target is a NewFixedWorkPlane (authored from geometry, not a datum in the part's work geometry —
// the form the Inventor to-face translator emits) has no WorkRef, so it was serialized as an empty
// ref and lost on reopen: the restored extent's ToPlane went nil, toPlaneSpan errored, and the
// feature built nothing (ST3215Bracket's walls vanished on save/reopen). Now the fixed plane's
// geometry is written and restored, so the target survives.
func TestFixedToFacePlaneRoundTrips(t *testing.T) {
	x, _ := math.NewUnitVector3(1, 0, 0)
	y, _ := math.NewUnitVector3(0, 1, 0)
	pl, err := sketch.NewPlane(math.P3(1, 2, 3), x, y)
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	ext := Extent{Type: ToFaceExtent, ToPlane: NewFixedWorkPlane(pl)}

	d := &ExtrudeData{Extent: "to-face"}
	serializeExtentPlanes(ext, d)
	if d.ToPlane != "" {
		t.Errorf("a keyless target must not serialize a ref; got %q", d.ToPlane)
	}
	if d.ToPlaneFixed == nil {
		t.Fatalf("keyless to-face target lost its geometry on serialize")
	}

	restored, err := restoreExtent(d, nil)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if restored.ToPlane == nil {
		t.Fatalf("to-face target went nil on restore — the reopened extrude would build nothing")
	}
	got := restored.ToPlane.Plane().Origin()
	if stdmath.Abs(float64(got.X)-1) > 1e-9 || stdmath.Abs(float64(got.Y)-2) > 1e-9 || stdmath.Abs(float64(got.Z)-3) > 1e-9 {
		t.Errorf("restored to-face origin = %v, want (1,2,3)", got)
	}
	n := restored.ToPlane.Plane().Normal().AsVector()
	if stdmath.Abs(float64(n.Z)-1) > 1e-9 {
		t.Errorf("restored to-face normal = %v, want +Z", n)
	}
}
