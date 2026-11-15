# Extrusion undo and redo diagnostic

On 2026-11-15, the pinned `extrusion-06 Accepted creation supports Undo and
Redo` run stopped at its undefined setup step. It did not exercise feature
acceptance, history, or the two edge measurements. This is a diagnostic, not a
passing scenario receipt. The application commit was `3f2db44d7`; the scenario
snapshot and its unchanged-source check are recorded in `source-revision.json`.

The current host Measure path supports face area, while this scenario asks for
solid-edge length. The application gap is tracked in
[#3967](https://github.com/Oblikovati/Oblikovati/issues/3967). Any required
kernel capability must be reported to the Martensite owner. This run injects
input through the native frame loop and does not prove OS-level input.
