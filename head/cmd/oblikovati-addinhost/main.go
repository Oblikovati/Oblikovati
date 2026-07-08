// SPDX-License-Identifier: GPL-2.0-only

// Command oblikovati-addinhost is a headless (no Vulkan/GUI) host for shared-library
// add-ins. It replicates the add-in wiring of cmd/oblikovati-head/addins.go — install
// the router as the host-call handler, load and activate every add-in in a directory,
// and drain the dispatch queue on this single "session goroutine" forever — but without
// a renderer. It brings an add-in's endpoint up on a machine (or CI) that cannot build
// the cgo head, and proves the loader → c-shared → live-kernel chain end to end. Its
// intended use is a GUI-free bridge endpoint for CI, automation, and external clients
// such as the Inventor exporter.
//
// Environment:
//
//	OBK_ADDINS_DIR  directory holding each add-in's shared library (.dll/.so/.dylib) and
//	                its manifest.json. Defaults to an "addins" folder beside the exe.
//	OBK_MCP_ADDR    read by the MCP bridge add-in (not by this host) to choose the
//	                address it serves on, e.g. 127.0.0.1:7800.
//
// Example (serve the MCP bridge add-in, then drive it):
//
//	OBK_ADDINS_DIR=/path/to/bridge OBK_MCP_ADDR=127.0.0.1:7800 ./oblikovati-addinhost &
//	go run ./cmd/mcpcheck --url http://127.0.0.1:7800/mcp   # mcpcheck ships with the bridge add-in
package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"oblikovati.org/addin/dispatch"
	"oblikovati.org/addin/opregistry"
	"oblikovati.org/addin/router"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/addinhost"
	"oblikovati.org/persistence"
)

// defaultHostCallTimeout caps how long an add-in's host call waits to be drained. The
// drain loop runs every drainInterval, so normal latency is sub-millisecond; the timeout
// only guards a stalled loop. Batch/automation runs that recompute heavy features (a
// boolean on a large body) can exceed it, so OBK_HOST_CALL_TIMEOUT overrides it (seconds).
const defaultHostCallTimeout = 10 * time.Second

// hostCallTimeout is defaultHostCallTimeout unless OBK_HOST_CALL_TIMEOUT (a positive number
// of seconds) overrides it; a malformed or non-positive value keeps the default.
func hostCallTimeout() time.Duration {
	raw := os.Getenv("OBK_HOST_CALL_TIMEOUT")
	if raw == "" {
		return defaultHostCallTimeout
	}
	secs, err := strconv.ParseFloat(raw, 64)
	if err != nil || secs <= 0 {
		fmt.Fprintf(os.Stderr, "oblikovati-addinhost: ignoring invalid OBK_HOST_CALL_TIMEOUT %q; using %s\n", raw, defaultHostCallTimeout)
		return defaultHostCallTimeout
	}
	return time.Duration(secs * float64(time.Second))
}

// drainInterval is how often the session goroutine services queued add-in calls. A few
// milliseconds keeps MCP round-trips snappy without busy-spinning the CPU.
const drainInterval = 3 * time.Millisecond

// drainPerTick bounds how many queued add-in calls run per tick, so a burst of requests
// cannot monopolize the loop.
const drainPerTick = 32

// dispatchCapacity is the depth of the add-in→host work queue before Submit blocks for
// backpressure; matches the head's frame-loop dispatcher (cmd/oblikovati-head/addins.go).
const dispatchCapacity = 64

func main() {
	session, err := newHostSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "oblikovati-addinhost: %v\n", err)
		os.Exit(1)
	}
	d := dispatch.New(dispatchCapacity)
	installHost(session, d)
	startAddIns(session, addInsDir())
	runDrainLoop(d)
}

// newHostSession builds a session with a real file-backed store (so save_document /
// documents.saveAs actually write .obk files, as the CLI does) and the standard command
// set registered — the same baseline the head starts from.
func newHostSession() (*app.Session, error) {
	session := app.NewSessionWithStore(persistence.NewPackageStore())
	if err := app.RegisterStandardCommands(session); err != nil {
		return nil, fmt.Errorf("register standard commands: %w", err)
	}
	return session, nil
}

// newRouterHandler returns the host-call handler that routes every add-in request
// through the router over session. It runs on the session goroutine (inside
// Dispatcher.Drain), so it may touch the non-thread-safe model.
func newRouterHandler(session *app.Session) addinhost.Handler {
	rtr := router.New(opregistry.Default())
	return func(method string, req []byte) ([]byte, error) {
		return rtr.Handle(session, method, req)
	}
}

