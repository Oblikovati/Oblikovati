# Model-state suppression on the saved solid Part

The [initial published case](before/cancel/cucumber.json) opened the
named `workspace-layout-native.opd` Part and stopped when it reached the
unbound suppression setup. The fixture now adds the saved extrusion to
the native model-state table, sets Primary and Capture State to Suppress,
and verifies that Copy State remains Compute. Later steps use the actual
extrusion and verify its browser and viewport presentation.

| Native scenario | Published | Pinned |
| --- | ---: | ---: |
| Switch between states with different feature suppression | [15 steps](published/switch/cucumber.json) | [11 steps](pinned/switch/cucumber.json) |
| Cancel a staged model-state suppression change | [17 steps](published/cancel/cucumber.json) | [13 steps](pinned/cancel/cucumber.json) |
| Apply a feature suppression change through the model-state table | [20 steps](published/apply/cucumber.json) | [16 steps](pinned/apply/cucumber.json) |
| Apply an inserted model-state table row | [14 steps](published/insert/cucumber.json) | [10 steps](pinned/insert/cucumber.json) |

All 66 published and 50 pinned steps passed. The four pinned scenarios
gained R0 records. The inventory dropped from
[51 unresolved published steps](../2026-11-24-browser-property-fixture-3896/published-bindings.json)
to [43](published-bindings.json).

The [published snapshot](../2026-11-24-model-browser-named-part-3896/snapshot.json)
contains the saved native solid. The runs used build changes based on
`e78f553a3`, including the Category setup and existing local file-browser
edits. The published cancellation case passed again on the final build
based on `16a6cd196`. The tests ran on Linux X11, Xvfb, Openbox and
Vulkan Lavapipe.

See [#3896](https://github.com/Oblikovati/Oblikovati/issues/3896).
