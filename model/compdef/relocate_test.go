// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/persistence"
)

// moveFile relocates a saved .obk package from src to dst (creating dst's directory),
// simulating a project tree moved as a whole on disk.
func moveFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %q: %v", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", filepath.Dir(dst), err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write %q: %v", dst, err)
	}
	if err := os.Remove(src); err != nil {
		t.Fatalf("remove %q: %v", src, err)
	}
}

// TestAssemblyReopensAfterTreeMove saves an assembly that references a component in a
// subfolder, moves the whole tree (assembly + subfolder) to a new location, and reopens
// the assembly there — its occurrence must re-resolve through the owner-relative reference
// record, rebinding to the relocated component without manual repair (#750).
func TestAssemblyReopensAfterTreeMove(t *testing.T) {
	store := persistence.NewPackageStore()
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "parts"), 0o755); err != nil {
		t.Fatalf("mkdir parts: %v", err)
	}
	ws := doc.NewWorkspace(store)
	widget := savePartDoc(t, ws, filepath.Join(src, "parts"), "widget.obk")
	asm, asmDef := newAssembly(t, ws, src, "asm.obk")
	placeFromFile(t, asm, widget, asmDef, "widget:1", math.Identity4())
	if err := ws.Save(asm); err != nil {
		t.Fatalf("Save assembly: %v", err)
	}

	// Move the assembly and its parts/ subfolder to a fresh location.
	dst := t.TempDir()
	moveFile(t, filepath.Join(src, "asm.obk"), filepath.Join(dst, "asm.obk"))
	moveFile(t, filepath.Join(src, "parts", "widget.obk"), filepath.Join(dst, "parts", "widget.obk"))

	def := openAssembly(t, store, filepath.Join(dst, "asm.obk"))
	if def.Occurrences().Count() != 1 {
		t.Fatalf("reopened assembly has %d occurrences, want 1", def.Occurrences().Count())
	}
	occ := def.Occurrences().Item(0)
	if _, ok := occ.Definition().(*compdef.PartComponentDefinition); !ok {
		t.Fatalf("occurrence definition = %T, want the relocated widget part (re-anchored, not a missing placeholder)", occ.Definition())
	}
	if want := filepath.Join(dst, "parts", "widget.obk"); occ.ComponentName() != want {
		t.Errorf("occurrence component name = %q, want re-anchored %q", occ.ComponentName(), want)
	}
}
