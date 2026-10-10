# Rust implementation evidence

## Parameter dependency graph — R2

Tracking: [#3561](https://github.com/Oblikovati/Oblikovati/issues/3561).
Implementation: `crates/model/src/dependency`.
Go source: `model/param/graph.go` and `graph_test.go`, revision
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The model crate now supports caller-owned stable IDs, insertion-order edge
queries, atomic edge replacement, cycle rejection, protected removal, and
ordered recomputation of exactly the affected nodes. Equal-priority nodes use
insertion order. Traversal is iterative to support deep imported recipes.

Tests cover chains, diamonds, independent branches, duplicate edges, replacement,
unknown IDs, cycles, removal, cloned graph isolation, and a 10,000-node chain.
An independent reachability matrix checks all 64 forward-edge graphs on four
nodes and every single-driver replacement. Run `make gate` from the repo root.

The graph does not own parameter expressions, IDs, health, or feature evaluation.
The Go parameter collection retains a failed authored expression and marks its
parameter sick after a cycle. That owner-level behavior remains to be ported;
this graph rejects invalid edge edits without changing existing edges. The future
owner must explicitly detach failed bindings when required by the Go behavior.
Owner-cascade deletion and its health propagation also remain unimplemented.

This is partial R2 evidence. It is not evidence for document compatibility,
sketch solving, API conformance, geometry, GUI operation, or a completed milestone.

## Kernel package ownership — R0 dependency

Tracking: [kernel issue #3](https://github.com/Oblikovati/RustyCADKernel/issues/3).
The kernel release tooling is integrated into its local `main` branch. It creates
one versioned `.crate` archive per workspace package, plus the lockfile, toolchain,
license, shared test support, manifest, and checksums. It requires owner CI and
extracted-package tests in default and all-feature configurations before it can
publish an immutable stable `vX.Y.Z` release.

The owner verification record is
`RustyCADKernel/porting/integration/2026-10-10/release-packages.json`.
It records 31 packages at version 0.1.0, 16,748 tests and 1,478 doctests passed in
each feature configuration, with 106 documentation tests explicitly ignored.
It names the checked source revision and package hashes. Concurrent later kernel
work is not covered by that receipt. Release CI must check the final tagged tree.

No kernel release was published by this work. Keep the application kernel pin
empty until the published release, tag commit, and all consumed crate assets
pass the application preflight. The application does not take ownership of the
kernel's unit tests, oracle tests, or package publication.

## Parameter quantity arithmetic — partial R2

Tracking: [issue #3564](https://github.com/Oblikovati/Oblikovati/issues/3564).
`crates/model/src/quantity` ports `model/param/quantity.go` and `unit.go` from
Go baseline `aa8297d1b0ac33c024f19e54821536d76e8a447a`. Values stay in document
working units. Arithmetic returns a new value, checks category compatibility,
and rejects unnamed products or quotients and zero divisors.

The internal `Dimension` enum has no numeric API IDs or serialization encoding.
The API owns wire categories; its documented host mapping remains a later task.
For source parity, equal-category addition/subtraction and negation retain the
Go behavior even for flag/text categories. The future expression evaluator must
apply its own nonnumeric value restrictions. IEEE non-finite results are retained,
not clamped; parameter health validation is not implemented by this component.

An independent product table checks multiplication and inverse division for all
81 category pairs. Tests also cover addition/subtraction for all pairs, signed
zero, diagnostic operands, negation, and IEEE arithmetic. Parsing, named-unit
conversion, working-scale selection, persistence, and parameter health remain
open. This component does not complete R2 or provide an editable part.

## Named units and working-scale conversion — partial R2

Tracking: [issue #3566](https://github.com/Oblikovati/Oblikovati/issues/3566).
`crates/model/src/units` ports the 26 named-unit spellings, preferred-unit
conversion, literal scanner, scale exponents, and new-document recommendation
from Go `model/param/units_of_measure.go` at the fixed migration baseline.
Default storage remains centimetres; display preferences are independent owned
copies and cannot modify stored quantities. A suffix overrides the default
category; a bare number uses its category's preferred unit. Flag/text categories
retain the Go bare-number behavior until the expression owner validates them.

Scale construction rejects non-finite values and unrepresentable squared/cubed
scales, as well as non-positive values. This intentionally closes a Go validation
gap: its `<= 0` test admits NaN/infinity. Literal input rejects invalid or
non-finite numeric values. Arithmetic conversion order remains multiply then
divide, with explicit scale multiplication for area and volume.

Tests check every unit against physical values in metres, square metres, or cubic
metres across centimetre, picometre, micrometre, and kilometre working scales.
They also check preference conversion, clone isolation, unchanged stored values,
conditioning policy, malformed UTF-8-safe scanning, signed zero, and invalid
configuration. The numerical checks allow only eight machine epsilons of relative
rounding error; they are not geometric tolerances.

Display string rendering and precision/format preferences are not implemented.
The document owner must still freeze scale after geometry exists and restore
legacy missing-scale fields to the centimetre default. Expression parsing,
parameter health, document persistence, and the API boundary remain open.
