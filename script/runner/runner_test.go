// SPDX-License-Identifier: GPL-2.0-only

package runner

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/addin/router"
	"oblikovati.org/api/client"
	"oblikovati.org/app"
	"oblikovati.org/script"
	"oblikovati.org/script/bridge"
	"oblikovati.org/script/gopherlua"
)

// newRealRunner builds a runner over a real Session + router (the in-proc CLI wiring),
// returning the runner and the session so a test can assert model effects.
func newRealRunner() (*ScriptRunner, *app.Session) {
	s := app.NewSession()
	rtr := router.New(opregistry.Default())
	caller := bridge.NewDirectCaller(rtr.Handle, s)
	return New(gopherlua.New(), caller, rtr.Methods), s
}

func TestRunScriptAffectsModel(t *testing.T) {
	r, s := newRealRunner()
	src := `oblikovati.call("documents.create", { type = "part", name = "block" })
	        oblikovati.call("parameters.add", { name = "width", expression = "4 cm" })`
	res, err := r.Run(context.Background(), src, fastLimits(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Err != nil {
		t.Fatalf("script: %v", res.Err)
	}
	if s.ActiveDocument() == nil {
		t.Fatal("script did not create a document")
	}
	assertParameter(t, r, "width", "40 mm")
}

// assertParameter runs a follow-up script that reads the parameter back, proving the
// table↔JSON↔router round-trip both ways (the model really changed).
func assertParameter(t *testing.T, r *ScriptRunner, name, wantValue string) {
	t.Helper()
	var line string
	src := `local ps = oblikovati.call("parameters.list", {})
	        for _, p in ipairs(ps.parameters) do
	          if p.name == "` + name + `" then print(p.value) end
	        end`
	res, err := r.Run(context.Background(), src, fastLimits(), func(s string) { line = s })
	if err != nil || res.Err != nil {
		t.Fatalf("read-back run: err=%v scriptErr=%v", err, res.Err)
	}
	if strings.TrimSpace(line) != wantValue {
		t.Errorf("parameter %q = %q, want %q", name, line, wantValue)
	}
}

// TestTypedGroupSugarAffectsModel proves the auto-derived group tables
// (oblikovati.documents.create{…}) drive the real model end to end, identical to the
// generic oblikovati.call path (ADR-0028 §4.1).
func TestTypedGroupSugarAffectsModel(t *testing.T) {
	r, s := newRealRunner()
	src := `oblikovati.documents.create{ type = "part", name = "block" }
	        oblikovati.parameters.add{ name = "depth", expression = "5 cm" }`
	res, err := r.Run(context.Background(), src, fastLimits(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Err != nil {
		t.Fatalf("script: %v", res.Err)
	}
	if s.ActiveDocument() == nil {
		t.Fatal("typed-group script did not create a document")
	}
	assertParameter(t, r, "depth", "50 mm")
}

func TestRunInfiniteLoopReturnsErrorHostSurvives(t *testing.T) {
	r, _ := newRealRunner()
	res, err := r.Run(context.Background(), "while true do end", script.Limits{Wall: 150 * time.Millisecond}, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Err == nil {
		t.Fatal("infinite loop must yield a script error")
	}
	// Host survives: a normal script still runs afterwards.
	ok, _ := r.Run(context.Background(), `print("alive")`, fastLimits(), nil)
	if ok.Err != nil {
		t.Fatalf("host did not survive the runaway: %v", ok.Err)
	}
}

// panicEngine is a named fake script.Engine whose Run panics, proving the runner's
// engine is the recover seam (a panicking engine must not crash the process).
type panicEngine struct{}

func (panicEngine) Run(context.Context, string, script.Globals, script.Limits) script.Result {
	panic("engine exploded")
}

func TestRunSurfacesRecoveredEnginePanicAsError(t *testing.T) {
	// The script.Engine contract is that Run NEVER panics to its caller (ADR-0028 §2);
	// the real gopherlua engine recovers internally. recoveringAdapter models that
	// contract over a panicking engine, and this test pins that the runner plumbs the
	// recovered error through as Result.Err — not as a process crash.
	r := New(recoveringAdapter{panicEngine{}}, &nopCaller{}, nil)
	res, err := r.Run(context.Background(), "noop", fastLimits(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Err == nil {
		t.Fatal("a recovered engine panic must surface as a script error")
	}
}

func TestRunLockRejectsConcurrentRun(t *testing.T) {
	r := New(blockingEngine{started: make(chan struct{}), release: make(chan struct{})}, &nopCaller{}, nil)
	be := r.engine.(blockingEngine)
	var wg sync.WaitGroup
	wg.Go(func() {
		_, _ = r.Run(context.Background(), "first", fastLimits(), nil)
	})
	<-be.started
	if _, err := r.Run(context.Background(), "second", fastLimits(), nil); err != ErrBusy {
		t.Errorf("second concurrent run: err = %v, want ErrBusy", err)
	}
	close(be.release)
	wg.Wait()
}

func fastLimits() script.Limits { return script.Limits{Wall: 2 * time.Second} }

// nopCaller is a named fake client.Caller that returns an empty object.
type nopCaller struct{}

func (*nopCaller) Call(string, []byte) ([]byte, error) { return []byte(`{}`), nil }

var _ client.Caller = (*nopCaller)(nil)

// recoveringAdapter wraps an engine so its panic is recovered into Result.Err, matching
// the real gopherlua engine's contract (used to test the runner's lock/result plumbing
// against a panic without crashing the test process).
type recoveringAdapter struct{ inner script.Engine }

func (a recoveringAdapter) Run(ctx context.Context, src string, g script.Globals, lim script.Limits) (res script.Result) {
	defer func() {
		if rec := recover(); rec != nil {
			res.Err = fmt.Errorf("recovered panic: %v", rec)
		}
	}()
	return a.inner.Run(ctx, src, g, lim)
}

// blockingEngine blocks in Run until released, so a test can observe the run lock.
type blockingEngine struct {
	started chan struct{}
	release chan struct{}
}

func (b blockingEngine) Run(context.Context, string, script.Globals, script.Limits) script.Result {
	close(b.started)
	<-b.release
	return script.Result{}
}
