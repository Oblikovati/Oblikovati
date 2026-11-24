# Copied Model Browser views on the saved solid Part

The [initial native run](before/published/hide-show/cucumber.json) opened
`fixtures/cad/workspace-layout-native.opd` and stopped at the unbound
copy step. The three published cases now use the Primary row's Copy
command to create and activate `BrowserView` on that Part. The harness
checks the active document, the single extrusion, and the visible solid
before running the viewport visibility and view-switch steps.

| Native scenario | Published | Pinned |
| --- | ---: | ---: |
| Hide and show a solid using the active view commands | [11 steps](published/hide-show/cucumber.json) | [9 steps](pinned/hide-show/cucumber.json) |
| Restore a locked view after temporarily hiding its solid | [13 steps](published/locked/cucumber.json) | [11 steps](pinned/locked/cucumber.json) |
| Restore an unlocked view after hiding its solid | [12 steps](published/unlocked/cucumber.json) | [10 steps](pinned/unlocked/cucumber.json) |

All 66 steps passed, and the three pinned scenarios gained R0 records.
The published binding inventory fell from
[90 unresolved steps](../2026-11-24-browser-view-fixture-3896/after-bindings.json)
to [87](published-bindings.json) across 100 scenarios. The
[published snapshot](../2026-11-24-model-browser-named-part-3896/snapshot.json)
contains the saved solid.

The runs used application base `985384313`, local BDD changes, and
existing local file-browser edits on Linux X11 with Xvfb, Openbox, and
Vulkan Lavapipe. The initial run used the same base before the solid-view
bindings.

See [#3896](https://github.com/Oblikovati/Oblikovati/issues/3896).
