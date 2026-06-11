// SPDX-License-Identifier: GPL-2.0-only

package addinstate

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/api/types"
)

func TestLoadMissingFileIsEmpty(t *testing.T) {
	s := NewFileStore(filepath.Join(t.TempDir(), "addin-behaviors.yaml"))
	m, err := s.Load()
	if err != nil || len(m) != 0 {
		t.Fatalf("Load(missing) = (%v, %v), want empty map, nil", m, err)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "addin-behaviors.yaml")
	s := NewFileStore(path)
	in := map[string]types.AddInLoadBehavior{
		"com.x.a": types.LoadOnDemand,
		"com.x.b": types.LoadDisabled,
	}
	if err := s.Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	out, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(out) != 2 || out["com.x.a"] != types.LoadOnDemand || out["com.x.b"] != types.LoadDisabled {
		t.Errorf("round trip = %v, want %v", out, in)
	}
}

func TestLoadSkipsUnknownBehaviorName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "addin-behaviors.yaml")
	yaml := "behaviors:\n  com.x.a: sometimes\n  com.x.b: demand\n"
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := NewFileStore(path).Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, has := out["com.x.a"]; has {
		t.Error("a typo'd behavior name should fall back to the default, not load")
	}
	if out["com.x.b"] != types.LoadOnDemand {
		t.Errorf("com.x.b = %v, want demand", out["com.x.b"])
	}
}
