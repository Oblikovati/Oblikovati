// SPDX-License-Identifier: GPL-2.0-only

// Package bridge connects the sandboxed script runtime to the host's wire method
// surface. It defines the two client.Caller flavours a script can run behind — a
// direct in-proc caller (CLI, no UI to protect) and a dispatched caller (the head,
// where each call must hop to the session goroutine) — and the generic
// oblikovati.call adapter that turns a client.Caller into a script.CallFunc.
//
// The bridge is method-name- and schema-agnostic by construction: it forwards
// (method, JSON) pairs verbatim, so every oblikovati.org/api/wire method is callable from
// Lua the instant it is registered in addin/router, with zero per-method code
// (ADR-0028 §3).
package bridge

import (
	"context"
	"fmt"

	"oblikovati.org/addin/dispatch"
	"oblikovati.org/api/client"
	"oblikovati.org/app"
)

// RouteFunc is the host's method router as a bare function: addin/router's
// Router.Handle bound to one *app.Session. Injecting it (rather than importing router
// here) keeps the bridge decoupled from the router package and trivially fakeable.
type RouteFunc func(s *app.Session, method string, req []byte) ([]byte, error)

// DirectCaller runs each host call synchronously on the calling goroutine. Correct for
// the single-threaded CLI, where there is no frame loop and no UI to protect: the
// script worker calls straight into Router.Handle (ADR-0028 §4, CLI mode).
type DirectCaller struct {
	route   RouteFunc
	session *app.Session
}

// NewDirectCaller binds route to session for synchronous in-proc calls.
func NewDirectCaller(route RouteFunc, session *app.Session) *DirectCaller {
	return &DirectCaller{route: route, session: session}
}

// Call satisfies client.Caller by invoking the router directly on this goroutine.
func (c *DirectCaller) Call(method string, req []byte) ([]byte, error) {
	if c.route == nil {
		return nil, fmt.Errorf("bridge: DirectCaller has no route for %q", method)
	}
	return c.route(c.session, method, req)
}

var _ client.Caller = (*DirectCaller)(nil)

// DispatchedCaller marshals each host call onto the session goroutine via the existing
// addin/dispatch.Dispatcher — the same seam add-ins use — so a looping script never
// touches the model off-goroutine and the UI frame loop keeps draining (ADR-0028 §4).
// The head wires this in Phase 2; it is built and tested now so the seam is real.
type DispatchedCaller struct {
	route      RouteFunc
	session    *app.Session
	dispatcher *dispatch.Dispatcher
	ctx        context.Context
}

// NewDispatchedCaller binds route+session to dispatcher; ctx cancels a blocked submit
// (Stop/shutdown). A nil ctx means context.Background().
func NewDispatchedCaller(route RouteFunc, session *app.Session, d *dispatch.Dispatcher, ctx context.Context) *DispatchedCaller {
	if ctx == nil {
		ctx = context.Background()
	}
	return &DispatchedCaller{route: route, session: session, dispatcher: d, ctx: ctx}
}

// Call satisfies client.Caller by submitting the routed call as a dispatch.Job and
// blocking until the session goroutine drains it (or ctx fires / the dispatcher closes).
func (c *DispatchedCaller) Call(method string, req []byte) ([]byte, error) {
	if c.route == nil || c.dispatcher == nil {
		return nil, fmt.Errorf("bridge: DispatchedCaller not wired for %q", method)
	}
	return c.dispatcher.Submit(c.ctx, func() ([]byte, error) {
		return c.route(c.session, method, req)
	})
}

var _ client.Caller = (*DispatchedCaller)(nil)
