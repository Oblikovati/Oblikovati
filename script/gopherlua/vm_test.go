// SPDX-License-Identifier: GPL-2.0-only

package gopherlua

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"oblikovati.org/script"
)

// errBoom is a sentinel host-call error used by the pcall containment test.
var errBoom = errors.New("host call failed")

// fakeCall is a named fake CallFunc that records the last method/args and returns a
// scripted reply (or error) — the host door under test without a real router.
type fakeCall struct {
	reply    []byte
	err      error
	panicMsg string
	method   string
	argsJSON []byte
}

func (f *fakeCall) call(method string, argsJSON []byte) ([]byte, error) {
	f.method = method
	f.argsJSON = argsJSON
	if f.panicMsg != "" {
		panic(f.panicMsg)
	}
	return f.reply, f.err
}

func globalsWith(f *fakeCall) script.Globals {
	return script.Globals{Call: f.call, Print: func(string) {}}
}

func TestRunCallsHostAndConvertsResult(t *testing.T) {
	f := &fakeCall{reply: []byte(`{"id":7,"name":"box"}`)}
	src := `local r = oblikovati.call("documents.create", { type = "part", name = "box" })
	        print(r.id)
	        print(r.name)`
	res := New().Run(context.Background(), src, globalsWith(f), script.Limits{Wall: time.Second})
	if res.Err != nil {
		t.Fatalf("Run: %v", res.Err)
	}
	if f.method != "documents.create" {
		t.Errorf("method = %q, want documents.create", f.method)
	}
	var got map[string]any
	if err := json.Unmarshal(f.argsJSON, &got); err != nil {
		t.Fatalf("args not JSON: %v", err)
	}
	if got["type"] != "part" || got["name"] != "box" {
		t.Errorf("args round-trip wrong: %v", got)
	}
	if !strings.Contains(res.Stdout, "7") || !strings.Contains(res.Stdout, "box") {
		t.Errorf("stdout missing converted result: %q", res.Stdout)
	}
}

func TestRunInfiniteLoopHitsWallClock(t *testing.T) {
	start := time.Now()
	res := New().Run(context.Background(), "while true do end", script.Globals{}, script.Limits{Wall: 150 * time.Millisecond})
	if res.Err == nil {
		t.Fatal("infinite loop must return an error, not hang")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("loop not cut promptly: %v", elapsed)
	}
}

func TestRunSyntaxErrorReturnsError(t *testing.T) {
	res := New().Run(context.Background(), "this is not lua %%%", script.Globals{}, script.Limits{Wall: time.Second})
	if res.Err == nil {
		t.Fatal("syntax error must return an error")
	}
}

func TestRunRuntimeErrorCarriesMessage(t *testing.T) {
	res := New().Run(context.Background(), `error("boom-value")`, script.Globals{}, script.Limits{Wall: time.Second})
	if res.Err == nil {
		t.Fatal("runtime error must return an error")
	}
	if !strings.Contains(res.Err.Error(), "boom-value") {
		t.Errorf("error should carry the offending value: %q", res.Err.Error())
	}
}

func TestRunRecoversHostCallPanic(t *testing.T) {
	f := &fakeCall{panicMsg: "host blew up"}
	res := New().Run(context.Background(), `oblikovati.call("x", {})`, globalsWith(f), script.Limits{Wall: time.Second})
	if res.Err == nil {
		t.Fatal("a panicking host call must be recovered into an error")
	}
	// The host survives: a second run on the same engine still works.
	ok := New().Run(context.Background(), `print("alive")`, script.Globals{Print: func(string) {}}, script.Limits{Wall: time.Second})
	if ok.Err != nil {
		t.Fatalf("engine should survive a recovered panic: %v", ok.Err)
	}
}

func TestRunHostCallErrorIsCatchableWithPcall(t *testing.T) {
	f := &fakeCall{err: errBoom}
	src := `local ok, msg = pcall(function() oblikovati.call("x", {}) end)
	        if ok then error("expected failure") end
	        print("caught")`
	res := New().Run(context.Background(), src, globalsWith(f), script.Limits{Wall: time.Second})
	if res.Err != nil {
		t.Fatalf("pcall should let the script handle the host error: %v", res.Err)
	}
	if !strings.Contains(res.Stdout, "caught") {
		t.Errorf("script did not catch the host error: %q", res.Stdout)
	}
}

func TestRunCancellationStopsPromptly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled
	start := time.Now()
	res := New().Run(ctx, "while true do end", script.Globals{}, script.Limits{})
	if res.Err == nil {
		t.Fatal("a cancelled context must stop the run with an error")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("cancellation not honoured promptly: %v", elapsed)
	}
}

func TestSandboxDeniesGlobals(t *testing.T) {
	for _, name := range deniedLibNames {
		src := "if " + name + " ~= nil then error(\"" + name + " is reachable\") end"
		res := New().Run(context.Background(), src, script.Globals{}, script.Limits{Wall: time.Second})
		if res.Err != nil {
			t.Errorf("denied global %q: %v", name, res.Err)
		}
	}
}

func TestSandboxAllowsSafeLibs(t *testing.T) {
	src := `assert(math.floor(3.7) == 3)
	        assert(string.upper("a") == "A")
	        local t = {1,2,3}; assert(#t == 3)`
	res := New().Run(context.Background(), src, script.Globals{}, script.Limits{Wall: time.Second})
	if res.Err != nil {
		t.Fatalf("safe libs should be available: %v", res.Err)
	}
}

func TestMethodsBindingExposesList(t *testing.T) {
	g := script.Globals{
		Print:   func(string) {},
		Methods: func() []string { return []string{"a.b", "c.d"} },
	}
	res := New().Run(context.Background(), `print(#oblikovati.methods())`, g, script.Limits{Wall: time.Second})
	if res.Err != nil {
		t.Fatalf("Run: %v", res.Err)
	}
	if !strings.Contains(res.Stdout, "2") {
		t.Errorf("methods() should expose 2 names, stdout=%q", res.Stdout)
	}
}
