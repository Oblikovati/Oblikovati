// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"path/filepath"
	"testing"
)

func TestCurrentVersionFileNeedsNoMigration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "current.obk")

	saved := NewPackage()
	if err := saved.SetManifest(Manifest{SchemaVersion: CurrentSchemaVersion, DocumentType: 1, DisplayName: "Bracket"}); err != nil {
		t.Fatalf("SetManifest: %v", err)
	}
	if err := saved.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reopened, err := OpenPackage(path) // OpenPackage runs Migrate (a no-op at the current version)
	if err != nil {
		t.Fatalf("OpenPackage: %v", err)
	}
	m, _ := reopened.Manifest()
	if m.SchemaVersion != CurrentSchemaVersion || m.DisplayName != "Bracket" {
		t.Errorf("reopened manifest = %+v, want v%d Bracket", m, CurrentSchemaVersion)
	}
}

func TestNewerVersionFileRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "future.obk")
	future := NewPackage()
	_ = future.SetManifest(Manifest{SchemaVersion: CurrentSchemaVersion + 1, DocumentType: 1})
	if err := future.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := OpenPackage(path); err == nil {
		t.Error("OpenPackage accepted a package from a newer schema version")
	}
}

func TestMigrateNoManifestIsNoOp(t *testing.T) {
	p := packageWith(map[string][]byte{"client.bin": {1, 2, 3}})
	if err := Migrate(p); err != nil {
		t.Errorf("Migrate on a manifest-less container errored: %v", err)
	}
}
