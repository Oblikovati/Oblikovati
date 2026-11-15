# Conditional prism precision obligations (#3928)

The session-borrowed conditional sketch evidence can inspect an actual authored
sketch-prism feature. It resolves the feature using the current session
snapshot, requires its sketch identity to match, derives the root and segment
bounds, and checks that every selected curve identity has supported line
and direction evidence. The API returns named outstanding obligations, never
numeric bounds or a certificate. Missing features, mismatched sketches,
unsupported selections, and invalid root boxes decline. The result does not
admit an operation and does not change existing feature behavior.

For a future tolerance budget, let `p` and `q` be true positions and let
`p0` and `q0` be the positions that an operation will consume. Given proven
Euclidean error upper bounds `ep` and `eq`, the triangle inequality gives
`|p-q| <= |p0-q0| + ep + eq`. Each term needs an outward-rounded bound in the
same document length unit. A shared authored point identity is a stronger
structural coincidence and must not be replaced by a numeric weld. If distinct
operands originate from the same computation, any allocated comparison budget
must use the operation's weld semantics. Independent source operands require
their source and frame bounds and the operation's sew semantics. A line angular
bound is in radians and cannot be added to either length budget; when compared
as an arc-length effect it also needs a proven extent and directed arithmetic.

These inequalities do not establish that the currently exposed weld or sew
limits allocate any budget to sketch solve error. The contracts do not state
which operand pairs each operation compares, how its achieved tolerance is
composed, or what fraction of the cap is available. Extrusion displacement
source uncertainty and cross-platform interval guarantees also remain open.
Therefore no resolution comparison or feature admission is justified here.

A future gate must cover profile discovery, preview, feature commit and edit,
ordered replay, and `retain_certified_sketch_prism` direct adapter calls. The
current diagnostic is called only by its explicit consumer and is not a gate;
existing paths can bypass it. Do not interpret its successful diagnostic return
as feature qualification.

A real session test creates a parameter-driven rectangle and prism, checks the
named obligations, changes its shared size, undoes/redoes, saves and reopens.
Existing root and segment tests use independent exact rational oracles. This
test also checks that a missing feature, invalid box, or selected point in place
of a line is declined. No GUI or
kernel operation is claimed by this test.

Checks (Linux):

- `CARGO_TARGET_DIR=/tmp/obk-3928-prism-target CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 cargo test --quiet --locked -p oblikovati-session --test lifecycle`
- `CARGO_TARGET_DIR=/tmp/obk-3928-prism-target CARGO_PROFILE_DEV_DEBUG=0 CARGO_INCREMENTAL=0 cargo clippy --quiet --locked -p oblikovati-session --lib --tests -- -D warnings`
- `cargo fmt --all --check` and `git diff --check`

The full repository gate and OS-level GUI tests were not run for this draft.
