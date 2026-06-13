// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oblikovati.org/model/doc"
)

// sampleProject is a fully-populated design project used to pin the .opj round-trip.
func sampleProject() *doc.DesignProject {
	return &doc.DesignProject{
		Name:          "Acme Gearbox",
		WorkspacePath: "/work/acme",
		LibraryPaths:  []string{"/lib/fasteners", "/lib/bearings"},
		Locations:     doc.FileLocations{Templates: "/work/acme/templates", DesignData: "/work/acme/design"},
	}
}

func TestProjectFileRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "acme.opj")
	want := sampleProject()
	if err := WriteProjectFile(want, path); err != nil {
		t.Fatalf("WriteProjectFile: %v", err)
	}
	got, err := ReadProjectFile(path)
	if err != nil {
		t.Fatalf("ReadProjectFile: %v", err)
	}
	if got.Name != want.Name || got.WorkspacePath != want.WorkspacePath {
		t.Errorf("identity = {%q %q}, want {%q %q}", got.Name, got.WorkspacePath, want.Name, want.WorkspacePath)
	}
	if len(got.LibraryPaths) != 2 || got.LibraryPaths[0] != "/lib/fasteners" {
		t.Errorf("libraries = %v, want the two roots in order", got.LibraryPaths)
	}
	if got.Locations != want.Locations {
		t.Errorf("locations = %+v, want %+v", got.Locations, want.Locations)
	}
}

// TestProjectFileIsReadableYAML guards the git-friendly text shape (ADR-0020): the
// .opj is plain YAML naming the project, not a binary blob.
func TestProjectFileIsReadableYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "acme.opj")
	if err := WriteProjectFile(sampleProject(), path); err != nil {
		t.Fatalf("WriteProjectFile: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	for _, want := range []string{"schemaVersion: 1", "name: Acme Gearbox", "workspace: /work/acme"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("project YAML missing %q; got:\n%s", want, raw)
		}
	}
}

func TestProjectFileRejectsWrongExtension(t *testing.T) {
	dir := t.TempDir()
	// A document extension must be refused on both write and read.
	if err := WriteProjectFile(sampleProject(), filepath.Join(dir, "acme.opd")); err == nil {
		t.Error("WriteProjectFile(.opd) = nil, want a wrong-extension error")
	}
	if _, err := ReadProjectFile(filepath.Join(dir, "acme.opd")); err == nil {
		t.Error("ReadProjectFile(.opd) = nil, want a wrong-extension error")
	}
}

func TestProjectFileRejectsUnknownSchemaVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "future.opj")
	if err := os.WriteFile(path, []byte("schemaVersion: 999\nname: future\n"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := ReadProjectFile(path); err == nil {
		t.Error("ReadProjectFile(schemaVersion 999) = nil, want an unknown-version error")
	}
}
