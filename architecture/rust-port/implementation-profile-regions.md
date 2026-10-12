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

The kernel analytic face classifier distinguishes an inner region from its
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
uses sampled wire polygons and an exact classifier near uncertain boundaries.
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
