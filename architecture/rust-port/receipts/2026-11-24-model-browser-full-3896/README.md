# Published Model Browser feature-wide run

The [published run](before/cucumber.json) attempted all 100 scenarios on
application base `7de9a444a` with the existing local file-browser edits.
It used Linux X11, Xvfb, Openbox and Vulkan Lavapipe, and the
[published source snapshot](../2026-11-24-model-browser-named-part-3896/snapshot.json).

| Scenario outcome | Count |
| --- | ---: |
| Passed | 54 |
| Stopped at an undefined step | 44 |
| Failed at a bound step | 2 |

The bound failures were the Favorites add-pane menu and view-mode
interactions. Both have since passed in focused native runs. Many of
the fixtures that stopped at undefined steps also have newer focused
receipts. These counts describe the older executable. See the
[add-pane run](../2026-11-24-model-browser-favorites-dock-3896/README.md)
and the [view-mode run](../2026-11-24-browser-origin-and-favorites-fixture-3896/README.md).
