// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"math"
	"testing"

	"oblikovati.org/api/wire"
)

// viewport.scroll exists so a client can drive the camera the way a real mouse wheel
// does (#1822's own router handler, mechanically completed here — the wire method and
// its DTOs already existed; only the router registration was missing).

// eyeTargetDistance is the look-at distance a wheel notch's Dolly moves.
func eyeTargetDistance(v wire.CameraView) float64 {
	dx, dy, dz := v.Eye.X-v.Target.X, v.Eye.Y-v.Target.Y, v.Eye.Z-v.Target.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// TestScrollZoomsTheCamera: a positive dy (scroll up) is the zoom-IN direction — the
// camera moves closer to its target.
func TestScrollZoomsTheCamera(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "view.setCamera", `{"eye":[0,0,10],"target":[0,0,0],"up":[0,1,0],"fov":0.8}`, &wire.CameraView{})
	var before wire.CameraView
	call(t, r, s, "view.getCamera", "{}", &before)

	var res wire.ScrollViewportResult
	call(t, r, s, "viewport.scroll", `{"dy":1}`, &res)

	var after wire.CameraView
	call(t, r, s, "view.getCamera", "{}", &after)
	if d1, d2 := eyeTargetDistance(before), eyeTargetDistance(after); d2 >= d1 {
		t.Errorf("distance after a positive-dy scroll = %v, want less than the starting %v (zoom in)", d2, d1)
	}
}

// TestScrollZeroDeltaIsANoOp: dy=0 must leave the camera untouched, not e.g. treat a
// missing field as some default notch count.
func TestScrollZeroDeltaIsANoOp(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "view.setCamera", `{"eye":[0,0,10],"target":[0,0,0],"up":[0,1,0],"fov":0.8}`, &wire.CameraView{})
	var before wire.CameraView
	call(t, r, s, "view.getCamera", "{}", &before)

	call(t, r, s, "viewport.scroll", `{}`, &wire.ScrollViewportResult{})

	var after wire.CameraView
	call(t, r, s, "view.getCamera", "{}", &after)
	if after != before {
		t.Errorf("camera after a zero-delta scroll = %+v, want unchanged %+v", after, before)
	}
}
