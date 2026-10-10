# ADR-0002: Public automation and add-in hosting

Date: 2026-10-10. Status: Accepted for the Rust target design.
Implementation evidence: pending R2/R4 and the API release gate.

Boundary: public calls, session dispatch, and native add-in lifecycle.
Owner: `oblikovati-rust`; Oblikovati.API owns the shared contract.

## Context

Existing add-ins use a C ABI and typed JSON clients. Their Go runtime does not
share Rust's ownership model. The API now provides both Go and Rust definitions;
the host must use that contract without declaring a second public surface.

## Decision

Import `oblikovati_api::{types, contract, wire, client, ffi}`. Implement the
applicable traits and wire operations against session commands. Import public
DTOs, numeric IDs, method names, and C signatures. Keep extensions contract-first
in Oblikovati.API, including generated Rust definitions, clients, examples for
both languages, and its parity/release checks. This ADR does not take ownership
of the generator or API version policy.

Use the existing C ABI for native Go and Rust add-ins. Keep JSON as its payload
and the typed client as the add-in entry point. gRPC is a possible later
transport, with no implementation or compatibility promise in this port.
Built-in registration follows [ADR-0001](ADR-0001-rust-application-architecture.md).

The automation router classifies each call as query or edit once. All edits
use the session transaction path, including CLI and script edits. Preserve
errors, null/omitted fields, event order, and single undo recording. Publish
the implemented method inventory; a version match alone does not certify a
partially ported host.

## Interfaces and invariants

- Use the canonical C `int` lengths and status codes. Validate negative lengths,
  null/length combinations, size conversion, UTF-8 where required, and payload
  shape before making Rust references. Copy borrowed requests before queueing.
- Host responses, including error buffers, are freed only by `ObkHostFree`.
  Copy optional automation responses, then call the add-in's `ObkFree`.
  Add-in ID/manifest strings remain add-in-owned until deactivation.
- C callbacks carry no context argument. A minimal callback registry must be
  tied to host lifetime and contain a dispatch endpoint, not model state.
  Reject calls after shutdown. Do not invent a context parameter in this ABI.
- Do not unwind a Rust panic through C. Contain recoverable host panics before
  return; this cannot contain process aborts or faults in a foreign add-in.
  In-process libraries remain trusted code with no crash isolation.
- Retain the current handshake: required version exports, matching major,
  add-in minor no newer than host minor. Preserve the checks even during API
  0.x; do not claim that this alone proves compatibility for every older release.
- A caller timeout after enqueue does not cancel an accepted model mutation.
  Preserve this Go behavior and never retry a timed-out edit automatically.
  Recompute cancellation is a different operation, owned by
  [ADR-0004](ADR-0004-session-and-feature-evaluation.md).
- Keep the session queue bounded. The CLI supplies an owner/drain loop too.
  Host-to-add-in callbacks must not synchronously reenter a session call that
  waits on the same owner, including `ObkAddInAutomation`.
- Shutdown stops admission, resolves pending waiters, stops notifications, and
  deactivates add-ins while required dispatch remains available. Join work and
  retire callback state before release. Go shared libraries can retain runtime
  threads after deactivation; keep activated libraries mapped for process
  lifetime unless safe unloading is proved for that library class.

## Alternatives and consequences

Rust traits or pointers across shared libraries would replace the stable C
contract with a language ABI and invalid lifetimes. A new gRPC service would
delay compatibility with installed add-ins. Both are rejected for this port.
Keeping in-process libraries preserves current workflows but also preserves
their crash and reentrancy risks; a timeout cannot kill foreign code safely.

## Legacy disposition

Consolidate Oblikovati ADR-0003, 0016, and 0018 plus the native-call parts of
0008 at the [fixed revision](legacy-map.md#source-baseline). Retain the C ABI
and shared Apache contract. Omit Go `init` and the obsolete protobuf source of
truth. Use the actual header and loader instead of early Windows-stub claims.

## Validation required

Evidence inputs: API `include/oblikovati_addin.h`, `rust/src/ffi.rs`, its
`Makefile`/`RELEASING.md`; Go `addin/dispatch/queue.go` and
`head/internal/addinhost/`. R4 needs real Go and Rust test libraries plus
MCPBridge: ownership/free checks, malformed requests, version rejection,
notifications, queue timeout, reentrancy, shutdown, and repeated lifecycle
tests. Run both typed clients against the same document workflow. Do not
replace these checks with serialization-only tests.
