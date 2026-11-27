# Transfer all visible sketch-exit commands

Current case 97 passed all 16 steps in the native X11 window. Ctrl+A
highlighted the three visible rows. Transfer placed the focused Cancel
Sketch row first, followed by the separator and Finish Sketch in their
display order. It retained a reusable source separator and left Large
and Text unchecked on both commands. Cancel dismissed the pending edit.

The build used base `4d35f6551` with the transfer changes and existing
local GUI filesystem, document-opening, and document-saving edits.
The run used a writable copy of the published Normal preferences. The
12-step existing visible-row selection case, 49 focused GUI tests, and
GUI and Head Clippy also passed.

[Cucumber](case-097/cucumber.json) · [JUnit](case-097/junit.xml) ·
[Scenario arguments](case-097/scenario-arguments.json) ·
[Source revision](case-097/source-revision.json)
