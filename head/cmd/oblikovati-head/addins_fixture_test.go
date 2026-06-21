//go:build linux || darwin

// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
)

// fakeBehaviorStore is a named in-memory AddInBehaviorStore so a test can mark an add-in
// LoadDisabled — keeping registerResults' register path off the activation host-call seam.
type fakeBehaviorStore struct {
	stored map[string]types.AddInLoadBehavior
}

func (f *fakeBehaviorStore) Load() (map[string]types.AddInLoadBehavior, error) {
	return f.stored, nil
}
func (f *fakeBehaviorStore) Save(map[string]types.AddInLoadBehavior) error { return nil }

// buildEchoFixtureInto compiles the echo test fixture (shared by the addinhost tests) as a
// c-shared library directly into dir, returning its path. GOWORK=off builds it against its
// own go.mod (it stands in for an external add-in); GOTOOLCHAIN=local keeps it offline.
func buildEchoFixtureInto(t *testing.T, dir string) string {
	t.Helper()
	ext := ".so"
	if runtime.GOOS == "darwin" {
		ext = ".dylib"
	}
	out := filepath.Join(dir, "echo"+ext)
	_, thisFile, _, _ := runtime.Caller(0)
	src := filepath.Join(filepath.Dir(thisFile), "..", "..", "internal", "addinhost", "testdata", "echoaddin")

	cmd := exec.Command("go", "build", "-buildmode=c-shared", "-o", out, ".")
	cmd.Dir = src
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1", "GOTOOLCHAIN=local", "GOWORK=off")
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build echo fixture: %v\n%s", err, combined)
	}
	return out
}

// TestLoadInstalledAddInsRegistersFixture drives the per-user install path end to end: a real
// c-shared add-in laid out as <user>/<name>/<lib> is found, loaded and registered. It is
// marked LoadDisabled so registration does not activate (which would need the host-call
// dispatcher), letting the test cover the register-and-append path directly.
func TestLoadInstalledAddInsRegistersFixture(t *testing.T) {
	root := t.TempDir()
	addInDir := filepath.Join(root, "com.oblikovati.echo-fixture")
	if err := os.MkdirAll(addInDir, 0o755); err != nil {
		t.Fatal(err)
	}
	buildEchoFixtureInto(t, addInDir)
	t.Setenv("OBK_USER_ADDINS_DIR", root)

	session := app.NewSession()
	if err := session.AddIns().UseBehaviorStore(&fakeBehaviorStore{
		stored: map[string]types.AddInLoadBehavior{"com.oblikovati.echo-fixture": types.LoadDisabled},
	}); err != nil {
		t.Fatalf("UseBehaviorStore: %v", err)
	}

	h := &addInHost{}
	h.loadInstalledAddIns(session)

	if len(h.loaded) != 1 {
		t.Fatalf("loaded %d add-ins, want 1", len(h.loaded))
	}
	defer func() { _ = h.loaded[0].Close() }()
	if got := h.loaded[0].ID(); got != "com.oblikovati.echo-fixture" {
		t.Errorf("ID = %q, want com.oblikovati.echo-fixture", got)
	}
}
