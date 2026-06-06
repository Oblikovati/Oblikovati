//go:build linux || darwin

// SPDX-License-Identifier: GPL-2.0-only

package addinhost

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"oblikovati/addin/dispatch"
)

// TestOpenReloadCycle proves the hot-reload core: a single library can be Open'd,
// activated, closed (dlclose), then Open'd again from the SAME path and reactivated.
// dlopen returns the running image while a handle is held, so reload depends on the
// close fully releasing it before the re-open — exactly what this exercises.
func TestOpenReloadCycle(t *testing.T) {
	so := buildFixture(t, "echoaddin")

	d := dispatch.New(8)
	defer d.Close()
	SetHost(d, func(method string, req []byte) ([]byte, error) {
		if method != "echo" {
			return nil, fmt.Errorf("unexpected method %q", method)
		}
		return req, nil
	}, 2*time.Second)

	for i := 0; i < 2; i++ {
		lib, err := Open(so)
		if err != nil {
			t.Fatalf("Open (cycle %d): %v", i, err)
		}
		if lib.Path() != so {
			t.Errorf("Path = %q, want %q", lib.Path(), so)
		}
		if lib.ID() != "com.oblikovati.echo-fixture" {
			t.Fatalf("ID = %q, want com.oblikovati.echo-fixture", lib.ID())
		}
		activateWithDrain(t, d, lib)
		if err := lib.Deactivate(nil); err != nil {
			t.Fatalf("Deactivate (cycle %d): %v", i, err)
		}
		if err := lib.Close(); err != nil {
			t.Fatalf("Close (cycle %d): %v", i, err)
		}
	}
}

// TestOpenCopiesCoexistWithoutClose mirrors the host's crash-safe hot-reload: load
// the add-in from one temp copy, activate then deactivate it (leaving it resident,
// NOT dlclosed), and load + activate a SECOND temp copy. Both instances coexist —
// the strategy the host uses because a Go c-shared cannot be safely dlclosed.
func TestOpenCopiesCoexistWithoutClose(t *testing.T) {
	so := buildFixture(t, "echoaddin")

	d := dispatch.New(8)
	defer d.Close()
	SetHost(d, func(method string, req []byte) ([]byte, error) {
		if method != "echo" {
			return nil, fmt.Errorf("unexpected method %q", method)
		}
		return req, nil
	}, 2*time.Second)

	first, err := Open(copyTo(t, so))
	if err != nil {
		t.Fatalf("Open first copy: %v", err)
	}
	activateWithDrain(t, d, first)
	if err := first.Deactivate(nil); err != nil {
		t.Fatalf("Deactivate first: %v", err)
	}
	// Do NOT close first; load and activate a second copy alongside it.
	second, err := Open(copyTo(t, so))
	if err != nil {
		t.Fatalf("Open second copy: %v", err)
	}
	activateWithDrain(t, d, second)
}

// copyTo copies src to a unique temp file and returns its path.
func copyTo(t *testing.T, src string) string {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %q: %v", src, err)
	}
	dst := filepath.Join(t.TempDir(), fmt.Sprintf("copy-%d.so", time.Now().UnixNano()))
	if err := os.WriteFile(dst, data, 0o755); err != nil {
		t.Fatalf("write %q: %v", dst, err)
	}
	return dst
}

// TestOpenMissing reports a clear error for a non-existent library path.
func TestOpenMissing(t *testing.T) {
	if _, err := Open(filepath.Join(t.TempDir(), "nope.so")); err == nil {
		t.Fatal("Open of a missing file should error")
	}
}

// activateWithDrain runs Activate while draining the dispatcher, since the fixture's
// Activate blocks on a host call that the session goroutine must service.
func activateWithDrain(t *testing.T, d *dispatch.Dispatcher, lib *LoadedAddIn) {
	t.Helper()
	errc := make(chan error, 1)
	go func() { errc <- lib.Activate(nil) }()
	deadline := time.After(5 * time.Second)
	for {
		d.Drain(0)
		select {
		case err := <-errc:
			if err != nil {
				t.Fatalf("Activate: %v", err)
			}
			return
		case <-deadline:
			t.Fatal("Activate did not complete: host-call seam stuck")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}
