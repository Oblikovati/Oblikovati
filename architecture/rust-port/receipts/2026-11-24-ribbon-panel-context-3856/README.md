# Ribbon panel menus and sketch panels

The [full published baseline](baseline/cucumber.json) at `dd55e3d84`
passed 11 of 55 cases. Forty-four stopped at their first undefined step.
That run used the [49-file snapshot](../2026-11-24-ribbon-panel-visibility-3856/snapshot.json)
at `d53b12c1115aa35f38876e51976f7a28b3d43a43`.

| Native scenario | Steps | Pinned | Published |
| --- | ---: | --- | --- |
| Toggle Modify from the left-docked context menu | 15 | [Run](pinned/left/cucumber.json) | [Run](published/left/cucumber.json) |
| Preserve a hidden panel when docking left | 11 | [Run](pinned/preserve/cucumber.json) | [Run](published/preserve/cucumber.json) |
| Preserve a hidden panel when redocking at the top | 10 | [Run](pinned/redock/cucumber.json) | [Run](published/redock/cucumber.json) |
| Dismiss Show Panels without changing visibility | 12 | [Run](pinned/dismiss/cucumber.json) | [Run](published/dismiss/cucumber.json) |
| Replace the direct selector with an Options context menu | 10 | [Run](pinned/context/cucumber.json) | [Run](published/context/cucumber.json) |
| Toggle Layout during sketch editing | 16 | [Run](pinned/layout/cucumber.json) | [Run](published/layout/cucumber.json) |
| Toggle Parameters during sketch editing | 14 | [Run](pinned/parameters/cucumber.json) | [Run](published/parameters/cucumber.json) |
| Keep selector row focus when pressing Home and End | 11 | [Run](pinned/keyboard/cucumber.json) | [Run](published/keyboard/cucumber.json) |

The menu handoff checks the direct selector by popup identity and uses the
Show Panel Titles row to detect the context menu. The Show Panels tooltip
can remain after Escape. The left-docked case compares the Work Features
group's position before hiding Modify and after restoring it. The Sketch
Layout case checks the positions of its neighboring captions after removal.

The focused runs used application base `3ed3abd45` with the local test
changes and pinned source `b576445b98cffd686dcc05aa8c017de2a591c8c8` or the
published snapshot above. They ran on X11 with Xvfb, Openbox and Vulkan
Lavapipe. The [published binding inventory](published-bindings.json)
reports 195 unresolved steps, down from 217 in the preceding batch.
Eight pinned declarations gained R0 records.

See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).
