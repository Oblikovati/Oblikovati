// SPDX-License-Identifier: GPL-2.0-only

// Package runner stitches a script.Engine, a host Globals door, and resource Limits
// together and runs one script on a dedicated worker goroutine. It owns the run lock
// (one script per ScriptRunner at a time, for deterministic dispatch ordering) and the
// default Limits for the CLI and GUI (ADR-0028 §4, §5).
package runner

import (
	"context"
	"errors"
	"sync"
	"time"

	"oblikovati.org/api/client"
	"oblikovati.org/script"
	"oblikovati.org/script/bridge"
)

// ErrBusy is returned when a Run is requested while another is in flight on the same
// ScriptRunner (the one-script-per-session lock, ADR-0028 §4).
var ErrBusy = errors.New("runner: a script is already running")

// DefaultCLILimits are the headless CLI defaults: a generous wall-clock and memory cap
// so a real automation script finishes, while still bounding a runaway. The CLI has no
// UI to protect, so the budget is larger than the GUI's.
var DefaultCLILimits = script.Limits{
	Instructions: 0, // gopher-lua exposes no opcode hook; the wall-clock is the enforced guard
	Wall:         30 * time.Second,
	MemBytes:     256 << 20, // 256 MiB
}

// DefaultGUILimits are the interactive defaults: a short wall-clock so a buggy console
// script can't stall a frame for long, backstopped by the bounded dispatcher drain.
var DefaultGUILimits = script.Limits{
	Instructions: 0,
	Wall:         5 * time.Second,
	MemBytes:     128 << 20, // 128 MiB
}

// ScriptRunner runs Lua sources against one host caller under one engine. It is reused
// across runs; the run lock serialises them.
type ScriptRunner struct {
	engine  script.Engine
	caller  client.Caller
	methods func() []string
	mu      sync.Mutex // the run lock: one script at a time
	running bool
}

// New builds a runner. engine is the VM (e.g. gopherlua.New()); caller is the host
// transport (a bridge.DirectCaller for the CLI, a bridge.DispatchedCaller for the
// head); methods backs oblikovati.methods() for discoverability (may be nil).
func New(engine script.Engine, caller client.Caller, methods func() []string) *ScriptRunner {
	return &ScriptRunner{engine: engine, caller: caller, methods: methods}
}

// Run executes source under lim on a fresh worker goroutine and returns its Result.
// It holds the run lock for the duration (ErrBusy if a run is already in flight), wires
// the host door from the caller, and forwards print output to onPrint (may be nil).
// ctx cancellation (Stop / Ctrl-C / shutdown) unwinds the run promptly via the engine's
// context-aware loop.
func (r *ScriptRunner) Run(ctx context.Context, source string, lim script.Limits, onPrint func(string)) (script.Result, error) {
	if !r.acquire() {
		return script.Result{}, ErrBusy
	}
	defer r.release()
	globals := script.Globals{
		Call:    bridge.CallFuncOf(r.caller),
		Print:   onPrint,
		Methods: r.methods,
	}
	return r.runOnWorker(ctx, source, globals, lim), nil
}

// runOnWorker runs engine.Run on a dedicated goroutine so the script's own computation
// (loops, recursion) never executes on the caller's goroutine; the caller blocks on the
// result. Host API calls hop to the session goroutine inside the caller (bridge), not
// here — this goroutine is purely the Lua interpreter's.
func (r *ScriptRunner) runOnWorker(ctx context.Context, source string, g script.Globals, lim script.Limits) script.Result {
	done := make(chan script.Result, 1)
	go func() { done <- r.engine.Run(ctx, source, g, lim) }()
	return <-done
}

// acquire takes the run lock; returns false if a run is already in flight.
func (r *ScriptRunner) acquire() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running {
		return false
	}
	r.running = true
	return true
}

// release frees the run lock.
func (r *ScriptRunner) release() {
	r.mu.Lock()
	r.running = false
	r.mu.Unlock()
}
