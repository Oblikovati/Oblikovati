# Conditional session evidence (#3928)

A session can derive conditional canonical-sketch evidence through a read-only
solve against its current snapshot. The current sketch must already be
converged; the newly solved candidate must have the same authored sketch recipe
bit for bit, or the request declines and requires an explicit recompute. The
fresh outcome is checked against its own derived snapshot and source graph. The
result owns that snapshot and outcome while borrowing the actual session, so it
cannot outlive or be reused across a session mutation. A compile-fail test
checks each restriction. This derivation never adopts the candidate.

The real workspace test builds a projected one-size rectangle, obtains its
root and line bounds, changes size, and rederives evidence after undo, redo,
save/reopen, a same-value expression change, a document switch, and save-copy
and reopen. The exact 1 cm and 2 cm corners are independently known binary64
values. Every derivation checks the authored sketch recipe, generation, undo
labels, and document dirty state are unchanged. A different document without the
sketch is refused. Evidence is re-solved after reopening; no certificate is
serialized or trusted from disk.

The wrapper does not make copied numeric bounds lifetime-bound; consumers must
not treat those numbers as reusable certificates. The actual feature pipeline
is not yet wired to this conditional evidence. A separate session regression retains a converged 1.01 cm endpoint under a
loose solve policy and confirms that the strict policy proposes a changed
candidate near the independent 1 cm target. It checks that the read-only request
declines and leaves evaluated state, recipe, coordinates, generation, ID high
water, dirty status, and nonempty undo and redo history unchanged. The proof still depends on
source and frame assumptions and platform arithmetic, and lacks conditioning and
model-tolerance budgets. No weld or sew comparison, production qualification,
or feature admission is asserted.

Checks (Linux):

- `CARGO_TARGET_DIR=/tmp/obk-3928-session-target CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 cargo test --quiet --locked -p oblikovati-session --test lifecycle`
- `CARGO_TARGET_DIR=/tmp/obk-3928-session-target CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 cargo test --quiet --locked -p oblikovati-session --doc`
- `CARGO_TARGET_DIR=/tmp/obk-3928-session-target CARGO_PROFILE_DEV_DEBUG=0 CARGO_INCREMENTAL=0 cargo clippy --quiet --locked -p oblikovati-session --lib --tests -- -D warnings`
- `cargo fmt --all --check` and `git diff --check`

The full repository gate and OS-level GUI tests were not run for this draft.

The candidate-change regression was also checked with
`CARGO_TARGET_DIR=/tmp/obk-3928-change-target CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 cargo test --quiet --locked -p oblikovati-session --test lifecycle`
(459 passed), followed by the corresponding session Clippy check.
