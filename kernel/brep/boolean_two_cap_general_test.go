// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// A steep rod that enters a cylinder through one CAP and leaves through the other, never touching the
// wall, must build through the general pipeline. It is the configuration the two-cap recognizer was
// written for (#1724), and the general path reached it only once an empty wall-versus-wall verdict
// counted as the PROOF it is: the two walls' infinite surfaces do cross, every crossing lies clear of
// one of the two bands, and the target's wall therefore stays whole (ADR-0061 stage 4).
//
// The result is four faces: the whole wall, both caps holed by their exit ellipses, and the tunnel.
func TestTwoCapCrossingCutBuildsThroughTheGeneralPath(t *testing.T) {
	t.Parallel()
	th := 20.0 * stdmath.Pi / 180
	target, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("SolidCylinder target: %v", err)
	}
	tool, err := SolidCylinder(math.P3(-2.416, 0, -2.518),
		math.V3(math.Scalar(stdmath.Sin(th)), 0, math.Scalar(stdmath.Cos(th))), 0.7, 16)
	if err != nil {
		t.Fatalf("SolidCylinder tool: %v", err)
	}
	res, err := Boolean(Difference, target, tool)
	if err != nil {
		t.Fatalf("Boolean(Difference) on the two-cap crossing: %v", err)
	}
	assertWatertight(t, res)
	if got := len(res.Faces()); got != 4 {
		t.Errorf("two-cap cut has %d faces, want 4 (whole wall + two holed caps + tunnel)", got)
	}
	assertEveryFaceWinds(t, res)
}

// TestClearWallPairIsCarried is the unit statement of the same rule: two walls whose infinite surfaces
// cross OUTSIDE both bands are DECIDED clear, not undecided. wallWallImprint returns no curves and
// ok=true, and the pairing must read that as covered.
func TestClearWallPairIsCarried(t *testing.T) {
	t.Parallel()
	th := 20.0 * stdmath.Pi / 180
	target, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	tool, _ := SolidCylinder(math.P3(-2.416, 0, -2.518),
		math.V3(math.Scalar(stdmath.Sin(th)), 0, math.Scalar(stdmath.Cos(th))), 0.7, 16)
	pa, pb := partitionFaces(target), partitionFaces(tool)
	if len(pa.wall) != 1 || len(pb.wall) != 1 {
		t.Fatalf("fixture: target has %d walls, tool %d; want one each", len(pa.wall), len(pb.wall))
	}
	curves, ok := wallWallImprint(pa.wall[0], pb.wall[0])
	if !ok {
		t.Fatal("the wall pair is undecided; the two-cap crossing needs it decided")
	}
	if len(curves) != 0 {
		t.Errorf("the wall pair produced %d imprint curves; the tool never touches the target's wall", len(curves))
	}
	if overlapsUncarriedWall(pa.wall[0], inflateBox(pa.wallBox[0]), &pb) {
		t.Error("a decided-clear wall pair still reads as uncovered")
	}
}
