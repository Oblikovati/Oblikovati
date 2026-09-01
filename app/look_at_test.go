// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"
)

// facesNormal asserts the camera now looks straight along n (forward ∥ n) after the swing completes.
func facesNormal(t *testing.T, s *Session, n [3]float64) {
	t.Helper()
	runCameraAnimation(s)
	f := s.Camera().Forward()
	dot := stdmath.Abs(float64(f.X)*n[0] + float64(f.Y)*n[1] + float64(f.Z)*n[2])
	if dot < 1-1e-6 {
		t.Errorf("view not facing the reference: |fwd·n| = %v, want 1", dot)
	}
}

// TestLookAtSelectionFacesWorkPlane: with a work plane selected, Look At swings the view normal to it.
func TestLookAtSelectionFacesWorkPlane(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	wp := def.OriginPlanes()[0]
	s.Selection().Add(WorkPlaneHandle{Plane: wp})
	if !s.CanLookAt() {
		t.Fatal("CanLookAt should be true with a work plane selected")
	}
	if !s.LookAtSelection() {
		t.Fatal("LookAtSelection should report the work plane as a planar reference")
	}
	n := wp.Plane().Normal().AsVector()
	facesNormal(t, s, [3]float64{float64(n.X), float64(n.Y), float64(n.Z)})
}

// TestLookAtSelectionFacesPlanarFace: with a planar face selected, Look At faces its normal (+Z here).
func TestLookAtSelectionFacesPlanarFace(t *testing.T) {
	t.Parallel()
	s, _ := emptyPartSession(t)
	s.Selection().Add(FaceHandle{Face: aFace()}) // a +Z planar face
	if !s.CanLookAt() {
		t.Fatal("CanLookAt should be true with a planar face selected")
	}
	if !s.LookAtSelection() {
		t.Fatal("LookAtSelection should report the planar face as a reference")
	}
	facesNormal(t, s, [3]float64{0, 0, 1})
}

// TestLookAtSelectionNoPlanarSelection: nothing planar selected ⇒ Look At is a reported no-op.
func TestLookAtSelectionNoPlanarSelection(t *testing.T) {
	t.Parallel()
	s, _ := emptyPartSession(t)
	if s.CanLookAt() {
		t.Error("CanLookAt with nothing selected should be false")
	}
	if s.LookAtSelection() {
		t.Error("LookAtSelection with nothing selected should report false")
	}
	if s.CameraAnimating() {
		t.Error("a no-op Look At must not start a camera transition")
	}
}
