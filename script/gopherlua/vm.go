// SPDX-License-Identifier: GPL-2.0-only

// Package gopherlua is the ONLY importer of github.com/yuin/gopher-lua. It implements
// the project-owned script.Engine seam over the pure-Go Lua 5.1 VM: an opt-in
// sandbox (sandbox.go), context/memory resource limits (quota.go), structural Lua⇄JSON
// conversion (convert.go), and total panic containment here (ADR-0028). No other
// package may import gopher-lua, so the VM stays swappable behind script.Engine.
package gopherlua

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	lua "github.com/yuin/gopher-lua"

	"oblikovati/script"
)

// Engine is the gopher-lua-backed script.Engine. It holds no per-run state — each Run
// builds a fresh, isolated LState — so one Engine is safe to reuse across runs (the
// runner serialises them with a per-session lock).
type Engine struct{}

// New returns a gopher-lua Engine. It is the only constructor the rest of the app
// calls to obtain a script.Engine implementation.
func New() *Engine { return &Engine{} }

// compile-time proof the gopher-lua impl satisfies the project-owned seam.
var _ script.Engine = (*Engine)(nil)

// Run executes source against globals under lim and returns the outcome. It NEVER
// panics to its caller: a VM panic, a binding panic, or a convert panic is recovered
// into Result.Err (the absolute guarantee of ADR-0028 §2). Syntax errors, runtime
// errors, and context cancellation (timeout/Stop) likewise come back as Result.Err.
func (e *Engine) Run(ctx context.Context, source string, globals script.Globals, lim script.Limits) (res script.Result) {
	start := time.Now()
	defer func() {
		res.Duration = time.Since(start)
		res.Op = instructionBudget(lim)
		if rec := recover(); rec != nil {
			res.Err = recoveredPanic(rec)
		}
	}()
	return e.runGuarded(ctx, source, globals, lim, start)
}

// runGuarded builds the sandboxed state, applies the deadline, injects the host door,
// and runs the source. It is the body the outer Run recovers around.
func (e *Engine) runGuarded(ctx context.Context, source string, g script.Globals, lim script.Limits, start time.Time) script.Result {
	l := lua.NewState(memoryOptions(lim))
	defer l.Close()
	openSafeLibs(l)

	runCtx, cancel := runDeadline(ctx, lim)
	defer cancel()
	l.SetContext(runCtx)

	var out outputSink
	registerPrint(l, g.Print, &out)
	installOblikovati(l, g)

	err := l.DoString(source)
	return script.Result{Stdout: out.String(), Err: doErr(err, runCtx), Duration: time.Since(start)}
}

// recoveredPanic turns a recovered value into a logged, descriptive error so a genuine
// host bug stays visible in observability (slog + stack) yet is never fatal to the app
// (ADR-0028 §2, risk "two layers of recover masking real bugs").
func recoveredPanic(rec interface{}) error {
	stack := string(debug.Stack())
	slog.Error("script: recovered panic in Lua VM", "value", rec, "stack", stack)
	return fmt.Errorf("script: recovered panic: %v", rec)
}

// doErr maps DoString's error to the run's failure, preferring the context error so a
// timeout/cancel surfaces as such rather than as the Lua "context deadline exceeded"
// re-raise. A nil DoString error with an expired context still reports the context
// error (defensive); otherwise it returns the Lua error verbatim (it carries the line).
func doErr(luaErr error, ctx context.Context) error {
	if ctx.Err() != nil {
		return fmt.Errorf("script: %w", ctx.Err())
	}
	return luaErr
}
