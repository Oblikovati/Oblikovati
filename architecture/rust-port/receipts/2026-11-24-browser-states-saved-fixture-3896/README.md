# Named model states on the saved blank Part

The [initial table run](before/table-accept/cucumber.json) stopped after
opening the saved blank metric Part because the member creation step
was unbound. The harness now creates Capture State and Copy State through
the Model State browser menu and checks the active member after each
creation. The subsequent setup checks the existing Part and its members.

The first runs with member creation exposed a second harness problem.
The table setup and the single-member feature case clicked the Model State
branch when it was already expanded. The
[table](before-branch-reuse/table-accept/cucumber.json) and
[feature](before-branch-reuse/single-feature/cucumber.json) reports show
the resulting missing rows. The branch now stays expanded through both
setup and member switching.

| Native scenario | Published | Pinned |
| --- | ---: | ---: |
| Toggle the model-state edit scope | [13 steps](published/scope/cucumber.json) | [11 steps](pinned/scope/cucumber.json) |
| Cancel adding a property to the model-state table | [19 steps](published/table-cancel/cucumber.json) | [17 steps](pinned/table-cancel/cucumber.json) |
| Accept a property column in the model-state table | [17 steps](published/table-accept/cucumber.json) | [15 steps](pinned/table-accept/cucumber.json) |
| Create a feature while editing a single model-state member | [12 steps](published/single-feature/cucumber.json) | [10 steps](pinned/single-feature/cucumber.json) |

All 61 published and 53 pinned steps passed. The two table cases
and the single-member feature case gained R0 records. The binding inventory
fell from [80 unresolved steps](../2026-11-24-browser-view-saved-fixture-3896/published-bindings.json)
to [56](published-bindings.json). The setup binding is shared by other
published scenarios that still require additional steps and native runs.

The passing builds used application base `7d1c53777`, local harness
changes, and existing file-browser edits. The runs used Linux X11, Xvfb,
Openbox, and Vulkan Lavapipe. The
[published snapshot](../2026-11-24-model-browser-named-part-3896/snapshot.json)
contains the blank metric Part.

See [#3896](https://github.com/Oblikovati/Oblikovati/issues/3896).
