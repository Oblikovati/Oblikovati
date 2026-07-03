// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"errors"
	"testing"

	"oblikovati.org/app"
)

// The named fakes the typed-adapter tests drive — a model context, decode/marshal DTOs, a
// scripted context resolver, a call-recording handler, and a Count()/Item(i) collection. They
// stand in for the real modelaccess resolvers and wire handlers so the adapter scaffold is tested
// in isolation from any live session state (F.I.R.S.T.).

type fakeContext struct{ label string }

type fakeArgs struct {
	Value int `json:"value"`
}

type fakeResult struct {
	Doubled int `json:"doubled"`
}

// stubResolver stands in for modelaccess.ActivePart: it yields a fixed context, or a fixed error.
type stubResolver struct {
	ctx fakeContext
	err error
}

func (r stubResolver) resolve(*app.Session) (fakeContext, error) { return r.ctx, r.err }

// recordingHandler captures the context and args it was invoked with, and returns a scripted
// result/error, so a test can assert both the value flow and the invocation.
type recordingHandler struct {
	called  bool
	gotCtx  fakeContext
	gotArgs fakeArgs
	err     error
}

func (h *recordingHandler) run(_ *app.Session, ctx fakeContext, in fakeArgs) (fakeResult, error) {
	h.called, h.gotCtx, h.gotArgs = true, ctx, in
	return fakeResult{Doubled: in.Value * 2}, h.err
}

func (h *recordingHandler) runNoArgs(_ *app.Session, ctx fakeContext) (fakeResult, error) {
	h.called, h.gotCtx = true, ctx
	return fakeResult{Doubled: 7}, h.err
}

func (h *recordingHandler) runNoCtx(_ *app.Session, in fakeArgs) (fakeResult, error) {
	h.called, h.gotArgs = true, in
	return fakeResult{Doubled: in.Value * 2}, h.err
}

// intRows is a named Count()/Item(i) collection backing the projectAll test.
type intRows struct{ items []int }

func (c intRows) Count() int     { return len(c.items) }
func (c intRows) Item(i int) int { return c.items[i] }

func indexPlusValue(i, v int) fakeResult { return fakeResult{Doubled: i*100 + v} }

func TestMarshalResult(t *testing.T) {
	sentinel := errors.New("boom")
	if _, err := marshalResult(fakeResult{Doubled: 1}, sentinel); !errors.Is(err, sentinel) {
		t.Fatalf("marshalResult error passthrough = %v, want boom", err)
	}
	raw, err := marshalResult(fakeResult{}, nil) // zero-value result still marshals
	if err != nil {
		t.Fatalf("marshalResult(zero) err = %v", err)
	}
	if string(raw) != `{"doubled":0}` {
		t.Fatalf("marshalResult(zero) = %s, want {\"doubled\":0}", raw)
	}
}

func TestTypedCtxSuccess(t *testing.T) {
	rec := &recordingHandler{}
	h := typedCtx(stubResolver{ctx: fakeContext{label: "part"}}.resolve, rec.run)
	raw, err := h(app.NewSession(), json.RawMessage(`{"value":21}`))
	if err != nil {
		t.Fatalf("typedCtx err = %v", err)
	}
	if !rec.called || rec.gotCtx.label != "part" || rec.gotArgs.Value != 21 {
		t.Fatalf("handler saw ctx=%+v args=%+v called=%v", rec.gotCtx, rec.gotArgs, rec.called)
	}
	if string(raw) != `{"doubled":42}` {
		t.Fatalf("typedCtx result = %s, want {\"doubled\":42}", raw)
	}
}

func TestTypedCtxResolveErrorWinsOverDecode(t *testing.T) {
	// Context is resolved BEFORE decode, so with both a resolve error and invalid JSON the
	// resolve error is what an add-in observes — the canonical order (#1649). The handler must
	// never run.
	noCtx := errors.New("no active part")
	rec := &recordingHandler{}
	h := typedCtx(stubResolver{err: noCtx}.resolve, rec.run)
	_, err := h(app.NewSession(), json.RawMessage(`{ not json`))
	if !errors.Is(err, noCtx) {
		t.Fatalf("typedCtx err = %v, want the resolve error", err)
	}
	if rec.called {
		t.Fatal("handler ran despite the resolve error")
	}
}

func TestTypedCtxDecodeError(t *testing.T) {
	rec := &recordingHandler{}
	h := typedCtx(stubResolver{ctx: fakeContext{label: "part"}}.resolve, rec.run)
	_, err := h(app.NewSession(), json.RawMessage(`{"value":"not-an-int"}`))
	if err == nil {
		t.Fatal("typedCtx(bad args) err = nil, want a decode error")
	}
	if rec.called {
		t.Fatal("handler ran despite the decode error")
	}
}

func TestTypedCtxHandlerError(t *testing.T) {
	sentinel := errors.New("kernel refused")
	rec := &recordingHandler{err: sentinel}
	h := typedCtx(stubResolver{ctx: fakeContext{label: "part"}}.resolve, rec.run)
	if _, err := h(app.NewSession(), json.RawMessage(`{"value":1}`)); !errors.Is(err, sentinel) {
		t.Fatalf("typedCtx handler error = %v, want kernel refused", err)
	}
}

func TestTypedNoContext(t *testing.T) {
	rec := &recordingHandler{}
	h := typed(rec.runNoCtx)
	raw, err := h(app.NewSession(), json.RawMessage(`{"value":9}`))
	if err != nil || string(raw) != `{"doubled":18}` {
		t.Fatalf("typed = %s, %v; want {\"doubled\":18}", raw, err)
	}
	if _, err := h(app.NewSession(), json.RawMessage(`{bad`)); err == nil {
		t.Fatal("typed(bad args) err = nil, want a decode error")
	}
}

func TestCtxQuery(t *testing.T) {
	rec := &recordingHandler{}
	h := ctxQuery(stubResolver{ctx: fakeContext{label: "asm"}}.resolve, rec.runNoArgs)
	raw, err := h(app.NewSession(), nil) // no-arg handler ignores raw entirely
	if err != nil || string(raw) != `{"doubled":7}` {
		t.Fatalf("ctxQuery = %s, %v; want {\"doubled\":7}", raw, err)
	}
	if rec.gotCtx.label != "asm" {
		t.Fatalf("ctxQuery ctx = %q, want asm", rec.gotCtx.label)
	}

	resolveErr := errors.New("no active assembly")
	fail := ctxQuery(stubResolver{err: resolveErr}.resolve, rec.runNoArgs)
	if _, err := fail(app.NewSession(), nil); !errors.Is(err, resolveErr) {
		t.Fatalf("ctxQuery resolve error = %v, want no active assembly", err)
	}
}

func TestProjectAll(t *testing.T) {
	got := projectAll(intRows{items: []int{5, 6, 7}}, indexPlusValue)
	want := []fakeResult{{Doubled: 5}, {Doubled: 106}, {Doubled: 207}}
	if len(got) != len(want) {
		t.Fatalf("projectAll len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("projectAll[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
	if empty := projectAll(intRows{}, indexPlusValue); len(empty) != 0 {
		t.Fatalf("projectAll(empty) = %v, want []", empty)
	}
}
