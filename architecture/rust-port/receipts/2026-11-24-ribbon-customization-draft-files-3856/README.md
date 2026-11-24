# Native Ribbon Customization file choosers and pending edits

Issue #3856. The runs use behavior source
`b576445b98cffd686dcc05aa8c017de2a591c8c8` on Linux X11 with Xvfb,
Openbox, and Vulkan Lavapipe.

| Native scenario | Evidence |
| --- | --- |
| Cancel a file chooser while a ribbon edit is pending (Import and Export) | [Cucumber](pinned/chooser/cucumber.json) |
| Export an unapplied custom ribbon command | [Cucumber](pinned/export/cucumber.json) |
| Export customization settings to a new XML file | [Cucumber](pinned/fresh-export/cucumber.json) |
| Stage an import of freshly exported customization settings | [Cucumber](pinned/fresh-import/cucumber.json) |

Both chooser examples check that Cancel returns to Customize with the
staged Tools command, pending marker, and edit controls still visible.
The export scenario parses the XML for Part Tools commands and checks that
the draft stays open with its staged command and pending marker.

The [earlier chooser run](before/chooser/cucumber.json) stopped at undefined
steps. After binding the export steps, the
[run with the old export behavior](regression-red/export/cucumber.json)
wrote the staged Tools command to XML and failed the file-content assertion.

The [binding inventory](after-bindings.json) has 422 undefined steps across
126 scenarios, down from [447](../2026-11-24-ribbon-customization-entry-points-3856/after-bindings.json).
