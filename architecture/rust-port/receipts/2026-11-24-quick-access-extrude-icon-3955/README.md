# Native Quick Access Extrude icon

Four scenarios passed individually with 44 steps. An added Extrude shortcut
uses the existing compact glyph and exposes its command name on hover.
The native tests find that glyph in the toolbar draw list. They exercise
adding and removing it, inserting a visible divider, invoking it with
Model or View active, and cancelling plane selection.

| Scenario | Steps | Evidence |
| --- | ---: | --- |
| Add and remove a ribbon command through its context menu | 15 | [Cucumber](icon-ribbon-toggle/cucumber.json) |
| Add a separator from a toolbar command context menu | 8 | [Cucumber](icon-separator/cucumber.json) |
| Invoke an added command from quick access in an empty part | 10 | [Cucumber](icon-invoke-empty/cucumber.json) |
| Invoke an added shortcut while another ribbon tab is active | 11 | [Cucumber](icon-invoke-view/cucumber.json) |

The native runs used base revision `915468584733a1fdbac0887cdec20547ad7cf19d`
with the compact icon and direct image checks, plus existing local GUI
filesystem, opening, and saving edits. The environment was Linux x86-64,
Xvfb, Openbox, and Vulkan Lavapipe. The [input hashes](current-inputs.sha256)
matched the shared source and isolated copy after the runs. Strict build
and runtime provenance remains open under #3978.
