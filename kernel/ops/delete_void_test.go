// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// distinctCavityBody is cavityBody with the outer and inner boxes carrying DISTINCT lineages, so
// the two shells' faces have globally-unique reference keys (as real feature bodies do — the shared
// cavityBody fixture reuses one "box" token, which is fine for its volume tests but breaks key→shell
// lookup). 4³ block centred at origin minus a 2³ cavity centred at (1,1,1).
func distinctCavityBody(t *testing.T, outer, inner string) *topo.Body {
	t.Helper()
	box := func(p math.Point3, s float64, name string) *topo.Body {
		m := subd.Box(s, s, s)
		for i := range m.Verts {
			m.Verts[i] = m.Verts[i].TranslateBy(p.AsVector())
		}
		return subd.ToBody(m, name)
	}
	res, err := Boolean(Cut, box(math.P3(0, 0, 0), 4, outer), box(math.P3(1, 1, 1), 2, inner))
	if err != nil {
		t.Fatalf("cavity cut: %v", err)
	}
	return res
}

// voidFaceKey returns the reference key of one face on the body's internal void shell.
func voidFaceKey(t *testing.T, b *topo.Body) []byte {
	t.Helper()
	for _, sh := range b.Shells() {
		if ShellIsVoidInBody(b, sh) {
			return sh.Faces()[0].ReferenceKey()
		}
	}
	t.Fatal("body has no void shell")
	return nil
}

// outerFaceKey returns the reference key of one face on the body's outer (non-void) shell.
func outerFaceKey(t *testing.T, b *topo.Body) []byte {
	t.Helper()
	for _, sh := range b.Shells() {
		if !ShellIsVoidInBody(b, sh) {
			return sh.Faces()[0].ReferenceKey()
		}
	}
	t.Fatal("body has no outer shell")
	return nil
}

// TestRemoveVoidShellRestoresMass is the FaceShell arm of Delete Face (#1884): selecting a face on
// the 4³−2³ cavity's void shell removes the void, restoring the 64 solid volume.
func TestRemoveVoidShellRestoresMass(t *testing.T) {
	t.Parallel()
	body := distinctCavityBody(t, "outer", "inner")
	if v := BodyGeometryProperties(body, DefaultQuality()).Volume; stdmath.Abs(v-56) > 0.1 {
		t.Fatalf("fixture volume = %g, want 56 (4³ minus 2³ cavity)", v)
	}
	key := voidFaceKey(t, body)
	if !FacesOnVoidShell(body, [][]byte{key}, DefaultQuality()) {
		t.Fatal("a void-shell face should report FacesOnVoidShell true")
	}
	solid, err := RemoveVoidShellByFaces(body, [][]byte{key}, DefaultQuality())
	if err != nil {
		t.Fatalf("RemoveVoidShellByFaces: %v", err)
	}
	if got := len(solid.Shells()); got != 1 {
		t.Fatalf("result has %d shells, want 1 (void removed)", got)
	}
	if v := BodyGeometryProperties(solid, DefaultQuality()).Volume; stdmath.Abs(v-64) > 0.1 {
		t.Errorf("result volume = %g, want 64 (void filled)", v)
	}
	if r := Validate(solid); !r.Valid {
		t.Errorf("result invalid: %v", r.Issues)
	}
}

// TestFacesOnVoidShellRejectsOuterFace: an outer-shell face is not a void selection, so the feature
// routes to an ordinary face delete instead.
func TestFacesOnVoidShellRejectsOuterFace(t *testing.T) {
	t.Parallel()
	body := distinctCavityBody(t, "outer", "inner")
	if FacesOnVoidShell(body, [][]byte{outerFaceKey(t, body)}, DefaultQuality()) {
		t.Error("an outer-shell face should report FacesOnVoidShell false")
	}
	if FacesOnVoidShell(body, nil, DefaultQuality()) {
		t.Error("an empty selection should report FacesOnVoidShell false")
	}
}

// TestRemoveVoidShellErrorsOnNonVoid: RemoveVoidShellByFaces refuses an outer/lost face so the
// feature goes Sick rather than mangle the body.
func TestRemoveVoidShellErrorsOnNonVoid(t *testing.T) {
	t.Parallel()
	body := distinctCavityBody(t, "outer", "inner")
	if _, err := RemoveVoidShellByFaces(body, [][]byte{outerFaceKey(t, body)}, DefaultQuality()); err == nil {
		t.Error("removing a void by an outer-shell face should error")
	}
	if _, err := RemoveVoidShellByFaces(body, [][]byte{[]byte("ghost")}, DefaultQuality()); err == nil {
		t.Error("a lost face key should error")
	}
}

// TestDropFacesLeavesOpenSurface pins the heal=false arm at the kernel level: dropping a box face
// leaves an open (non-solid) surface body with one fewer face.
func TestDropFacesLeavesOpenSurface(t *testing.T) {
	t.Parallel()
	block, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(2, 2, 2), "b")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	key := block.Faces()[0].ReferenceKey()
	open, err := DropFaces(block, [][]byte{key}, false)
	if err != nil {
		t.Fatalf("DropFaces: %v", err)
	}
	if open.IsSolid() {
		t.Error("dropping a face should leave an open (non-solid) surface body")
	}
	if got, want := len(open.Faces()), len(block.Faces())-1; got != want {
		t.Errorf("open body has %d faces, want %d", got, want)
	}
}
