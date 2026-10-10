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
