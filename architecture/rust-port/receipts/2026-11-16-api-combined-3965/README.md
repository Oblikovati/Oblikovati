<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Combined API candidate checks

On 2026-11-16, API commit `b48b923718a4248ebdc5113ee99a8f95731f7ccf` passed
`make gate` on Linux with separate Cargo and Go caches. The gate included the
Go and Rust suites, generated-source verification, C ABI tests, documentation,
and a full Go/Rust source duplication scan of 843 out of 82,181 lines (1.0258%).

`go test -cover ./...` reported at least 80.3% for each package with executable
statements. `cargo llvm-cov --manifest-path rust/Cargo.toml --locked
--summary-only` reported 97.77% Rust line coverage. The logs record the results.

This is a local candidate, not a published API release or hosted CI result.
API issue #187 host work and the Move policy clarification remain open. A
release must wait for all open API issues, the final combined gate, and hosted
checks on the exact release commit.

A disposable preview based on the same candidate bumped the API to `0.160.0`
with breaking scope, generated the bindings, and committed tree
`34afd3e141bc37f0176dfc59820800e560d959a8`. Its full gate and `cargo package`
passed. The crate SHA-256 was
`4c30b7010bf8e6cbc0a93f31d579922812f45453b8f905ec6baf51a7cdf0a4c4`.
The preview notes contain the source migration. This was not the final release
tree, and no tag or release was created.
