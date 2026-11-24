# Current native Docking System

All eleven current scenarios passed individually, with 111 steps.
They cover the edge preview, docking and resizing panels, joining
and switching tabs, floating and restoring panels, and retaining
the layout while switching between the two published Parts.

The restart scenario saved the left Model Browser and right
Appearance and Material tabs. It reopened the same Part and verified
the restored panel sizes, bounds, and selected browser tab. A second
check launched two independent desktop processes with one temporary
profile and compared the saved layouts and native screenshots.
See the [desktop restart summary](restart/desktop/restart-summary.json),
[before](restart/desktop/before.png), and [after](restart/desktop/after.png).

| Scenario | Steps | Native result |
| --- | ---: | --- |
| Dock a floating panel to an edge with a visible target preview | 8 | [Cucumber](preview/cucumber.json) |
| Dock a panel to each supported workspace edge | 11 | [Cucumber](edges/cucumber.json) |
| Resize a docked panel from its shared boundary | 8 | [Cucumber](resize/cucumber.json) |
| Dock and resize Appearance Browser on the right | 13 | [Cucumber](appearance/cucumber.json) |
| Tab, switch, close, and restore docked browser panels | 21 | [Cucumber](browser-tabs/cucumber.json) |
| Open a workspace panel from the dock tab bar | 9 | [Cucumber](tab-menu/cucumber.json) |
| Join panels in one dock region as selectable tabs | 10 | [Cucumber](join/cucumber.json) |
| Float a docked panel and dock it again | 8 | [Cucumber](float/cucumber.json) |
| Close a docked panel and restore it from the interface menu | 9 | [Cucumber](restore-panel/cucumber.json) |
| Retain a panel layout while switching active documents | 8 | [Cucumber](retention/cucumber.json) |
| Restore the saved panel arrangement after restarting the application | 6 | [Cucumber](restart/cucumber.json) |

The builds used base revision `74d10f4f7a226d9a18bfe9cfc7b386bcb74f57aa`, the
scenario bindings for each run, and existing local GUI filesystem,
opening, and saving edits. The environment was Linux X11 with Xvfb,
Openbox, and Vulkan Lavapipe. The restart used a 1440×900 display. The feature and fixture hashes matched
the shared source and isolated copy before and after the runs. Strict
build and runtime provenance remains open under #3978. See the
[combined report](cucumber.json) and [input hashes](current-inputs.sha256).
