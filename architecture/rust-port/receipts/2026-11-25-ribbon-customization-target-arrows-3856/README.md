# Navigate a staged separator with the arrow keys

Current case 102 passed all 14 steps in the native X11 window. The
target held Cancel Sketch, a separator, and Finish Sketch. Down moved
through the separator to Finish Sketch, and Up returned to the
separator. The rendered row order remained unchanged, and Cancel
dismissed the staged edits.

Case 100 passed all 13 target-paging steps again with the published
Part kept active. The earlier test setup opened a new Part after the
fixture, so the combined report now uses this replacement run. The
replacement keeps the previous pass count.
The 13-step source-paging regression also passed.

The build used base `6fc51dceb` with the arrow-key change and existing
local GUI filesystem, document-opening, and document-saving edits.
The runs used writable copies of the published Normal preferences.
GUI and Head Clippy and 49 focused GUI customization tests passed.

[Case 102 Cucumber](case-102/cucumber.json) · [JUnit](case-102/junit.xml) ·
[Scenario arguments](case-102/scenario-arguments.json) ·
[Source revision](case-102/source-revision.json)

[Case 100 Cucumber](case-100/cucumber.json) · [JUnit](case-100/junit.xml) ·
[Scenario arguments](case-100/scenario-arguments.json) ·
[Source revision](case-100/source-revision.json)
