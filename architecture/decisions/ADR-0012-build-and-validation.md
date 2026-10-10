# ADR-0012: Build and validation gates

Date: 2026-10-10. Status: Accepted for the Rust target design.
Implementation evidence: pending R0; this documentation branch has no build gate yet.

Boundary: evidence required to integrate and ship the Rust application.
Owner: application build/test tooling; dependency repositories own their gates.

## Context

The Go root test command did not compile all nested modules or the native head.
The Rust port must not repeat that gap or mistake passing source tests for
complete application behavior. Legacy ADR status is not implementation evidence.

## Decision

Provide one local command, `make gate`, with the same required checks as CI.
Discover the full Cargo workspace and any independent nested manifests from
the tree. Compare the discovered set with tested packages/targets. Include
native bindings, host test libraries, CLI, and GUI when they exist; a missing
tool, display, target, or suite must fail with its name rather than silently
skip a required check.

Pin the tested Rust toolchain, dependency revisions, lockfiles, native tools,
shader generator, and reference fixtures. Check clean builds without sibling
working-tree overrides. API generation and two-language contract parity stay
in Oblikovati.API; the application gate verifies its pinned output and host
method inventory. Martensite toolkit pins must identify one source revision.

Run formatting, warnings/lint, tests, architecture imports, API isolation,
Markdown/link checks, SPDX/notices, source coverage greater than 80%, and
duplication less than 3%. Measure coverage and duplication over a documented
scope; any exclusions must be visible. Include cross-platform hard-coded-value
and comment/coherence review before the first push.

Keep a behavior inventory linking each required capability to source, target,
milestone, and acceptance evidence. Classify a case as source present, tested,
demonstrated, or still incomplete. Counts of ported files, declarations, or
passing tests are not substitutes for feature coverage.

## Interfaces and invariants

- Unit and integration tests use meaningful outcomes and independent expected
  results. Named fakes cover external I/O; kernel fakes cannot certify geometry.
- Kernel work needs invariants, independent OCCT/SolveSpace comparisons where
  applicable, and generated degeneracies. Non-OCCT extensions need their own
  independent evidence under [ADR-0007](ADR-0007-martensite-integration.md).
- Compare IDs, canonical keys, ordered events, and other exact artifacts exactly.
  Test deterministic model/kernel output across runs and supported targets.
  Check FMA-sensitive arithmetic and dependency behavior; a no-op Rust cast is
  not a port of Go's rounding rule. Record any unresolved determinism gap.
- Numerical tolerances and GPU image bounds have explicit units and reasons.
  Renderer checks follow [ADR-0008](ADR-0008-vulkan-renderer.md). An image
  tolerance cannot weaken a topology or persistent-identity assertion.
- Before an implementation PR, use MCPBridge to stress the affected foreground
  application path and inspect screenshots and model diagnostics. Retain
  source/dependency revisions, inputs, logs, captures, and platform details.
- Full-gate success precedes the first implementation push. Batch necessary
  follow-up corrections. A documentation-only check is labeled as such and
  never reported as a successful runtime or full gate.

## Alternatives and consequences

Running only default workspace members would repeat the Go nested-module gap.
Skipping the head on a machine without its dependencies would hide a broken
desktop. Reject both. Full checks cost more and require a suitable machine;
smaller local loops remain useful but cannot replace required integration proof.

## Legacy disposition

Adapt Oblikovati ADR-0015 and validation aspects of 0014/0064 at the
[fixed revision](legacy-map.md#source-baseline). Replace GoReleaser,
golangci-lint, build tags, and Go arithmetic guards with tested Rust tooling.
Do not inherit the early zero coverage threshold, runtime stubs, or deferred
renderer checks as permanent exemptions. The full-module lesson in #3526
is retained in the gate discovery rule.

## Validation required

R0 demonstrates that adding a workspace member includes it automatically and
that a missing required tool/display fails clearly. Later milestones attach
their actual suites and receipts. The
[verification plan](../rust-port/verification.md) defines the platform matrix
and live corpus. This ADR migration checks Markdown, links, inventory coverage,
and record structure only; it provides no runtime acceptance receipt.
