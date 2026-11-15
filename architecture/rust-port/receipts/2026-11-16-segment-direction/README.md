# Conditional segment and direction bounds (#3928)

For each authored line on a source-proven canonical origin plane, compare the
accepted and enclosed segments at the same affine parameter `t` in `[0, 1]`,
using the original ordered endpoint identities. Convexity bounds the distance
by the larger endpoint displacement. The result is in document length units.

For direction, the sum of endpoint displacement bounds is an upper bound `e`
on the difference of direction vectors. The maximum absolute component of the
accepted vector is a lower bound `L` on its length. Directed interval arithmetic
checks `2e < L`, excluding zero length and reversal; otherwise the line is
refused as degenerate or uncertain. The unit-vector difference is at most
`2e/L`, and the angle in radians is at most twice the unit-vector chord, so
`4e/L` is a conservative angular upper bound. This is deliberately loose.

The rectangle regression retains the four authored line and endpoint IDs and
compares equal-parameter positions using exact binary rational arithmetic. An
independent rational cross/dot slope bounds its angle; directed division has an
independent exact rational corner oracle. Tests also refuse zero length,
possible reversal, uncertain near-degeneracy, unbounded values, and an
unproven frame. Existing snapshot-root checks reject stale sources. The bounds
remain conditional on source uncertainty and the interval arithmetic backend;
no cross-platform arithmetic certification, tolerance budget, weld or sew
comparison, curve bounds, or feature admission is asserted.

Checks (Linux):

- `CARGO_TARGET_DIR=/tmp/obk-3928-segment-target CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 cargo test --quiet --locked -p oblikovati-model --lib --test associative_point_projection --test document_sketch`
- `CARGO_TARGET_DIR=/tmp/obk-3928-segment-target CARGO_PROFILE_DEV_DEBUG=0 CARGO_INCREMENTAL=0 cargo clippy --quiet --locked -p oblikovati-model --lib --tests -- -D warnings`
- `cargo fmt --all --check` and `git diff --check`

The full repository gate and OS-level GUI tests were not run for this draft.
