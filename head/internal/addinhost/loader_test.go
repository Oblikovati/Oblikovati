//go:build linux || darwin

// SPDX-License-Identifier: GPL-2.0-only

package addinhost

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Oblikovati/oblikovati/addin/dispatch"
)

// buildFixture compiles the echo test fixture (testdata/echoaddin) as a c-shared
// library into a temp dir and returns its path. It uses GOTOOLCHAIN=local so the
// build never reaches the network, and GOWORK=off so the fixture builds against its
// own go.mod — it is a separate module standing in for an external add-in, and the
// repo's go.work workspace (which does not `use` it) must not capture it.
func buildFixture(t *testing.T, dirName string) string {
	t.Helper()
	ext := ".so"
	if runtime.GOOS == "darwin" {
		ext = ".dylib"
	}
	out := filepath.Join(t.TempDir(), dirName+ext)
	_, thisFile, _, _ := runtime.Caller(0)
	src := filepath.Join(filepath.Dir(thisFile), "testdata", dirName)

	cmd := exec.Command("go", "build", "-buildmode=c-shared", "-o", out, ".")
	cmd.Dir = src
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1", "GOTOOLCHAIN=local", "GOWORK=off")
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fixture add-in %q: %v\n%s", dirName, err, combined)
	}
	return out
}

// TestLoadDirRoundTripsHostCall is the seam proof: load a real c-shared add-in,
// activate it, and confirm its add-in->host->add-in echo round-trips through the
// Dispatcher. Activate blocks on the host-call, so we drain concurrently.
func TestLoadDirRoundTripsHostCall(t *testing.T) {
	so := buildFixture(t, "echoaddin")
	dir := filepath.Dir(so)

	d := dispatch.New(8)
	defer d.Close()
	SetHost(d, func(method string, req []byte) ([]byte, error) {
		if method != "echo" {
			return nil, fmt.Errorf("unexpected method %q", method)
		}
		return req, nil
	}, 2*time.Second)

	libs, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	if len(libs) != 1 {
		t.Fatalf("loaded %d add-ins, want 1", len(libs))
	}
	if got := libs[0].ID(); got != "com.oblikovati.echo-fixture" {
		t.Fatalf("ID = %q, want com.oblikovati.echo-fixture", got)
	}
	defer func() {
		if err := libs[0].Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	errc := make(chan error, 1)
	go func() { errc <- libs[0].Activate(nil) }()

	deadline := time.After(5 * time.Second)
	for {
		d.Drain(0)
		select {
		case err := <-errc:
			if err != nil {
				t.Fatalf("Activate (host-call round-trip failed): %v", err)
			}
			return
		case <-deadline:
			t.Fatal("Activate did not complete: host-call seam stuck")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

// TestLoadDirMissingDir treats an absent add-in folder as "none installed".
func TestLoadDirMissingDir(t *testing.T) {
	libs, err := LoadDir(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("LoadDir(missing) err = %v, want nil", err)
	}
	if libs != nil {
		t.Fatalf("LoadDir(missing) = %v, want nil", libs)
	}
}
