# Default multi-character command aliases

The native keyboard editor showed the initial selectors, empty search,
populated command table, unchecked aliases checkbox, and disabled edit
controls. Current case 15 passed 15 of 16 steps. Close returned to the
published Part, whose Sketch1 was visible in its browser outside edit
mode. The final step requires an already active sketch edit. This
scenario remains open pending a fixture or setup correction.

With an actual sketch in edit, the separate native preference test
staged the checkbox, canceled it while preserving the sketch, then
applied it. It restarted the application, typed `RECTANG` into a new
sketch, and verified the four created rectangle edges. Explicit
aliases take priority. The opt-in defaults use Go command words for
supported desktop sketch and application actions.

The build used base `45c9bd34e` with the alias settings changes and
existing local GUI filesystem, document-opening, and document-saving
edits. The benchmark used a writable copy of the Normal preferences.
There were 14 passing keymap tests and 16 passing persistence checks.
The existing native custom-alias regression also passed.

[Cucumber](case-015/cucumber.json) · [JUnit](case-015/junit.xml) ·
[Scenario arguments](case-015/scenario-arguments.json) ·
[Source revision](case-015/source-revision.json)
