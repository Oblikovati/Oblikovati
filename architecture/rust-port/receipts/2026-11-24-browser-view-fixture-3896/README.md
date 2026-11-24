# Copied Model Browser views on the saved blank Part

The [initial native run](before/published/toggle/cucumber.json) opened
`fixtures/cad/blank-metric-part.opd` and stopped at the unbound copy step.
The harness now uses the Primary row's Copy command, names the new view
`BrowserView`, and activates it through the Model Browser. Subsequent
steps check the active document and use that view for its context menu,
camera, scale, lock, and publication controls.

| Published native scenario | Passed steps | Receipt |
| --- | ---: | --- |
| Toggle an origin plane while the active view is locked | 14 | [Run](published/toggle/cucumber.json) |
| Switch locked views after showing an origin plane | 12 | [Run](published/switch/cucumber.json) |
| Choose and restore annotation scale for an active view | 19 | [Run](published/scale/cucumber.json) |
| Restore a saved camera for an active view representation | 14 | [Run](published/camera/cucumber.json) |
| Cancel a custom annotation scale | 16 | [Run](published/cancel/cucumber.json) |
| Accept a custom annotation scale | 13 | [Run](published/accept/cucumber.json) |
| Correct an invalid custom annotation scale | 19 | [Run](published/invalid/cucumber.json) |
| Toggle the publication setting of an active copied view | 10 | [Run](published/publish/cucumber.json) |
| View bulk visibility commands leave the origin plane unchanged in an empty part | 11 | [Run](published/bulk/cucumber.json) |

The published inventory fell from [99 unresolved steps](before-bindings.json)
to [90 unresolved steps](after-bindings.json) across 100 scenarios.
The [input snapshot](../2026-11-24-model-browser-named-part-3896/snapshot.json)
contains the feature and the saved blank Part. All 25 snapshot hashes matched.

The runs used Linux X11, Xvfb, Openbox, and Vulkan Lavapipe. The initial run
used application base `ee5ec8fbe`. The successful runs used base
`15787ba7c`, local BDD changes, and existing local file-browser edits.
Most published runs used the first build of the shared empty-Part step.
After moving that step into the shared bindings, the published
[annotation-scale case](published/scale/cucumber.json) passed on rerun
and the pinned [quick-access case](pinned/quick-access/cucumber.json) passed.

See [#3896](https://github.com/Oblikovati/Oblikovati/issues/3896).
