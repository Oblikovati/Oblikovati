# Conditional point displacement (#3928)

For a snapshot-bound projected root on an explicitly authored canonical origin
plane, derive a per-point upper bound on Euclidean displacement between the
retained binary64 solve center and any root in its enclosure. The directed sum
of the absolute coordinate differences bounds the Euclidean norm. Canonical XY,
XZ, and YZ frames preserve lengths exactly; caller-assumed or otherwise
unproven frames are declined. The point identity and coordinate order are
checked against the retained sketch. The existing root check supplies the
parameter, projection, solve, and snapshot conditions.

Independent exact rational squared-distance tests cover interval corners and
the solved rectangle in all three canonical planes. Non-finite and overflowing
interval inputs are refused. Existing root tests cover stale same-value sources,
changed projections and frames, singular and uncertain ill-conditioned systems.
A zero-width input also tests a degenerate point displacement; this does not
establish validity for degenerate lines or curves.

This is conditional evidence in document length units. It is not an edge error,
angular error, model tolerance, or production qualification. Point displacement
is not itself a weld or sew operand pair, so comparing it to either limit would
require a separately justified error budget and operand provenance. The code
therefore makes no such comparison. Noncanonical frame propagation, general
curve and angular bounds, conditioning and result-tolerance policy, platform
arithmetic guarantees, and persistent document-lifetime binding remain open.
No feature or extrusion gate was changed.

Checks (Linux):

- `CARGO_TARGET_DIR=/tmp/obk-3928-spatial-target CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 cargo test --quiet --locked -p oblikovati-model --lib --test associative_point_projection --test document_sketch`
- `CARGO_TARGET_DIR=/tmp/obk-3928-spatial-target CARGO_PROFILE_DEV_DEBUG=0 CARGO_INCREMENTAL=0 cargo clippy --quiet --locked -p oblikovati-model --lib --tests -- -D warnings`
- `cargo fmt --all --check` and `git diff --check`

The full repository gate and OS-level GUI tests were not run for this draft.
