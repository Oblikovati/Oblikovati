// SPDX-License-Identifier: GPL-2.0-only

// Package router is the host-side API: it dispatches the bridge's JSON method
// contract (commands.*, documents.*, parameters.*, model.*, features.*) to the live
// *app.Session. It is the single place that contract is wired to the model, and is
// pure Go (no cgo, no MCP/HTTP) so it is fully headless-testable. Handle runs on the
// session goroutine (via the Dispatcher), so handlers may READ the model directly —
// queries carry no invariants to drift. MUTATIONS delegate to an app.Session verb
// (the application service the head UI also drives), so both drivers share one
// invariant per aggregate instead of the divergence audited as B1 (#1612); the
// archguard router-seam test enforces this, with the not-yet-migrated handlers
// pinned in its shrink-only allowlist.
//
// Entry points: [New] builds the [Router]; every wire method is a [MethodHandler]
// (one that also implements [MutatingMethod] counts as a document edit — read-only
// is the absence of that second interface, so there is no flag to drift from the
// handler set). Handlers are registered keyed on the api/wire method-name
// constants, never on literal strings — the wire module is the single source of
// truth for the surface (ADR-0018), and the router's parity guard keeps the two
// aligned.
//
// This is the same contract the future M16 gRPC api/ will serve; keeping it here and
// transport-agnostic lets the bridge retarget onto that layer with minimal churn.
package router
