# Prism input tolerance contract audit (#3928)

The Rust host pins Martensite commit
[`d051ff699982faba4ec2df3ee9a47e14f38dedbd`](https://github.com/Oblikovati/Martensite/tree/d051ff699982faba4ec2df3ee9a47e14f38dedbd).
This analysis is limited to that revision; it does not assert the current
kernel branch has the same contract.

[`Resolution`](https://github.com/Oblikovati/Martensite/blob/d051ff699982faba4ec2df3ee9a47e14f38dedbd/crates/martensite-tkbo/src/profile_extrusion/resolution.rs)
defines `weld` as a minimum separation of distinct used vertices and minimum
curve length, `sew` as a maximum admitted curve-end/vertex gap, and `angular`
as an admission threshold. The
[admission implementation](https://github.com/Oblikovati/Martensite/blob/d051ff699982faba4ec2df3ee9a47e14f38dedbd/crates/martensite-tkbo/src/profile_extrusion/admission.rs)
compares each line length and each pair of distinct used vertices against weld.
The [sweep admission](https://github.com/Oblikovati/Martensite/blob/d051ff699982faba4ec2df3ee9a47e14f38dedbd/crates/martensite-tkbo/src/profile_extrusion/extrude.rs)
also uses weld as a minimum direction length and angular to reject directions
near the plane. Frame orthogonality has a different angular comparison. The
maximum stored B-rep vertex, edge and face tolerances are reported as
[`AchievedTolerance`](https://github.com/Oblikovati/Martensite/blob/d051ff699982faba4ec2df3ee9a47e14f38dedbd/crates/martensite-tkbo/src/profile_extrusion/validation.rs);
they are not documented as a total bound from uncertain authored inputs.

## Conditional mathematical margins

For independently enclosed true positions `p,q`, accepted positions `p0,q0`,
and Euclidean displacement bounds `ep,eq`, the triangle inequality gives
`max(0, |p0-q0|-ep-eq) <= |p-q| <= |p0-q0|+ep+eq`. With directed rounding,
a lower bound strictly greater than weld proves the mathematical separation
for that pair. The same reasoning applies to the two endpoints of one line. A
curve-end and vertex gap uses the upper bound, and that upper bound must be at
most sew. Shared topological identities must be handled as identities, not
classified through a distance budget. Angular direction error is in radians;
combining it with a length effect requires an appropriate bounded extent.

These inequalities are not a kernel-acceptance proof: the kernel evaluates
its own floating operations, the profile contains all non-construction
curves rather than only the selected boundary, and source identities and
correlations must match the profile that the kernel actually receives. The
current host converter groups coincident point identities transitively and
requires bit-identical coordinates for each group; unequal values, including
signed zero, decline without moving or welding points. The conditional
selected-line diagnostic does not prove this full-profile identity mapping.

The pinned kernel contract does not specify a general uncertainty input, the
rounding margin on its admission comparisons, or how input and construction
uncertainties combine with per-entity output tolerances and their cap. In
particular, neither weld nor sew is a freely available sketch error budget.
The host therefore cannot safely implement the requested general admission
budget from this contract. [Martensite issue #494](https://github.com/Oblikovati/Martensite/issues/494)
requests the kernel-owned contract and independent oracle cases. No kernel code
or existing admission gate is changed by this draft.

The host adapter incidence tests cover transitive identity grouping and refusal
of differing coordinates; the signed-zero case confirms the exact-bit rule.
The session feature assessment retains separate identity mapping, exact
coincidence, and tolerance-budget obligations. Checks (Linux):

- `CARGO_TARGET_DIR=/tmp/obk-3928-contract-target CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 cargo test --quiet --locked -p oblikovati-martensite --lib coinciden` (3 passed).
- `CARGO_TARGET_DIR=/tmp/obk-3928-contract-target CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 cargo test --quiet --locked -p oblikovati-session --test lifecycle` (461 passed).
- `CARGO_TARGET_DIR=/tmp/obk-3928-contract-target CARGO_PROFILE_DEV_DEBUG=0 CARGO_INCREMENTAL=0 cargo clippy --quiet --locked -p oblikovati-session -p oblikovati-martensite --lib --tests -- -D warnings`.
- `cargo fmt --all --check` and `git diff --check`.

No full release gate or GUI test is claimed.
