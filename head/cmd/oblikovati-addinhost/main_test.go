//go:build linux || darwin || windows

// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"oblikovati.org/addin/dispatch"
	"oblikovati.org/api/wire"
	"oblikovati.org/head/internal/addinhost"
)

const echoFixtureID = "com.oblikovati.echo-fixture"

// TestNewHostSessionRegistersCommands proves the session baseline builds without error —
// a file-backed store plus the standard command set, the same baseline the head starts from.
func TestNewHostSessionRegistersCommands(t *testing.T) {
	session, err := newHostSession()
	if err != nil {
		t.Fatalf("newHostSession: %v", err)
	}
	if session == nil {
		t.Fatal("newHostSession returned a nil session")
	}
}

// TestNewRouterHandlerRoutesReadMethod checks the router handler wired over the session
// answers a real read method (application.apiVersion) and rejects an unknown one — proving
// the add-in host call reaches the router, not a stub.
func TestNewRouterHandlerRoutesReadMethod(t *testing.T) {
	session, err := newHostSession()
	if err != nil {
		t.Fatalf("newHostSession: %v", err)
	}
	handle := newRouterHandler(session)

	out, err := handle(wire.MethodApplicationApiVersion, []byte("{}"))
	if err != nil {
		t.Fatalf("handle(%s): %v", wire.MethodApplicationApiVersion, err)
	}
	var got wire.ApplicationApiVersionResult
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal apiVersion result %s: %v", out, err)
	}
	if got.Version == "" {
		t.Errorf("apiVersion result = %s, want a non-empty version", out)
	}
	if _, err := handle("no.such.method", []byte("{}")); err == nil {
		t.Error("handle(no.such.method) err = nil, want an unknown-method error")
	}
}

// TestInstallHostWiresRouter runs installHost (the C-ABI host wiring) and confirms the
// same router handler it installs routes a real read method.
func TestInstallHostWiresRouter(t *testing.T) {
	session, err := newHostSession()
	if err != nil {
		t.Fatalf("newHostSession: %v", err)
	}
	d := dispatch.New(dispatchCapacity)
	defer d.Close()
	installHost(session, d)

	if _, err := newRouterHandler(session)(wire.MethodApplicationApiVersion, []byte("{}")); err != nil {
		t.Fatalf("router handler after installHost: %v", err)
	}
}

