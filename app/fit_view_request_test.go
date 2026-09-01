// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/scene"
)

// TestFitViewRequestOneShot pins the #1645 intent's one-shot semantics: a fresh session has no
// pending fit, RequestFitView arms it, and TakeFitViewRequest both reads AND clears it so the head
// fits exactly once per import (a second Take returns false).
func TestFitViewRequestOneShot(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if s.TakeFitViewRequest() {
		t.Fatal("a fresh session must not have a pending fit-view request")
	}
	s.RequestFitView()
	if !s.TakeFitViewRequest() {
		t.Fatal("RequestFitView must arm the pending fit-view request")
	}
	if s.TakeFitViewRequest() {
		t.Fatal("the fit-view request must clear once taken (one-shot)")
	}
}

// writeScan writes a temporary .xyz scan file with the given "x y z" lines and returns its path.
func writeScan(t *testing.T, lines string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "scan.xyz")
	if err := os.WriteFile(path, []byte(lines), 0o600); err != nil {
		t.Fatalf("write temp scan: %v", err)
	}
	return path
}

// TestAttachPointCloudFarFromOriginFitsCamera is the #1645 regression for the "import did nothing"
// symptom on a scan far outside the current view: attaching a cloud arms the fit request, and the
// subsequent FitView (which now unions point-cloud bounds) frames the cloud — every corner ends up
// in front of a camera that started pointed the other way.
func TestAttachPointCloudFarFromOriginFitsCamera(t *testing.T) {
	t.Parallel()
	s, _ := newPartWithSquare(t, 2)
	path := writeScan(t, "100000 100000 100000\n100010 100000 100000\n100000 100010 100000\n")

	if _, _, err := s.AttachPointCloud("FarScan", path); err != nil {
		t.Fatalf("AttachPointCloud: %v", err)
	}
	if !s.TakeFitViewRequest() {
		t.Fatal("attaching a non-empty scan must arm the fit-view request (#1645)")
	}

	cam := scene.NewCamera(800, 600)
	cam.Eye = math.P3(0, 0, 500)
	cam.Target = math.P3(0, 0, 490) // looking away from the far cloud
	s.SetCamera(cam)
	s.FitView()

	box := s.modelBounds()
	if box.IsEmpty() {
		t.Fatal("model bounds empty — the attached point cloud was not unioned into the view bounds")
	}
	got := s.Camera()
	for _, corner := range box.Corners() {
		if v := got.Eye.VectorTo(corner); float64(v.Dot(got.Forward())) <= 0 {
			t.Errorf("cloud corner %v is behind the camera after the post-import fit", corner)
		}
	}
}

// TestAttachEmptyScanDoesNotFit is the negative case: a scan that yields no points adds no visible
// geometry, so it must NOT arm the fit request (the camera is left untouched, #1645).
func TestAttachEmptyScanDoesNotFit(t *testing.T) {
	t.Parallel()
	s, _ := newPartWithSquare(t, 2)
	path := writeScan(t, "# only a comment, no points\n")

	if _, _, err := s.AttachPointCloud("EmptyScan", path); err != nil {
		t.Fatalf("AttachPointCloud: %v", err)
	}
	if s.TakeFitViewRequest() {
		t.Error("an empty scan adds no visible geometry and must not arm the fit-view request")
	}
}
