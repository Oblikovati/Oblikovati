// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/script"
	"oblikovati.org/script/gopherlua"
)

// Script-run budgets. The wall-clock bounds a runaway (the only enforced guard, since
// gopher-lua has no opcode hook — ADR-0028 §7); the memory cap is best-effort via the
// bounded VM. A caller may request a tighter/looser wall up to maxScriptWall.
const (
	defaultScriptWall = 10 * time.Second
	maxScriptWall     = 60 * time.Second
	scriptMemBytes    = 256 << 20 // 256 MiB
)

// scriptsRun runs a whole Lua program against the live model in one call
// (wire.MethodScriptRun), so an MCP/LLM client can submit a complete automation program
// instead of issuing N method calls. It runs the source INLINE on the calling goroutine —
// the same one router.Handle is on (the session goroutine for the GUI via the dispatcher;
// the only goroutine for the CLI) — with a host door that calls r.Handle re-entrantly.
// router.Handle holds no lock, so the nested calls are safe and deadlock-free, and each
// inner mutating call still emits its own edit.committed. The sandbox + wall-clock bound a
// runaway (ADR-0028); a script failure is returned in the result (Error), not as a handler
// error, so the caller always gets the output plus the reason.
func (r *Router) scriptsRun(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.ScriptRunArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Source) == "" {
		return nil, errors.New("scripts.run: source is required, got an empty program")
	}
	wall := scriptWall(in.WallMs)
	ctx, cancel := context.WithTimeout(context.Background(), wall)
	defer cancel()
	globals := script.Globals{
		Call:    func(method string, req []byte) ([]byte, error) { return r.Handle(s, method, req) },
		Methods: r.Methods,
	}
	res := gopherlua.New().Run(ctx, in.Source, globals, script.Limits{Wall: wall, MemBytes: scriptMemBytes})
	out := wire.ScriptRunResult{Output: res.Stdout, DurationMs: res.Duration.Milliseconds(), Ops: res.Op}
	if res.Err != nil {
		out.Error = res.Err.Error()
	}
	return json.Marshal(out)
}

// scriptWall resolves the per-run wall-clock budget: the requested milliseconds (clamped
// to maxScriptWall) or the default when unset/non-positive.
func scriptWall(ms int) time.Duration {
	if ms <= 0 {
		return defaultScriptWall
	}
	if d := time.Duration(ms) * time.Millisecond; d < maxScriptWall {
		return d
	}
	return maxScriptWall
}
