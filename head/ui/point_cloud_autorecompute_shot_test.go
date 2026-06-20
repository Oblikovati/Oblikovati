//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"encoding/json"
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/addin/router"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// TestInWindowAutoRecomputeOnCloudMove fits a plane to a scan, then moves the cloud through the real
// pointClouds.setTransform router method — the path an add-in or the UI uses — and captures the
// scene without any explicit recompute. The fitted plane must have already followed the cloud,
// confirming the auto-recompute on a cloud move (#645). Skips without a display.
func TestInWindowAutoRecomputeOnCloudMove(t *testing.T) {
	win := newShotWindow(t)
	defer win.Destroy()
	s := framedSession()
	pc, wp := attachSheetWithFitPlane(t, s)
	z0 := float64(wp.Plane().Origin().Z)

	// Move the cloud +12 in z via the router — and do NOT recompute by hand.
	r := router.New(opregistry.Default())
	args, _ := json.Marshal(wire.SetPointCloudTransformArgs{
		Name: "Sheet", Transform: types.TranslationMatrix(types.Vector{Z: 12}),
	})
	if _, err := r.Handle(s, "pointClouds.setTransform", args); err != nil {
		t.Fatalf("setTransform: %v", err)
	}
	z1 := float64(wp.Plane().Origin().Z)
	t.Logf("plane origin Z: %.3f before move, %.3f after the router move (no manual recompute)", z0, z1)
	if z1-z0 < 11 {
		t.Fatalf("plane did not auto-follow the cloud move: Z %.3f → %.3f", z0, z1)
	}

	captureFramed(t, win, s, pc.RangeBox(), "point-cloud-autorecompute")
}
