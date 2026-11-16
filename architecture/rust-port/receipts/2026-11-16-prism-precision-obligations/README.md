# Conditional prism precision obligations (#3928)

The session-borrowed conditional sketch evidence can inspect an actual authored
sketch-prism feature. It resolves the feature using the current session
snapshot, requires its sketch identity to match, derives the root and segment
bounds, and checks that every selected curve identity has supported line
and direction evidence. The API returns named outstanding obligations, never
numeric bounds or a certificate. Missing features, mismatched sketches,
unsupported selections, and invalid root boxes decline. The result does not
admit an operation and does not change existing feature behavior.

For a source-based tolerance analysis, let `p` and `q` be true positions,
`p0` and `q0` the accepted positions, and `ep` and `eq` proven Euclidean input
error bounds. The triangle inequality gives both `|p-q| >= |p0-q0|-ep-eq`
and `|p-q| <= |p0-q0|+ep+eq`. Each term needs a directed bound. These are
conditional mathematical relations, not an operation admission result.

The host-pinned Martensite contract uses weld as a *minimum* distance for
distinct used vertices and a minimum line length. Sew is a *maximum* allowed
curve-end/vertex gap, not a universal threshold for independent sources.
Angular controls different input tests, including frame orthogonality and sweep
inclination. No one tolerance can be treated as a generic input-error budget.
The kernel reports maximum stored vertex, edge and face tolerances; its contract
does not state how those values compose with input uncertainty.

The profile converter builds the complete non-construction profile, including
curves outside the selected loop. It groups authored coincidence relations and
requires the coordinates in each group to have identical binary64 bits. The
selected-line diagnostic here does not prove that grouping or the full profile.
Those are distinct outstanding obligations. The source analysis and its
conservative inequalities are recorded in the companion contract audit.

Martensite issue [#494](https://github.com/Oblikovati/Martensite/issues/494)
tracks the missing source uncertainty and tolerance composition contract. No
resolution comparison or feature admission is justified here.

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
