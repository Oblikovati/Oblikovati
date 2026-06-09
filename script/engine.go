// SPDX-License-Identifier: GPL-2.0-only

// Package script is the embedded-scripting seam for the GPL application: it lets a
// user run a Lua program that drives the live model through the exact same wire
// method surface (oblikovati.org/api/wire) that add-ins and the MCP bridge use, under a
// sandbox so a buggy or malicious script can never crash, hang, or escape the host
// (ADR-0028).
//
// The concrete Lua VM (gopher-lua) lives behind the Engine interface in the
// script/gopherlua subpackage — the only place that imports the third-party VM — so
// the runtime is swappable without touching call sites. script/bridge builds the one
// host door (the oblikovati global) over a wire Caller, and script/runner stitches
// the engine, the bridge, and the resource limits together on a worker goroutine.
package script

import (
	"context"
	"time"
)

// Limits bounds one script run so a runaway can never starve the host. A zero field
// means "unbounded" and is intended for tests only — production callers always set
// all three (see runner.DefaultCLILimits).
type Limits struct {
	Instructions uint64        // hard opcode budget across the run (0 = unlimited)
	Wall         time.Duration // wall-clock deadline (0 = no deadline)
	MemBytes     int64         // approximate Lua allocation cap (0 = unbounded)
}

// Result reports the outcome of one Run. Err is the single channel for every failure
// mode — syntax error, runtime error, quota breach, context cancellation, or a
// recovered panic — so a caller never has to handle a panic. Op and Duration are for
// metrics and the containment tests.
type Result struct {
	Stdout   string        // captured print() output
	Err      error         // nil on success; otherwise the failure (never a panic)
	Op       uint64        // opcodes executed (best-effort; for metrics/tests)
	Duration time.Duration // wall-clock spent in Run
}

// CallFunc backs the oblikovati.call(method, argsTable) host door. It takes a method
// name and the JSON-encoded args table and returns the JSON-encoded result. It runs
// (logically) on the host's session goroutine — script/bridge marshals it there.
type CallFunc func(method string, argsJSON []byte) (resultJSON []byte, err error)

// Globals is everything the host injects into the otherwise-empty sandbox: the call
// door, the captured print sink, and the introspection list. Nothing else reaches the
// host (ADR-0028 §2).
type Globals struct {
	Call    CallFunc        // backs oblikovati.call
	Print   func(string)    // backs the captured print(); nil discards
	Methods func() []string // backs oblikovati.methods(); nil yields an empty list
}

// Engine runs one Lua source against a set of host Globals under Limits. It MUST NEVER
// panic to its caller: a script panic, syntax error, quota breach, or context
// cancellation all come back as Result.Err. Implementations wrap a concrete VM.
//
// Example:
//
//	eng := gopherlua.New()
//	res := eng.Run(ctx, `oblikovati.call("documents.list", {})`, globals, lim)
//	if res.Err != nil { log.Print(res.Err) }
type Engine interface {
	Run(ctx context.Context, source string, globals Globals, lim Limits) Result
}
