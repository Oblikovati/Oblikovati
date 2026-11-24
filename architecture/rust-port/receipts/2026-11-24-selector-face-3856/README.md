# Face selection after dismissing the ribbon selector

The [pinned](before/pinned/face/cucumber.json) and
[published](before/published/face/cucumber.json) cases reproduced a
right-click that dismissed the Tools selector without selecting the face.
The application now sends that press through native face picking and
suppresses the face menu for the same gesture. The selected face's
mini toolbar appears near the pointer.

| Native scenario | Steps | Pinned | Published |
| --- | ---: | --- | --- |
| Right-click a solid face with the selector open | 7 | [Run](after/pinned/face/cucumber.json) | [Run](after/published/face/cucumber.json) |
| Toggle Modify from the right-docked selector | 12 | [Run](after/pinned/right/cucumber.json) | [Run](after/published/right/cucumber.json) |

The right-docked case also verifies the second-click dismissal fix from
`4f0b56990`. The [ordinary X11 face context-menu test](normal.log)
passed after the face-pick change.

The face runs used app base `92052efce` with the local gesture change.
All runs used X11, Xvfb, Openbox and Vulkan Lavapipe. The pinned source
was `b576445b98cffd686dcc05aa8c017de2a591c8c8`, and the published run used
the [49-file snapshot](../2026-11-24-ribbon-panel-visibility-3856/snapshot.json).
The [binding inventory](published-bindings.json) reports 107 unresolved
steps, down from 112. Both pinned cases gained R0 records.

See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).
