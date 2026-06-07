// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"oblikovati/api/wire"
)

// argsJSON marshals a request struct to the JSON string the call helper expects.
func argsJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	return string(b)
}

// TestScriptsRunDrivesModelAndCapturesOutput: a program adds a parameter through the
// re-entrant router and reads it back; the printed value proves the real model changed
// (not just the script's VM), end to end via scripts.run.
func TestScriptsRunDrivesModelAndCapturesOutput(t *testing.T) {
	r, s := seededSession(t)
	src := `oblikovati.parameters.add{ name = "depth", expression = "5 cm" }
	        local p = oblikovati.parameters.get{ name = "depth" }
	        print(p.value)`
	var res wire.ScriptRunResult
	call(t, r, s, wire.MethodScriptRun, argsJSON(t, wire.ScriptRunArgs{Source: src}), &res)
	if res.Error != "" {
		t.Fatalf("script error: %s", res.Error)
	}
	if !strings.Contains(res.Output, "50 mm") {
		t.Errorf("output = %q, want it to contain the read-back value 50 mm", res.Output)
	}
}

// TestScriptsRunReportsScriptErrorInResult: a runtime error rides in the result's Error
// field (with the offending value), and the call itself succeeds at the transport level.
func TestScriptsRunReportsScriptErrorInResult(t *testing.T) {
	r, s := seededSession(t)
	var res wire.ScriptRunResult
	call(t, r, s, wire.MethodScriptRun, argsJSON(t, wire.ScriptRunArgs{Source: `error("boom-value")`}), &res)
	if res.Error == "" || !strings.Contains(res.Error, "boom-value") {
		t.Errorf("script error should be reported with the value, got %q", res.Error)
	}
}

// TestScriptsRunRejectsEmptySource: an empty program is a request error (not a silent
// no-op), naming the problem.
func TestScriptsRunRejectsEmptySource(t *testing.T) {
	r, s := seededSession(t)
	if _, err := r.Handle(s, wire.MethodScriptRun, []byte(`{"source":"   "}`)); err == nil {
		t.Fatal("empty source should be rejected")
	}
}

// TestScriptsRunHonorsWallClock: an infinite loop is cut by the per-run wall-clock and
// reported as a script error, promptly — the host is never hung.
func TestScriptsRunHonorsWallClock(t *testing.T) {
	r, s := seededSession(t)
	start := time.Now()
	var res wire.ScriptRunResult
	call(t, r, s, wire.MethodScriptRun, argsJSON(t, wire.ScriptRunArgs{Source: "while true do end", WallMs: 150}), &res)
	if res.Error == "" {
		t.Fatal("an infinite loop must be reported as a script error")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("wall-clock not honoured promptly: %v", elapsed)
	}
}
