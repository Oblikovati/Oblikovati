# Point placement recipe

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Reference: Go PointRecipe at aa8297d1b0ac33c024f19e54821536d76e8a447a.
Kernel/API pins remain v0.2.5/v0.158.0 with no local source override.

The shared placement graph can now describe a standalone point. Its member
refers to the graph's point index. Adoption exposes that defining point and
uses its existing ID as the member ID. It does not reserve a second identity
or create a second solver point. Curve members retain their existing path.

A visible point can also serve as a curve endpoint. Member order controls
visible entity order, while hidden defining points retain their existing
ownership rules. Duplicate visible members, invalid indices and a construction
role on a point are refused. Candidate adoption remains atomic on failure.

Four independent Go fixtures cover the origin, translated points, negative
coordinates and fractional coordinates. The full fixture set has 83 cases;
the previous 79 cases are unchanged. Tests compare graph structure and two
unconstrained degrees of freedom. They also check single-ID ownership, member
order, shared endpoints, Undo/Redo and recipe serialization. Separate tests
preserve finite coordinate bits, including signed zero and extreme magnitudes,
and reject nonfinite coordinates. Extreme inputs are authoring tests, not a
claim that every later solver or rendering operation accepts those scales.

This is the model part of point placement. The later
[mouse input step](implementation-point-input.md) adds transient point markers
and one-click completion through the shared placement path. Existing point
creation and committed point rendering keep their owned identities. This step
does not complete R3/R4 or the full port. Work is local only.

Cross-platform review checked explicit member order, checked local indices,
single permanent identity, finite coordinate validation and unchanged input
bits. No geometric predicate, tolerance, wire contract or kernel algorithm was
added. This work needs no change to the kernel priority list.

The full local `make gate` passed for all 13 crates, all targets and features,
including native display tests under Xvfb with software Vulkan. Formatting,
Clippy, build, tests, documentation tests, Markdown/link checks, SPDX, workflow
lint and duplication checks passed. Every crate exceeded 80% line coverage;
model coverage was 98.84%. The point recipe, geometry resolution, point
ownership and recipe adoption modules each reached 100%. Duplication stayed at
0.79% (587/74,219 scanned lines, 68 clone pairs), with no added duplicated lines
or pairs. Evidence: `/tmp/point-recipe-gate.log`. This result paragraph was added
after the gate and checked with `make docs-lint`; no code changed afterward.