// installHost wires the dispatcher + router handler as the C-ABI host, so every add-in
// host call serializes onto the dispatcher the drain loop services. Call once before
// activating any add-in.
func installHost(session *app.Session, d *dispatch.Dispatcher) {
	addinhost.SetHost(d, newRouterHandler(session), hostCallTimeout())
}

// startAddIns loads, registers, and activates every add-in in dir, reporting load
// errors, version-skips, and per-add-in failures on stderr, and returns the ids it
// activated. The host must already be installed (installHost); the caller must start
// draining right after, because an add-in's Activate may make blocking host calls.
func startAddIns(session *app.Session, dir string) []string {
	fmt.Fprintf(os.Stderr, "oblikovati-addinhost: scanning add-ins in %q\n", dir)
	libs, skipped, err := addinhost.LoadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "oblikovati-addinhost: load add-ins in %q: %v\n", dir, err)
	}
	for _, sk := range skipped {
		fmt.Fprintf(os.Stderr, "oblikovati-addinhost: add-in %q skipped (incompatible): %s\n", sk.ID, sk.Reason)
	}
	activated, failures := activateEach(session, libs)
	for _, f := range failures {
		fmt.Fprintf(os.Stderr, "oblikovati-addinhost: %v\n", f)
	}
	fmt.Fprintf(os.Stderr, "oblikovati-addinhost: %d add-in(s) activated, %d skipped, %d failed\n",
		len(activated), len(skipped), len(failures))
	return activated
}

// activateEach registers and activates every add-in in libs, returning the ids it
// activated and any per-add-in failures. A bad add-in is skipped, never fatal, so one
// broken add-in cannot stop the others' endpoints from coming up.
func activateEach(session *app.Session, libs []*addinhost.LoadedAddIn) (activated []string, failures []error) {
	for _, lib := range libs {
		if err := registerAndActivate(session, lib); err != nil {
			failures = append(failures, err)
			continue
		}
		activated = append(activated, lib.ID())
	}
	return activated, failures
}

// registerAndActivate registers one loaded add-in and activates it. Activation runs the
// add-in's ObkAddInActivate (which starts its MCP server and may make host calls), so the
// caller must be draining the dispatcher. Errors carry the offending add-in id.
func registerAndActivate(session *app.Session, lib *addinhost.LoadedAddIn) error {
	if err := session.AddIns().Register(lib); err != nil {
		return fmt.Errorf("register add-in %q: %w", lib.ID(), err)
	}
	if err := session.AddIns().Activate(session, lib.ID()); err != nil {
		return fmt.Errorf("activate add-in %q: %w", lib.ID(), err)
	}
	return nil
}

// runDrainLoop services queued add-in calls on this (the session) goroutine until
// SIGINT/SIGTERM. The model is not thread-safe, so ALL draining happens here; an add-in's
// HTTP server runs in its own runtime and submits work onto the dispatcher this loop drains.
func runDrainLoop(d *dispatch.Dispatcher) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	drainUntil(d, stop)
}

// drainUntil ticks every drainInterval, draining the dispatcher, until stop delivers a
// signal. Split from runDrainLoop's signal setup so the ticking loop is unit-testable
// (feed it a stop channel) while only the os/signal wait in runDrainLoop stays uncovered.
func drainUntil(d *dispatch.Dispatcher, stop <-chan os.Signal) {
	ticker := time.NewTicker(drainInterval)
	defer ticker.Stop()
	fmt.Fprintln(os.Stderr, "oblikovati-addinhost: draining (Ctrl-C to stop)")
	for {
		select {
		case <-stop:
			fmt.Fprintln(os.Stderr, "oblikovati-addinhost: shutting down")
			return
		case <-ticker.C:
			drainTick(d)
		}
	}
}

// drainTick services up to drainPerTick queued add-in calls and reports how many ran.
// Factored out of runDrainLoop's blocking select so the drain step is unit-testable.
func drainTick(d *dispatch.Dispatcher) int {
	return d.Drain(drainPerTick)
}

// addInsDir is the directory scanned for shared-library add-ins: OBK_ADDINS_DIR if set,
// else an "addins" folder beside the executable.
func addInsDir() string {
	if dir := os.Getenv("OBK_ADDINS_DIR"); dir != "" {
		return dir
	}
	exe, err := os.Executable()
	if err != nil {
		return "addins"
	}
	return filepath.Join(filepath.Dir(exe), "addins")
}
