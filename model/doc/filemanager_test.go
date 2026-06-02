// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("seed %q: %v", path, err)
	}
}

func TestResolveSearchesWorkspaceThenLibraries(t *testing.T) {
	ws := t.TempDir()
	lib := t.TempDir()
	writeFile(t, filepath.Join(lib, "fastener.obk"))
	fm := NewFileManager(&DesignProject{WorkspacePath: ws, LibraryPaths: []string{lib}})

	got, ok := fm.Resolve("fastener.obk")
	if !ok || got != filepath.Join(lib, "fastener.obk") {
		t.Fatalf("Resolve via library = %q ok=%v, want the library copy", got, ok)
	}

	// A copy in the workspace must win over the library copy.
	writeFile(t, filepath.Join(ws, "fastener.obk"))
	got, _ = fm.Resolve("fastener.obk")
	if got != filepath.Join(ws, "fastener.obk") {
		t.Errorf("Resolve = %q, want the workspace copy to take priority", got)
	}
}

func TestResolveMissingIsBrokenNotFatal(t *testing.T) {
	fm := NewFileManager(&DesignProject{WorkspacePath: t.TempDir()})
	if _, ok := fm.Resolve("nope.obk"); ok {
		t.Error("Resolve found a file that does not exist")
	}
}

func TestResolveAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	abs := filepath.Join(dir, "thing.obk")
	writeFile(t, abs)
	fm := NewFileManager(&DesignProject{WorkspacePath: dir})
	if got, ok := fm.Resolve(abs); !ok || got != abs {
		t.Errorf("Resolve(abs) = %q ok=%v, want it returned as-is", got, ok)
	}
	if _, ok := fm.Resolve(filepath.Join(dir, "absent.obk")); ok {
		t.Error("Resolve(abs) reported a missing absolute path as found")
	}
}

func TestTemplateFileByType(t *testing.T) {
	templates := t.TempDir()
	writeFile(t, filepath.Join(templates, "part.obk"))
	writeFile(t, filepath.Join(templates, "assembly.obk"))
	fm := NewFileManager(&DesignProject{Locations: FileLocations{Templates: templates}})

	if got, ok := fm.TemplateFile(Part); !ok || got != filepath.Join(templates, "part.obk") {
		t.Errorf("TemplateFile(Part) = %q ok=%v", got, ok)
	}
	if _, ok := fm.TemplateFile(Drawing); ok {
		t.Error("TemplateFile(Drawing) found a nonexistent template")
	}
	if _, ok := fm.TemplateFile(Unknown); ok {
		t.Error("TemplateFile(Unknown) should never resolve")
	}
}

func TestRelativizeWithinWorkspace(t *testing.T) {
	ws := t.TempDir()
	fm := NewFileManager(&DesignProject{WorkspacePath: ws})
	rel, ok := fm.Relativize(filepath.Join(ws, "sub", "part.obk"))
	if !ok || rel != filepath.Join("sub", "part.obk") {
		t.Errorf("Relativize = %q ok=%v, want sub/part.obk", rel, ok)
	}
	if _, ok := fm.Relativize(filepath.Join(t.TempDir(), "outside.obk")); ok {
		t.Error("Relativize accepted a path outside the workspace")
	}
}

func TestDesignProjectManager(t *testing.T) {
	m := NewDesignProjectManager()
	if m.ActiveProject() != nil {
		t.Error("empty manager has an active project")
	}
	a := &DesignProject{Name: "A"}
	b := &DesignProject{Name: "B"}
	m.Add(a)
	m.Add(b)
	if m.ActiveProject() != a {
		t.Error("first added project should be active")
	}
	if len(m.Projects()) != 2 {
		t.Errorf("Projects len = %d, want 2", len(m.Projects()))
	}
	if err := m.SetActiveProject(b); err != nil {
		t.Fatalf("SetActiveProject: %v", err)
	}
	if m.ActiveProject() != b {
		t.Error("SetActiveProject did not switch the active project")
	}
	if err := m.SetActiveProject(&DesignProject{Name: "C"}); err == nil {
		t.Error("SetActiveProject accepted an unregistered project")
	}
}

func TestSearchPathsOrder(t *testing.T) {
	p := &DesignProject{WorkspacePath: "/ws", LibraryPaths: []string{"/lib1", "/lib2"}}
	got := p.SearchPaths()
	want := []string{"/ws", "/lib1", "/lib2"}
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("SearchPaths = %v, want %v", got, want)
	}
}
