// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// TestCameraRoundTrips drives view.setCamera then view.getCamera and asserts the look-at
// frame survives — the dogfood proof the camera contract reaches the session and back.
func TestCameraRoundTrips(t *testing.T) {
	r, s := seededSession(t)
	args := `{"eye":[10,20,30],"target":[1,2,3],"up":[0,1,0],"fov":0.8}`

	var set wire.CameraView
	call(t, r, s, "view.setCamera", args, &set)
	if set.Eye != types.NewPoint(10, 20, 30) || set.Target != types.NewPoint(1, 2, 3) || set.FOV != 0.8 {
		t.Fatalf("setCamera = %+v, want eye=[10 20 30] target=[1 2 3] fov=0.8", set)
	}

	var got wire.CameraView
	call(t, r, s, "view.getCamera", "{}", &got)
	if got != set {
		t.Fatalf("getCamera = %+v, want %+v", got, set)
	}
}

// TestSetCameraRejectsInvalidFrames checks degenerate/out-of-range frames are errors, not
// a silently corrupted view.
func TestSetCameraRejectsInvalidFrames(t *testing.T) {
	r, s := seededSession(t)
	cases := map[string]string{
		"coincident eye/target": `{"eye":[1,1,1],"target":[1,1,1],"up":[0,1,0],"fov":0.8}`,
		"fov out of range":      `{"eye":[0,0,10],"target":[0,0,0],"up":[0,1,0],"fov":0}`,
		"up parallel to view":   `{"eye":[0,0,10],"target":[0,0,0],"up":[0,0,1],"fov":0.8}`,
		"zero up":               `{"eye":[0,0,10],"target":[0,0,0],"up":[0,0,0],"fov":0.8}`,
	}
	for name, args := range cases {
		if _, err := r.Handle(s, "view.setCamera", []byte(args)); err == nil {
			t.Errorf("%s: expected an error, got none", name)
		}
	}
}
