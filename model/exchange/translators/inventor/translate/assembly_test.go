// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"math"
	"path/filepath"
	"strings"
	"testing"

	m "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/occurrence"
	"oblikovati.org/persistence"
)

// TestAssemblyTranslationPlacesOccurrences translates the two-box assembly (a box placed at
// the origin and at (5,0,0) cm) and checks the reopened assembly has both occurrences at
// their decoded placements — the end-to-end assembly path (node-graph transforms +
// recursive component translation + persistent placement).
func TestAssemblyTranslationPlacesOccurrences(t *testing.T) {
	out := filepath.Join(t.TempDir(), "asm.opd")
	iamPath := filepath.Join("..", "testdata", "asm_two_boxes.iam")
	warns, err := AssemblyFromInventor(iamPath, out)
	if err != nil {
		t.Fatalf("AssemblyFromInventor: %v (warns %v)", err, warns)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	asm := reopened.Content().(*compdef.AssemblyComponentDefinition)
	occs := asm.Occurrences().All()
	if len(occs) != 2 {
		t.Fatalf("reopened assembly has %d occurrences, want 2", len(occs))
	}
	var haveOrigin, haveOffset bool
	for _, o := range occs {
		tr := o.Transform().Translation()
		x, y, z := float64(tr.X), float64(tr.Y), float64(tr.Z)
		switch {
		case x == 0 && y == 0 && z == 0:
			haveOrigin = true
		case math.Abs(x-5) < 1e-6 && y == 0 && z == 0:
			haveOffset = true
		default:
			t.Errorf("occurrence %q unexpected translation (%.3f,%.3f,%.3f)", o.Name(), x, y, z)
		}
	}
	if !haveOrigin || !haveOffset {
		t.Errorf("placements: origin=%v offset(5,0,0)=%v (want both)", haveOrigin, haveOffset)
	}
}

// TestAssemblyRotationRoundTrips translates the rotated-placement assembly (a bar rotated
// 90° about +Z then translated (7,0,0)) and checks the reopened occurrence carries the full
// rotation — proving the decoded row-major transform maps to the kernel transform without
// transposition. Verified both by the matrix cells and by where a component corner lands.
func TestAssemblyRotationRoundTrips(t *testing.T) {
	out := filepath.Join(t.TempDir(), "asm.opd")
	iamPath := filepath.Join("..", "testdata", "asm_rotated.iam")
	if _, err := AssemblyFromInventor(iamPath, out); err != nil {
		t.Fatalf("AssemblyFromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	asm := reopened.Content().(*compdef.AssemblyComponentDefinition)
	occ := occurrenceNamed(asm.Occurrences().All(), "asm_bar:2")
	if occ == nil {
		t.Fatal("reopened assembly has no occurrence asm_bar:2")
	}
	want := m.Matrix4FromCells([16]m.Scalar{
		0, -1, 0, 7,
		1, 0, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	})
	if !occ.Transform().IsEqualTo(want, 1e-9) {
		t.Errorf("occ asm_bar:2 transform = %v, want %v", occ.Transform().Cells(), want.Cells())
	}
	// Semantic check: the bar's (4,0,0) corner rotates to (0,4,0) then translates to (7,4,0).
	got := occ.Transform().TransformPoint(m.P3(4, 0, 0))
	if wantP := m.P3(7, 4, 0); !got.IsEqualTo(wantP, 1e-9) {
		t.Errorf("corner (4,0,0) placed at %v, want %v (rotation flipped?)", got, wantP)
	}
}

// TestConstrainedAssemblyPlacesCorrectly translates the mate-constrained assembly and
// checks the reopened result: two occurrences (the spurious constraint-selection name is
// filtered out), placed at their mate-solved positions (occ2 pulled to z=4), and a warning
// noting the constraint was placed by solved position but not rebuilt as a relationship.
func TestConstrainedAssemblyPlacesCorrectly(t *testing.T) {
	out := filepath.Join(t.TempDir(), "asm.opd")
	iamPath := filepath.Join("..", "testdata", "asm_mate.iam")
	warns, err := AssemblyFromInventor(iamPath, out)
	if err != nil {
		t.Fatalf("AssemblyFromInventor: %v", err)
	}
	var sawConstraint bool
	for _, w := range warns {
		if strings.Contains(w, "mate constraint") {
			sawConstraint = true
		}
	}
	if !sawConstraint {
		t.Errorf("expected a mate-constraint warning, got %v", warns)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	asm := reopened.Content().(*compdef.AssemblyComponentDefinition)
	occs := asm.Occurrences().All()
	if len(occs) != 2 {
		t.Fatalf("reopened assembly has %d occurrences, want 2 (spurious name filtered)", len(occs))
	}
	var haveOrigin, haveZ4 bool
	for _, o := range occs {
		z := float64(o.Transform().Translation().Z)
		switch {
		case z == 0:
			haveOrigin = true
		case math.Abs(z-4) < 1e-9:
			haveZ4 = true
		}
	}
	if !haveOrigin || !haveZ4 {
		t.Errorf("occurrence Z placements origin=%v z4=%v (want both)", haveOrigin, haveZ4)
	}
}

// TestSubAssemblyTranslationNests translates a two-level assembly (top places two
// sub-assemblies, each placing two leaf boxes) and checks the reopened structure recursed:
// top has two sub occurrences at (0,0,0)/(0,6,0), and each sub is itself an assembly with
// two leaf occurrences at (0,0,0)/(3,0,0).
func TestSubAssemblyTranslationNests(t *testing.T) {
	out := filepath.Join(t.TempDir(), "top.opd")
	iamPath := filepath.Join("..", "testdata", "top.iam")
	if _, err := AssemblyFromInventor(iamPath, out); err != nil {
		t.Fatalf("AssemblyFromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	top := reopened.Content().(*compdef.AssemblyComponentDefinition)
	subs := top.Occurrences().All()
	if len(subs) != 2 {
		t.Fatalf("top has %d occurrences, want 2 sub-assemblies", len(subs))
	}
	assertYs(t, "top", subs, 0, 6)
	for _, sub := range subs {
		def, ok := sub.Definition().(*compdef.AssemblyComponentDefinition)
		if !ok {
			t.Fatalf("occurrence %q Definition is %T, want a sub-assembly", sub.Name(), sub.Definition())
		}
		leaves := def.Occurrences().All()
		if len(leaves) != 2 {
			t.Fatalf("sub %q has %d occurrences, want 2 leaves", sub.Name(), len(leaves))
		}
		assertXs(t, sub.Name(), leaves, 0, 3)
	}
}

// assertYs checks the occurrences' Y translations are exactly {y0, y1} (order-independent).
func assertYs(t *testing.T, ctx string, occs []*occurrence.Occurrence, y0, y1 float64) {
	t.Helper()
	got := map[float64]bool{}
	for _, o := range occs {
		got[float64(o.Transform().Translation().Y)] = true
	}
	if !got[y0] || !got[y1] {
		t.Errorf("%s: Y translations = %v, want {%g,%g}", ctx, got, y0, y1)
	}
}

// assertXs checks the occurrences' X translations are exactly {x0, x1} (order-independent).
func assertXs(t *testing.T, ctx string, occs []*occurrence.Occurrence, x0, x1 float64) {
	t.Helper()
	got := map[float64]bool{}
	for _, o := range occs {
		got[float64(o.Transform().Translation().X)] = true
	}
	if !got[x0] || !got[x1] {
		t.Errorf("%s: X translations = %v, want {%g,%g}", ctx, got, x0, x1)
	}
}

func occurrenceNamed(occs []*occurrence.Occurrence, name string) *occurrence.Occurrence {
	for _, o := range occs {
		if o.Name() == name {
			return o
		}
	}
	return nil
}
