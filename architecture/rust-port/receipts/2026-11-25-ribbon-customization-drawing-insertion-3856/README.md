# Insert a 2-D Drawing entry after a selected row

Current case 59 passed all 13 steps in the native X11 window. The
test staged 2D Sketches and 3D Annotations, selected 2D Sketches,
and inserted 2-D Drawing immediately after it. The source list
removed the staged entry, and Cancel dismissed the draft.

2-D Drawing has a stable settings ID and can be arranged in the
customization dialog. Its applied button is disabled and explains
that the command is unavailable in this build. The case covers
the published list insertion and cancellation behavior.

The build used base `58c48d9f2` with the catalog entry and existing
local GUI filesystem, document-opening, and document-saving edits.
The run used a writable copy of the published Normal preferences.
All 52 focused GUI customization tests and GUI and Head Clippy passed.

[Cucumber](case-059/cucumber.json) · [JUnit](case-059/junit.xml) ·
[Scenario arguments](case-059/scenario-arguments.json) ·
[Source revision](case-059/source-revision.json)
