# Inspect 2D Sketch marking-menu sub-environments

Current case 117 passed all 11 steps in the native X11 window. With
2D Sketch selected, it opened the Sub-Environment selector and verified
the nine required labels and the rendered highlight on None Selected.
Escape dismissed the selector and left Apply and Cancel unavailable.
The listed sub-environments were inspection-only entries in this build.

The build used base `a6ef27488` with the selector changes and existing
local GUI filesystem, document-opening, and document-saving edits. The
run used a writable copy of the published Normal preferences.

[Cucumber](case-117/cucumber.json) · [JUnit](case-117/junit.xml) ·
[Scenario arguments](case-117/scenario-arguments.json) ·
[Source revision](case-117/source-revision.json)
