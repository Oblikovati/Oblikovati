# Planar profile and prism integration — partial K02/K04/R1

Tracking: [#3563](https://github.com/Oblikovati/Oblikovati/issues/3563), under
[ADR-0007](../decisions/ADR-0007-martensite-integration.md).
This is application acceptance evidence against the released kernel. R1 stays
open and no production body feature is published by this probe.

## One profile construction path

The adapter now accepts an ordered outer line loop, oriented inner line loops,
and a displacement in working units. Each segment has a unique nonzero authored
key supplied by the caller. Kernel polygon, face, and prism builders perform
construction. The adapter does not triangulate a profile, repair coordinates,
classify polygon containment itself, or switch to a different engine for holes,
negative displacement, or skew displacement.

Inputs require finite XY vertices, matching segment keys, at least three
vertices per loop, and finite displacement with a nonzero normal component.
A collapsed segment returns a named error carrying its source key. Invalid
profiles and result shapes fail the shared geometric checker. Self-crossing and
incorrectly oriented inner loops have refusal tests. Inner-loop winding is part
of the caller's oriented profile contract; the adapter does not guess or fix it.

The path consumes published Martensite v0.1.2 at
`19108f6650cdfca7050febc759d91bdfec001769`. The sibling kernel repository remains
unchanged. The existing box probe and the prism path now share geometric checks
and analytic volume queries. Neither query reads triangulation data.

## Face provenance and result checks

The prism maker supplies start/end caps and generated side faces for the input
edges. The report tags each side by its supplied edge key and caps by semantic
construction role. Every input edge must generate exactly one face. Each output
face must have exactly one role, and the role set must cover the complete output
face set. Missing and ambiguous coverage produce named refusals.

The report includes analytic volume and centroid, and per-face area, centroid,
and loop count. Face areas must be finite and positive; all reported coordinates
and volume must be finite. Reports contain no native shape handles. Supplied
edge keys and semantic cap roles are construction provenance, not completed
persistent reference keys or proof of the full generated/modified/deleted
contract. Result naming must still round-trip through evaluated feature state.

## Independent OCCT corpus and local tests

The retained C++ generator uses OCCT 8.0.0. It produces 216 per-face rows for 27
prisms: rectangle, rectangle with an inner loop, and a nonconvex L profile; three
working scales; and positive, negative, or skew displacement. Rust hard-asserts
each corresponding cap/side area, centroid, and loop count, along with body
properties. A whole-volume match alone cannot pass these tests.

```sh
g++ -std=c++17 -I../occt-install/include/opencascade \
  crates/martensite/tests/prism/occt_oracle.cpp \
  -L../occt-install/lib -Wl,-rpath-link,../occt-install/lib \
  -lTKPrim -lTKTopAlgo -lTKBRep -lTKGeomBase -lTKG3d -lTKMath -lTKernel \
  -o /tmp/oblikovati-prism-occt-oracle
LD_LIBRARY_PATH=../occt-install/lib /tmp/oblikovati-prism-occt-oracle
cargo test --locked -p oblikovati-martensite
make gate
```

Tests also check source immutability, repeat results, duplicate/zero keys,
malformed loop lengths, nonfinite input, collapsed near-coincident segments,
zero/in-plane displacement, invalid face values, and incomplete/ambiguous
history. These checks run only on this Linux host. The installed library version
is not a source-commit certification of the installed binaries.

## Remaining requirements

This step checks straight-segment planar profiles. Curved profiles,
sketch-driven profile extraction, general revolve, complete immutable operation
results, full validity levels, measured tolerance budgets, partial-result
diagnostics, persistent naming, model feature recipes, edit/replay, and
model-driven viewport publication remain open. The retained diagnostic prism and derived display mesh are checked
in [prism presentation evidence](implementation-prism-presentation.md). Required
Go-kernel behavior outside OCCT still must be ported and adapted in Martensite before its dependent features pass. No alternate
application geometry engine fills these gaps.

Placed profiles on validated model planes are checked in
[sketch-plane prism evidence](implementation-prism-placement.md).
