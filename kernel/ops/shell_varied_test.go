// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// Per-face wall thickness (#1864). The overrides are checked by VOLUME against the analytic
// cavity, not by validity: a shell that quietly ignored an override would still be a valid solid,
// just the wrong one — which is exactly how a part imports with the wrong wall.

// faceKeyByNormal returns the reference key of the face whose outward normal points along n.
func faceKeyByNormal(t *testing.T, b *topo.Body, nx, ny, nz float64) []byte {
	t.Helper()
	for _, f := range b.Faces() {
		n := f.Geometry().NormalAt(0, 0)
		if float64(n.X)*nx+float64(n.Y)*ny+float64(n.Z)*nz > 0.99 {
			return f.ReferenceKey()
		}
	}
	t.Fatalf("no face with normal (%g,%g,%g)", nx, ny, nz)
	return nil
}

// shellVolume shells the box and returns the result's volume, failing on an invalid solid.
func shellVolume(t *testing.T, box *topo.Body, removed [][]byte, thick float64,
	dir ops.ShellDirection, over []ops.ShellFaceThickness) float64 {
	t.Helper()
	res, err := ops.ShellVaried(box, removed, thick, dir, over)
	if err != nil {
		t.Fatalf("ShellVaried: %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("varied shell not a valid solid: %+v", r)
	}
	return ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
}

// TestShellThickensOneWallInward: an open-top 4³ box walled 0.5, with the +X wall thickened to
// 1.5. The cavity loses exactly the extra 1.0 off its X span, and nothing else moves.
func TestShellThickensOneWallInward(t *testing.T) {
	box := shellBox(4, 4, 4)
	over := []ops.ShellFaceThickness{{FaceKey: faceKeyByNormal(t, box, 1, 0, 0), Thickness: 1.5}}
	got := shellVolume(t, box, [][]byte{topFaceKey(t, box)}, 0.5, ops.ShellInside, over)
	// Cavity [0.5,2.5]×[0.5,3.5]×[0.5,4] = 2·3·3.5 = 21 → wall 64 − 21 = 43.
	// The uniform shell would leave 32.5, so an ignored override is unmissable here.
	if want := 43.0; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("thickened-wall shell volume = %g, want %g (32.5 means the override was ignored)", got, want)
	}
}

// TestShellThinsOneWallInward: the override cuts both ways — a thin window in an otherwise
// uniform shell, which is the case where getting it wrong makes a part that cannot be moulded.
func TestShellThinsOneWallInward(t *testing.T) {
	box := shellBox(4, 4, 4)
	over := []ops.ShellFaceThickness{{FaceKey: faceKeyByNormal(t, box, 1, 0, 0), Thickness: 0.25}}
	got := shellVolume(t, box, [][]byte{topFaceKey(t, box)}, 0.5, ops.ShellInside, over)
	// Cavity [0.5,3.75]×[0.5,3.5]×[0.5,4] = 3.25·3·3.5 = 34.125 → wall 64 − 34.125 = 29.875.
	if want := 29.875; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("thinned-wall shell volume = %g, want %g", got, want)
	}
}

// TestShellVariedOutsideGrowsOnlyThatFace: outward, a per-face thickness moves that face's OUTER
// surface, so the part's overall size changes by the override and not by the default.
func TestShellVariedOutsideGrowsOnlyThatFace(t *testing.T) {
	box := shellBox(4, 4, 4)
	over := []ops.ShellFaceThickness{{FaceKey: faceKeyByNormal(t, box, 1, 0, 0), Thickness: 1.5}}
	got := shellVolume(t, box, [][]byte{topFaceKey(t, box)}, 0.5, ops.ShellOutside, over)
	// Outer [-0.5,5.5]×[-0.5,4.5]×[-0.5,4] = 6·5·4.5 = 135, minus the original 64 → 71.
	if want := 71.0; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("outward varied shell volume = %g, want %g", got, want)
	}
}

// TestShellVariedRejectsARemovedFace: a removed face is an opening — it has no wall, so a
// thickness for it is a selection mistake and honouring it silently would move the opening.
func TestShellVariedRejectsARemovedFace(t *testing.T) {
	box := shellBox(4, 4, 4)
	top := topFaceKey(t, box)
	_, err := ops.ShellVaried(box, [][]byte{top}, 0.5, ops.ShellInside,
		[]ops.ShellFaceThickness{{FaceKey: top, Thickness: 1.0}})
	if err == nil {
		t.Fatal("a thickness on a removed face should be refused")
	}
}

// TestShellVariedRejectsBadThickness: a non-positive override would invert the offset and eat the
// part from the outside; a lost key must go sick rather than fall back to the default wall.
func TestShellVariedRejectsBadThickness(t *testing.T) {
	box := shellBox(4, 4, 4)
	side := faceKeyByNormal(t, box, 1, 0, 0)
	if _, err := ops.ShellVaried(box, [][]byte{topFaceKey(t, box)}, 0.5, ops.ShellInside,
		[]ops.ShellFaceThickness{{FaceKey: side, Thickness: 0}}); err == nil {
		t.Error("a zero face thickness should be refused")
	}
	if _, err := ops.ShellVaried(box, [][]byte{topFaceKey(t, box)}, 0.5, ops.ShellInside,
		[]ops.ShellFaceThickness{{FaceKey: []byte("gone"), Thickness: 1}}); err == nil {
		t.Error("a lost face key in an override should be refused")
	}
}

// TestShellVariedWithNoOverridesIsTheUniformShell: the default path must be untouched, or every
// existing shell in every existing document changes.
func TestShellVariedWithNoOverridesIsTheUniformShell(t *testing.T) {
	box := shellBox(4, 4, 4)
	got := shellVolume(t, box, [][]byte{topFaceKey(t, box)}, 0.5, ops.ShellInside, nil)
	if want := 64.0 - 3*3*3.5; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("uniform shell through ShellVaried = %g, want %g", got, want)
	}
}
