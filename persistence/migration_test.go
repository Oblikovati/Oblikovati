// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestOlderVersionFileOpensAndMigrates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.obk")

	// Hand-build a v0 package: old parameter stream name, schema version 0.
	legacy := NewPackage()
	if err := legacy.SetManifest(Manifest{SchemaVersion: 0, DocumentType: 1, DisplayName: "Legacy"}); err != nil {
		t.Fatalf("SetManifest: %v", err)
	}
	legacy.WriteStream("model/params.bin", []byte("param-bytes"))
	if err := legacy.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	upgraded, err := OpenPackage(path) // OpenPackage migrates
	if err != nil {
		t.Fatalf("OpenPackage: %v", err)
	}
	m, _ := upgraded.Manifest()
	if m.SchemaVersion != CurrentSchemaVersion {
		t.Errorf("schema after migrate = %d, want %d", m.SchemaVersion, CurrentSchemaVersion)
	}
	if _, ok := upgraded.ReadStream("model/params.bin"); ok {
		t.Error("legacy stream name still present after migration")
	}
	got, ok := upgraded.ReadStream("model/parameters.bin")
	if !ok || !bytes.Equal(got, []byte("param-bytes")) {
		t.Errorf("migrated stream = %q ok=%v, want carried-over bytes (no data loss)", got, ok)
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
