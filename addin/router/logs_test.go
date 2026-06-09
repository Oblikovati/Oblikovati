// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"strings"
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// TestHandleRecoversPanicIntoError: a panicking handler does not crash the host — Handle
// returns a detailed, method-named error and records it (with a stack) in the trace.
func TestHandleRecoversPanicIntoError(t *testing.T) {
	r := New(opregistry.Default())
	r.handlers["test.boom"] = func(*app.Session, json.RawMessage) (json.RawMessage, error) {
		panic("kaboom")
	}
	s := app.NewSession()

	_, err := r.Handle(s, "test.boom", []byte("{}"))
	if err == nil || !strings.Contains(err.Error(), "test.boom: panic") || !strings.Contains(err.Error(), "kaboom") {
		t.Fatalf("err = %v, want a method-named panic error", err)
	}
	// The host is still usable after the recovered panic.
	if _, err := r.Handle(s, wire.MethodDocumentsList, []byte("{}")); err != nil {
		t.Fatalf("router unusable after panic: %v", err)
	}

	res := r.trace.Tail(0, "error", 0)
	if len(res.Records) != 1 {
		t.Fatalf("error-level records = %d, want 1 (the panic)", len(res.Records))
	}
	rec := res.Records[0]
	if rec.Method != "test.boom" || rec.Panic == "" || rec.Stack == "" {
		t.Errorf("panic record = %+v, want method+panic+stack", rec)
	}
}

// TestHandleTracesEachCall: a successful and a failing call each append one trace entry, with
// the failure carrying a method-prefixed error.
func TestHandleTracesEachCall(t *testing.T) {
	r := New(opregistry.Default())
	s := app.NewSession()

	if _, err := r.Handle(s, wire.MethodDocumentsList, []byte("{}")); err != nil {
		t.Fatalf("documents.list: %v", err)
	}
	// model.tree with no active part fails — proves failures are traced + named.
	if _, err := r.Handle(s, wire.MethodModelTree, []byte("{}")); err == nil {
		t.Fatal("model.tree with no part should fail")
	}

	res := r.trace.Tail(0, "", 0)
	if len(res.Records) != 2 {
		t.Fatalf("trace has %d records, want 2", len(res.Records))
	}
	if !res.Records[0].OK || res.Records[0].Method != wire.MethodDocumentsList {
		t.Errorf("rec0 = %+v, want ok documents.list", res.Records[0])
	}
	if res.Records[1].OK || !strings.HasPrefix(res.Records[1].Error, wire.MethodModelTree) {
		t.Errorf("rec1 = %+v, want failed model.tree with prefixed error", res.Records[1])
	}
}

// TestLogsTailNotSelfTraced: polling logs.tail must not append to the trace it reads.
func TestLogsTailNotSelfTraced(t *testing.T) {
	r := New(opregistry.Default())
	s := app.NewSession()

	out, err := r.Handle(s, wire.MethodLogsTail, []byte(`{}`))
	if err != nil {
		t.Fatalf("logs.tail: %v", err)
	}
	var res wire.LogsResult
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(res.Records) != 0 {
		t.Errorf("logs.tail recorded itself: %+v", res.Records)
	}
}