// TestDrainTickRunsQueuedCall submits one job from another goroutine and confirms a single
// drainTick runs it — the loop body runDrainLoop repeats on every ticker tick.
func TestDrainTickRunsQueuedCall(t *testing.T) {
	d := dispatch.New(dispatchCapacity)
	defer d.Close()
	go func() {
		_, _ = d.Submit(context.Background(), func() ([]byte, error) { return []byte("ok"), nil })
	}()

	deadline := time.After(2 * time.Second)
	for {
		if drainTick(d) == 1 {
			return
		}
		select {
		case <-deadline:
			t.Fatal("drainTick never ran the queued job")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

// TestDrainUntilDrainsThenStops proves the ticking loop drains a queued call and then
// returns when its stop channel delivers a signal — the whole of runDrainLoop except the
// os/signal registration.
func TestDrainUntilDrainsThenStops(t *testing.T) {
	d := dispatch.New(dispatchCapacity)
	defer d.Close()
	stop := make(chan os.Signal, 1)
	ran := make(chan []byte, 1)
	go func() {
		out, _ := d.Submit(context.Background(), func() ([]byte, error) { return []byte("x"), nil })
		ran <- out              // Submit returns only after the loop drained the job...
		stop <- syscall.SIGTERM // ...so asking to stop now cannot race ahead of the drain.
	}()

	drainUntil(d, stop)
	select {
	case <-ran:
	default:
		t.Fatal("drainUntil returned before draining the queued job")
	}
}

// TestAddInsDirUsesEnv checks OBK_ADDINS_DIR overrides the default location.
func TestAddInsDirUsesEnv(t *testing.T) {
	t.Setenv("OBK_ADDINS_DIR", filepath.Join("some", "addins"))
	if got := addInsDir(); got != filepath.Join("some", "addins") {
		t.Errorf("addInsDir() = %q, want the OBK_ADDINS_DIR value", got)
	}
}

// TestAddInsDirDefaultsBesideExe checks the fallback is an "addins" folder beside the exe
// when OBK_ADDINS_DIR is unset.
func TestAddInsDirDefaultsBesideExe(t *testing.T) {
	t.Setenv("OBK_ADDINS_DIR", "")
	if got := addInsDir(); filepath.Base(got) != "addins" {
		t.Errorf("addInsDir() = %q, want a path ending in \"addins\"", got)
	}
}

// TestStartAddInsActivatesFixture is the end-to-end proof without the real bridge: it
// loads the echo c-shared fixture, and startAddIns registers + activates it, round-tripping
// the fixture's Activate host call through the dispatcher (drained concurrently, since
// Activate blocks). It returns the activated id.
func TestStartAddInsActivatesFixture(t *testing.T) {
	dir := fixtureDir(t)
	buildFixtureInto(t, "echoaddin", dir)

	session, err := newHostSession()
	if err != nil {
		t.Fatalf("newHostSession: %v", err)
	}
	d := dispatch.New(dispatchCapacity)
	defer d.Close()
	// The fixture's Activate calls the host "echo" method and expects "ping" back.
	addinhost.SetHost(d, func(method string, req []byte) ([]byte, error) {
		if method != "echo" {
			return nil, fmt.Errorf("unexpected host method %q", method)
		}
		return req, nil
	}, 2*time.Second)

	ids := drainWhileActivating(t, d, func() []string { return startAddIns(session, dir) })
	if len(ids) != 1 || ids[0] != echoFixtureID {
		t.Fatalf("startAddIns activated %v, want [%s]", ids, echoFixtureID)
	}
}

// TestStartAddInsReportsSkippedAndFailed drives both non-happy paths: the incompatible
// fixture is skipped at load time, and the echo fixture's Activate fails because the host
// rejects its call — so startAddIns activates nothing and both are surfaced (not fatal).
func TestStartAddInsReportsSkippedAndFailed(t *testing.T) {
	dir := fixtureDir(t)
	buildFixtureInto(t, "echoaddin", dir)
	buildFixtureInto(t, "incompataddin", dir)

	session, err := newHostSession()
	if err != nil {
		t.Fatalf("newHostSession: %v", err)
	}
	d := dispatch.New(dispatchCapacity)
	defer d.Close()
	// Reject every host call, so the echo fixture's Activate returns an error → a failure.
	addinhost.SetHost(d, func(method string, _ []byte) ([]byte, error) {
		return nil, fmt.Errorf("host refuses %q", method)
	}, 2*time.Second)

	ids := drainWhileActivating(t, d, func() []string { return startAddIns(session, dir) })
	if len(ids) != 0 {
		t.Fatalf("startAddIns activated %v, want none (echo must fail, incompatible must skip)", ids)
	}
}

// TestStartAddInsLoadError surfaces a hard load error: a path that exists but is a file,
// not a directory, makes LoadDir's os.ReadDir fail with a non-"not exist" error, which
// startAddIns reports while activating nothing.
func TestStartAddInsLoadError(t *testing.T) {
	notADir := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(notADir, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	session, err := newHostSession()
	if err != nil {
		t.Fatalf("newHostSession: %v", err)
	}
	if ids := startAddIns(session, notADir); len(ids) != 0 {
		t.Fatalf("startAddIns(file) = %v, want none", ids)
	}
}

// TestRegisterAndActivateRejectsDuplicate covers the register-failure path: registering an
// add-in whose id is already registered fails before activation, and the error names the id.
func TestRegisterAndActivateRejectsDuplicate(t *testing.T) {
	dir := fixtureDir(t)
	buildFixtureInto(t, "echoaddin", dir)
	libs, _, err := addinhost.LoadDir(dir)
	if err != nil || len(libs) != 1 {
		t.Fatalf("LoadDir: libs=%d err=%v", len(libs), err)
	}
	lib := libs[0]
	defer func() { _ = lib.Close() }()

	session, err := newHostSession()
	if err != nil {
		t.Fatalf("newHostSession: %v", err)
	}
	if err := session.AddIns().Register(lib); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	if err := registerAndActivate(session, lib); err == nil {
		t.Fatal("registerAndActivate on an already-registered id: err = nil, want a duplicate error")
	}
}

// TestStartAddInsMissingDir treats an absent add-ins directory as "none installed": no
// error, nothing activated.
func TestStartAddInsMissingDir(t *testing.T) {
	session, err := newHostSession()
	if err != nil {
		t.Fatalf("newHostSession: %v", err)
	}
	if ids := startAddIns(session, filepath.Join(t.TempDir(), "does-not-exist")); len(ids) != 0 {
		t.Fatalf("startAddIns(missing) = %v, want none", ids)
	}
}

// drainWhileActivating runs activate on a background goroutine and drains d on this one
// until it returns, because a fixture's Activate blocks on a host call the drain services.
func drainWhileActivating(t *testing.T, d *dispatch.Dispatcher, activate func() []string) []string {
	t.Helper()
	done := make(chan []string, 1)
	go func() { done <- activate() }()
	deadline := time.After(20 * time.Second)
	for {
		d.Drain(0)
		select {
		case ids := <-done:
			return ids
		case <-deadline:
			t.Fatal("timed out draining while startAddIns activated add-ins")
			return nil
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

// buildFixtureInto compiles a testdata add-in fixture (shared by the addinhost package
// tests) as a c-shared library into destDir. GOTOOLCHAIN=local keeps the build offline and
// GOWORK=off builds the fixture against its own module, standing in for an external add-in.
// Go test helpers cannot cross package/test-binary boundaries, so this mirrors the addinhost
// package's buildFixture (sonar.cpd.exclusions covers _test.go for that reason).
func buildFixtureInto(t *testing.T, name, destDir string) {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	src := filepath.Join(filepath.Dir(thisFile), "..", "..", "internal", "addinhost", "testdata", name)
	out := filepath.Join(destDir, name+sharedLibExt())

	cmd := exec.Command("go", "build", "-buildmode=c-shared", "-o", out, ".")
	cmd.Dir = src
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1", "GOTOOLCHAIN=local", "GOWORK=off")
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fixture %q: %v\n%s", name, err, combined)
	}
}

// sharedLibExt is the c-shared extension isSharedLib recognizes on this OS.
func sharedLibExt() string {
	switch runtime.GOOS {
	case "windows":
		return ".dll"
	case "darwin":
		return ".dylib"
	default:
		return ".so"
	}
}

// fixtureDir makes a temp directory for c-shared fixtures with BEST-EFFORT cleanup: on
// Windows a loaded add-in DLL stays resident for the process lifetime and the image file
// stays locked, so a strict RemoveAll would fail — the OS reclaims it at process exit.
func fixtureDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "addinhost-cmd-fixture-")
	if err != nil {
		t.Fatalf("make fixture temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}
