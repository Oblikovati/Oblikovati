// SPDX-License-Identifier: GPL-2.0-only

package router

import "testing"

// extrudeBodies is the slimmed result shape of a features.add extrude call.
type extrudeBodies struct {
	Bodies int `json:"bodies"`
}

func TestExtrudeSymmetricDirectionViaAPI(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	var sk struct {
		SketchIndex int `json:"sketchIndex"`
	}
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &sk)
	var rect struct {
		Profiles int `json:"profiles"`
	}
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &rect)

	var ext extrudeBodies
	call(t, r, s, "features.add",
		`{"kind":"extrude","args":{"sketchIndex":0,"distance":"50 mm","direction":"symmetric","taper":"3 deg"}}`, &ext)
	if ext.Bodies != 1 {
		t.Fatalf("symmetric extrude via API produced %d bodies, want 1", ext.Bodies)
	}
}

func TestExtrudeUnknownExtentRejectedByAPI(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &struct{}{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"10 mm","height":"10 mm"}`, &struct{}{})
	if _, err := r.Handle(s, "features.add",
		[]byte(`{"kind":"extrude","args":{"sketchIndex":0,"distance":"5 mm","extent":"nonsense"}}`)); err == nil {
		t.Error("expected an error for an unknown extent")
	}
}
