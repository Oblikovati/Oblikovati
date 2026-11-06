# Released Martensite v0.10 adoption

Date: 2026-11-06. Tracking: [#3832](https://github.com/Oblikovati/Oblikovati/issues/3832).

## Release boundary

The renderer and authored sketch geometry graph targets
[v0.10.0](https://github.com/Oblikovati/Martensite/releases/tag/v0.10.0), peeled
commit `59bc4bda93e9b17100017608296bf053e10e5cbf`. All 15 locked Martensite
packages use this immutable source. The source regression rejects v0.9.0 for
renderer and sketch packages. The non-Martensite dependency graph is unchanged.

TKMath changes from 0.3.6 to 0.3.7 and TKTopAlgo from 0.4.1 to 0.4.2. The
remaining consumed package versions are unchanged. Component version numbers
are distinct from the SDK bundle version.

The full upstream tag diff contains 547 files. The review checks the changed
files in the consumed SDK graph: TKMath gains an optional absolute integration
floor, and the default geometric volume path scales that floor by bounding-box
length to the dimension of each integrated quantity. Explicit GK volume queries
retain the relative stop. The default math floor remains zero. New TKFeat and
TKOffset packages are not linked by this change, and their publication does not
by itself establish an application feature contract.

## Qualification scope

The combined candidate retains General Dimension, sketch Visibility, Fix and
continuous Line in addition to the previous native authoring integration. Each
feature keeps its commit history. The new integration PR targets `rust`; prior
stack bases stay stable until that combined state is qualified and merged.

Qualification uses the full local gate, portable host/GUI checks and native X11
mouse/keyboard acceptance with Go MCPBridge reads and inspected captures. The
rectangle path checks one size parameter driving two dimensions, horizontal,
vertical and coincident constraints at DOF zero, size edits, undo/redo, save and
fresh-process reopen. The Line path checks preview, closure, Fix, history and
reopen. Hosted platform checks remain required on the exact published head.

The production solid adapter retains its separately qualified baseline.
[Martensite#386](https://github.com/Oblikovati/Martensite/issues/386), skew volume,
[#388](https://github.com/Oblikovati/Martensite/issues/388), ARM64 Vulkan runtime,
and [#390](https://github.com/Oblikovati/Martensite/issues/390), analytic sketch
regions, remain separately tracked prerequisites. The complete cube/Extrude and
named-face journey remains unfinished. No fallback renderer is added.

## Local results

The full `make gate` passed across all 13 crates, including 354 head tests,
formatting, lint, documentation and SPDX checks. Coverage ranges from 91.96%
to 99.60%; duplicated lines are 1.00%. Six additional cross-target Clippy checks
passed: viewport on Windows x64/ARM64 and macOS Intel/ARM64, and host/GUI on
Windows x64 and macOS ARM64. These are compile checks, not native desktop runs.

Native X11 acceptance passed for a 10 mm rectangle driven by `size`, changed
to 20 mm, with both dimensions retaining that expression and DOF zero through
undo/redo and save/reopen. Constraint reads confirm two horizontal, two vertical
and one coincident constraint. Fresh-process reopen preserves the dimensions.

Continuous Line acceptance passed for preview, incomplete placement cancel,
Enter, Backspace, closure, Fix, undo/redo and persistence. Fresh-process reopen
preserves the three lines, entity IDs, three Fix constraints and DOF zero.
MCP captures were inspected. All four acceptance desktops closed normally; no
application or Xvfb process remains. Hosted checks must qualify the published
commit before merge.
