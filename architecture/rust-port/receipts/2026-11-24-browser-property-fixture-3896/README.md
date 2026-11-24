# Category property setup on the saved blank Part

The published runs for [cell edits](before/cell/cucumber.json),
[column removal](before/removal/cucumber.json), and
[category switching](before/staged/cucumber.json) opened the saved blank
metric Part and stopped at the unbound Category setup. The harness now
adds and accepts Category through the model-state table. It edits the
active member's cell when the setup requires Prototype. The assertions
resolve that cell by member ID and check the other members separately.

| Native scenario | Published | Pinned |
| --- | ---: | ---: |
| Accept and cancels edits to a model-state property cell | [22 steps](published/cell/cucumber.json) | [19 steps](pinned/cell/cucumber.json) |
| Cancel removal of a model-state property column | [19 steps](published/removal/cucumber.json) | [15 steps](pinned/removal/cucumber.json) |
| Retain a staged property column while switching table categories | [18 steps](published/staged/cucumber.json) | [14 steps](pinned/staged/cucumber.json) |

All 59 published and 48 pinned steps passed. Three pinned scenarios
gained R0 records. The published binding inventory went from
[56 unresolved steps](../2026-11-24-browser-states-saved-fixture-3896/published-bindings.json)
to [51](published-bindings.json).

## Native extrusion parameter gap

The [native extrusion probe](before/native-extrusion-parameter/cucumber.json)
created an undimensioned rectangle and accepted a positive distance of
`1 in` on the same saved Part. The model-state table's available parameters
did not include a feature-owned `d0` with that expression. The native Extrude
producer resolves a reference-free distance into a constant displacement.
The existing pinned parameter fixture supplies a pre-authored `d0`.
The published authoring cases still require verified parameter ownership.

The [published snapshot](../2026-11-24-model-browser-named-part-3896/snapshot.json)
contains the blank metric Part. The passing builds used base
`e78f553a3`, local BDD changes, and existing file-browser edits. The runs
used Linux X11, Xvfb, Openbox, and Vulkan Lavapipe.

See [#3896](https://github.com/Oblikovati/Oblikovati/issues/3896)
and [#4119](https://github.com/Oblikovati/Oblikovati/issues/4119).
