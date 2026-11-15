# Conditional generated preconditioner (#3928)

The projected-origin root check can derive an approximate inverse from the
midpoint of the accepted solve's interval Jacobian. Partial pivoting produces
only a candidate. The existing directed interval contraction and strict
inclusion checks decide whether to return conditional root evidence. Singular,
non-finite, nonsquare, or unprovable systems are declined. The rectangle test
uses the actual canonical XY origin, parameter sources, and retained solve.
An independent exact rational oracle checks its known coordinates and tests
matrix products. Additional tests exercise scaled matrices, singularity,
uncertain near-singularity, and adversarial candidate matrices.

A mathematically exact but poorly conditioned matrix can pass inclusion. This
result is not a conditioning or result-tolerance qualification. Source bounds,
cross-platform interval arithmetic guarantees, persistent document lifetime,
and operation admission remain separate unresolved requirements. No extrusion
gate is changed. Other unsupported equations and sources continue to decline.

Checks (Linux):

- `CARGO_TARGET_DIR=/tmp/obk-3928-auto-target CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 cargo test --locked -p oblikovati-model --lib --test associative_point_projection --test document_sketch`
- `CARGO_TARGET_DIR=/tmp/obk-3928-auto-target CARGO_PROFILE_DEV_DEBUG=0 CARGO_INCREMENTAL=0 cargo clippy --locked -p oblikovati-model --lib --tests -- -D warnings`
- `cargo fmt --all --check` and `git diff --check`

The full repository gate and OS-level GUI tests were not run for this draft.
