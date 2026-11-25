# Inspect the 3D Sketch marking-menu profile

Current case 121 passed all 14 steps in the native X11 window. Selecting
3D Sketch while the original Part remained open loaded the distinct
eight-slot profile. The case checked the commands in clockwise slot
order, the 3D sub-environment choices, and the rendered highlight on
None Selected. Escape dismissed the selector while keeping the 3D
profile selected and Apply unavailable.

The slot defaults use the benchmark's depicted positions and the Go
reference's command IDs. The sub-environments are inspection-only
choices. This case exercises profile configuration. It does not enter
3D Sketch or execute a 3D command on the canvas.

The native Environment selector and two-profile Apply and restart
regressions also passed. The focused model and persistence tests
reported four passes.

The build used base `01b02faf7` with the 3D profile changes and existing
local GUI filesystem, document-opening, and document-saving edits. The
run used a writable copy of the published Normal preferences.

[Cucumber](case-121/cucumber.json) · [JUnit](case-121/junit.xml) ·
[Scenario arguments](case-121/scenario-arguments.json) ·
[Source revision](case-121/source-revision.json)
