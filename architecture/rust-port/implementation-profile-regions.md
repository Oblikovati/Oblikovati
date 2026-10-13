# Released kernel region construction — partial K02/R3

Date: 2026-10-12. Dependency: Martensite v0.2.1,
`193572dc3665598c4948777c87375bad7dc8114a`.
Tracking: application [#3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
The full extrude port remains open.

## Verified integration

The new `profile_regions` integration test calls only released kernel geometry
operations. Fresh analytic edges have separate coincident endpoints, as real
sketch entities can. `BRepAlgoAPIBuilderAlgo` performs general fuse with
non-destructive operation and history enabled. Kernel pcurve construction and
`BuilderFace` then construct the planar regions from both edge orientations.
The application has no new intersection, arrangement, containment, or repair
algorithm. The added TKBO dependency is test-only and has the same exact release
pin as the other kernel packages. Its 357 source files byte-match the released
crate archive and pinned checkout.

Six inputs pass: rectangle, nested rectangle, interior divider, diagonal,
outer rectangle with a circle, and outer rectangle with an ellipse. Every one
of the eleven bounded faces passes the released geometric B-rep checker.
Each face's analytic area, centroid, wire count, and edge count is checked
against a separately compiled OCCT 8.0.0 result. Input edge history covers all
source edges. The divider splits five original edges into seven and retains
the distinct four-parent boundary sets for its two regions.

The released face classifier distinguishes an inner region from its
surrounding region, rejects an outside seed, and reports both incidences for a
seed on their shared boundary. No ordered first-match selection is introduced.
This is a test of kernel results, not a persisted application selection policy.

## Verified limits and next work

General fuse rejects a single closed circle or ellipse with
`BOPAlgo_AlertTooFewArguments` in both OCCT and the Rust release. These are valid
sketch profile inputs. The negative test records an integration limit; it does
not accept the missing application behavior or close K02. Do not add an
application branch that bypasses the kernel pipeline for one-curve sketches.
The kernel owner needs one general sketch-region contract that accepts these
inputs along with the multi-curve inputs and supplies their provenance.
An owning-kernel work item/ADR and its tested release remain prerequisites.

`BRepTopAdaptorFClass2d` is used here to identify bounded test faces. Its source
uses sampled wire polygons and a curve classifier near uncertain boundaries.
This evidence does not prove the stronger application requirement for exact or
filtered topology predicates. That policy belongs in the kernel's region
contract; do not promote the diagnostic fixture classifier into an application
modeling algorithm.

Persistent names, arbitrary curved intersections, spline/arc inputs, generated
near-degenerate cases, non-default units and planes, achieved tolerance,
self-intersection checks, profile seeds across edits, complete extrude extents,
and production feature/API integration remain open. The test parent indices
identify only fixture inputs. They are not persistent reference keys. The
current application continues to use its explicit directed-line prism recipe.

## Reproduce locally on Linux

```sh
g++ -std=c++17 -I../occt-install/include/opencascade \
  crates/martensite/tests/profile_regions/occt_oracle.cpp \
  -L../occt-install/lib -Wl,-rpath-link,../occt-install/lib \
  -lTKBO -lTKBool -lTKTopAlgo -lTKBRep -lTKGeomBase -lTKG3d -lTKMath -lTKernel \
  -o /tmp/oblikovati-profile-regions-occt-oracle
LD_LIBRARY_PATH=../occt-install/lib /tmp/oblikovati-profile-regions-occt-oracle
LD_LIBRARY_PATH=../occt-install/lib /tmp/oblikovati-profile-regions-occt-oracle --declines
CARGO_NET_OFFLINE=true cargo test --locked -p oblikovati-martensite --test profile_regions
CARGO_NET_OFFLINE=true make gate
```

The retained CSV files contain the independent successful-face and named-decline
results. No native OCCT dependency enters the production application. Tests
run only on this Linux host; no remote job, push, or PR is part of this work.

## Local validation receipt

The four region integration tests passed. The OCCT generator reproduced both
retained CSV files byte-for-byte. `CARGO_NET_OFFLINE=true make gate` passed for
the complete twelve-crate application workspace, including native Linux display
tests, formatting, clippy, tests, doctests, per-crate coverage, documentation,
workflow validation, SPDX, and duplication checks. Duplication was 0.59%.
This receipt verifies the integration probes and existing application regression
suite; it does not close the remaining K02/R3 requirements listed above.

## Nonempty-input extension experiment, 2026-10-13

Kernel work item [RustyCADKernel#10](https://github.com/Oblikovati/RustyCADKernel/issues/10)
now owns the production sketch-region prerequisite for R3/R5. Source inspection
of kernel main `ede3518dbd7c2ae5e49d54cfef3e5bcf9029ae9e` confirms that the
interference stage admits one input, while default `BOPAlgoBuilder::CheckData`
requires two. The runtime experiment still uses the application's pinned v0.2.1;
this is not a current-main runtime receipt or a dependency update.

A test-only `BuilderHooks` implementation retains the filler report and replaces
the two-argument precondition with the filler's nonempty-input contract. Every
case uses the same interference, image, history, and post-treatment pipeline.
There is no circle-specific branch, bypass, perturbed geometry, or retry. An
independent OCCT subclass changes only `CheckData` to nonempty + `CheckFiller`.
Both produce the same 13 bounded-face reports for the original six inputs plus
single-circle and single-ellipse inputs. Each face passes native geometric
validity and area/centroid/wire/edge assertions. Every returned boundary edge has
an original source parent. Empty input remains a named decline with no shape.

The original mapped-builder refusal tests remain unchanged. The existing OCCT
success and decline files reproduce byte-for-byte after extending the generator.
Run its new `--nonempty` mode for `occt-8.0.0-nonempty.csv`, then run
`cargo test -p oblikovati-martensite --test profile_regions` for all six tests.

This establishes a viable extension hook, not a production region contract.
Sampled bounded-face classification remains diagnostic. Exact/filtered topology
predicates, complete validation and measured tolerance, ownership, cancellation,
persistent split identity, seed ambiguity, and a tested owner release remain
required by issue #10. No production application or kernel code changes in this
experiment. Do not expose `features.add` by silently reducing these requirements.

## Crossing curves and region-to-solid evidence, 2026-10-13

The released-kernel diagnostic corpus now includes eight more inputs:
rectangle/circle and rectangle/ellipse crossings, a circle crossed by a divider,
overlapping circles inside a rectangle, a circle tangent to a rectangle, and
pairs of separated, tangent, and nearly overlapping unit circles. The last
three use center distances 2.0001, 2, and 1.9999. These are explicit finite
fixtures, not a proof over arbitrary near-degenerate inputs or scales.

All 23 bounded regions pass native geometric validity and per-region OCCT
area, centroid, wire-count, and edge-count assertions. Every source edge remains
covered by operation history. Each region's complete fixture parent set is
asserted, including repeated parent sets for distinct regions. In particular,
all three regions of two overlapping circles have the same two source parents.
This is evidence that persistent split identity needs more than a parent set.

Each region also goes through the same released prism builder with displacement
`(0,0,2)`. All 23 results are solids and pass native geometric validity. All 123
faces match the independent OCCT result for analytic area, centroid, surface
type, wire count, and edge count; volume is an additional smoke test. Prism
history assigns every face exactly one start-cap, end-cap, or source-edge role.
This is operation history, not persisted application reference-key acceptance.

Oracle comparisons retain the diagnostic absolute bound of `1e-8` in these
fixture units. Mathematically equal areas can have different rounding order in
the independent integrators. Comparison therefore requires a unique geometric
match and checks that every expected face is used once. It never uses geometric
matching to assign production identity. These tests do not measure achieved
modelling tolerance or certify exact analytic-curve incidence.

The C++ oracle has `--curved` and `--curved-prisms` modes; link `-lTKPrim` in
addition to the libraries in the earlier command. The five retained CSV files
reproduce byte-for-byte, including the three earlier files. All seven
`profile_regions` tests pass. The generator now shares bounded-region
construction between its region and prism reports; Rust diagnostics share the
same bounded-face selector too. No second production algorithm was added.

Martensite local main now contains exact 2D orientation in commit `90df47a70`,
with its owning ADR 0003. This application still consumes v0.2.1 and does not
use that unreleased primitive. The sampled diagnostic classifier remains
ineligible for production. General analytic bounded-region classification,
arc/spline coverage, ownership and cancellation contracts, complete validation,
measured tolerances, persistent naming, and a tested owner release remain open
under kernel issue #10. No feature API or production extrude scope is claimed.

The final `make gate` passed for the complete discovered application workspace
(13 crates), including native Linux display tests on Xvfb with lavapipe, build,
lint, coverage, documentation, SPDX, and duplication checks. Per-crate coverage
remains above 80%; duplication is 0.69%. An earlier gate invocation was stopped
because startup warnings were mistaken for the Xvfb display number. The final
run used the confirmed display `:2`. No test failure was accepted or skipped.
No push, PR, dependency update, or production geometry change is part of this work.

## Exact polynomial root prerequisite, 2026-10-13

Martensite local main commit `d983a7383` adds exact polynomial root isolation
under ADR 0004. It returns ordered, disjoint closed dyadic brackets and exact
multiplicities for supplied binary64 coefficients. Versioned degree and work
limits, cancellation, and typed failure keep incomplete results out of success.
The implementation uses one integer Sturm path without a floating fallback.

The retained independent Jacobian corpus has 22 inputs and 47 distinct roots.
Tests cover repeated roots, near-tangencies, all finite binary64 exponent
fields, roots below the smallest subnormal, maximal bounds, cancellation,
work limits, and input/result ownership. The complete TKMath suite passes
2,793 tests with none skipped. New root modules have 98.97% line coverage and
0% detected duplication; lint, Rustdoc, formatting, and documentation checks pass.
The kernel validation record retains the exact scope and benchmark receipt in
`docs/exact-polynomial-roots.md`.

This primitive certifies supplied coefficients. Exact analytic-curve conversion,
rational denominators, periodic trims, incidence ownership, complete region
validation, measured tolerances, and persistent split naming remain open under
kernel issue #10. The application still pins released Martensite v0.2.1. This
step does not enable general production extrusion or change dependency pins.
Validation above covers the kernel change; the application change is this
record only and is checked with `make docs-lint`.

## Exact Bernstein conversion prerequisite, 2026-10-13

Martensite local main commit `8c764eb99` adds scalar Bernstein input to the
same exact root solver under ADR 0005. Integer forward differences and binomial
factors preserve the supplied controls without rounded power coefficients.
The query interval does not rescale the [0,1] basis. Degree admission follows
exact reduction, so elevated linear and constant representations retain their
true degree. Existing power results remain unchanged.

The retained independent corpus adds 20 inputs and 38 distinct roots. Two
counterexamples prove that floating conversion can create a false tangent or
merge separate crossings. Other tests cover endpoint multiplicity, reversal,
degree elevation, every finite exponent field, roots below the binary64
parameter range, cancellation, and source/result ownership. The complete
TKMath suite passes 2,806 tests with none skipped. Shared root-module coverage
is 98.91%, including 100% for the new conversion module; duplication is 0%.
Lint, documentation, and formatting checks pass. Detailed evidence and the
combined benchmark are in kernel `docs/exact-bernstein-roots.md`.

This certifies supplied scalar controls, not rounded point/weight products,
rational denominator poles, spline knot extraction, or complete geometric
intersections. Those constructions and the region acceptance requirements remain
open under issue #10. The application dependency pin and production extrusion
scope are unchanged. This application record passes `make docs-lint`; the
runtime validation above is for the kernel change.

## Exact Bézier-to-line parameter prerequisite, 2026-10-13

Martensite local main commit `b0ebc2cf3` adds a kernel-owned parameter query for
planar polynomial/rational Bézier curves against an infinite line, under ADR
0006. Coordinates and weights enter exact integer arithmetic before differences,
determinants, and products. The existing Bernstein conversion and root solver
then certify denominator regularity and numerator contacts. Denominator zeros
are typed failures even when a common factor could cancel them. Coincidence
is distinct from completed empty and isolated-contact results.

The retained independent corpus contains 12 curves, 17 nonconstant Jacobian
queries, and 13 certified contacts in [0,1]. Six ordinary cases also match
OCCT 8.0.0 counts and parameters; the native receipt reproduces byte-for-byte.
Generated scale checks span all finite exponent fields. Other cases cover
projection overflow/underflow, rounded false coincidence, signed/zero weights,
endpoint and repeated denominator zeros, cancellation, and source/result
ownership. The complete TKMath suite passes 2,823 tests with none skipped.
New query-module coverage is 99.43%; predicate duplication is 0%. Lint,
Rustdoc, formatting, and documentation checks pass. Kernel
`docs/exact-bezier-line.md` records scope, commands, and benchmark evidence.

This query returns parameters, not edges or regions. Finite-line clipping,
ray-side selection, periodic spline extraction, curve/curve intersections,
complete region validation, achieved tolerance, source history, and persistent
split naming remain required under issue #10. The application still consumes
released Martensite v0.2.1; no local kernel source or native OCCT fallback is
added. General production profiles and extrusion remain incomplete. This
application record passes `make docs-lint`; runtime evidence above belongs to
the kernel change.

## Exact algebraic-root sign prerequisite, 2026-10-13

Martensite local main commit `488a7a4e4` adds exact signs of a query polynomial
at every isolated source root under ADR 0007. The shared signed remainder
sequence implements the Sturm–Tarski query; point roots use exact substitution.
Source brackets, multiplicities, and subdivision counts remain unchanged.
A zero query proves equality at every source root; a zero source still declines.
Errors retain their stage and original cause, with no successful partial list.

The retained independent corpus has 20 polynomial pairs, 40 root-isolation
results, and 75 exact algebraic comparisons establishing 42 source-root signs.
Tests include shared irrational roots, repeated roots, wrong midpoint signs,
two query roots inside one source bracket, extreme values, cancellation, and
input/result ownership. Four ordinary cases also compare signs against retained
OCCT parameters away from query zeros; the six-row OCCT receipt reproduces after
rebuilding its generator. The full TKMath suite passes 2,840 tests with none
skipped. New sign-module coverage is 98.26%; predicate duplication is 0%.
Lint, Rustdoc, formatting, and documentation checks pass. Kernel
`docs/exact-root-signs.md` records the evidence and measured benchmark.

This is a numerical prerequisite for directed Bézier/ray selection. That
consumer must preserve exact projected integer coefficients and denominator
signs, then handle origins, endpoint ownership, tangencies, and overlaps.
Complete region validation, achieved tolerances, source history, and persistent
split naming remain required under issue #10. The application still pins
released Martensite v0.2.1; production profiles and extrusion are unchanged.
This application record passes `make docs-lint`; runtime checks above cover the
kernel change.

## Exact directed Bézier contacts, 2026-10-13

Martensite local main commit `6b0c27bd9` connects exact curve projection to
algebraic-root signs under ADR 0008. The ray query retains every isolated
supporting-line contact and classifies it behind, at, or ahead of the origin.
It certifies the sign of the exact longitudinal numerator times the regular
denominator. Brackets, multiplicities, and shared subdivision counts remain
unchanged. Supporting-line coincidence is a named decline; isolated roots do
not represent continuous overlaps. Denominator failures retain precedence.

The independent corpus retains 16 curves, 36 exact root-isolation results, and
79 algebraic comparisons establishing 18 contact sides. It includes common
irrational roots and adjacent binary64 origin offsets where a bracket midpoint
gives the wrong side. Six ordinary cases match native OCCT 8.0.0 sides; the
previous native parameter receipts reproduce byte-for-byte. Generated tests
cover every finite binary64 exponent field, weight sign, direction reversal,
cancellation, and source/result ownership. The full TKMath suite passes 2,857
tests with none skipped. The two new ray modules have 100% line coverage;
predicate duplication is 0%. Clippy, strict Rustdoc, formatting, documentation,
and SPDX checks pass. Kernel `docs/exact-bezier-rays.md` records the commands,
limits, and measured costs. This application record passes `make docs-lint`.

Overlap intervals, wire endpoint ownership, winding, full region validation,
achieved tolerances, source history, and persistent split naming remain under
issue #10. The application still pins released Martensite v0.2.1; this local
kernel increment does not change production profiles or extrusion. The runtime
evidence above covers the kernel, not a new application live test.
