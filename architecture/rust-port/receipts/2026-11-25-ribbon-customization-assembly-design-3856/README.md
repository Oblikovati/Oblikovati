# Stage commands for Part Tools and Assembly Design

Current case 74 passed all 27 steps in the native X11 window. The
published Part remained active while the same command was staged
independently in both targets. Each target showed its own pending
marker and command availability. Cancel discarded both changes.
Reopening Customize showed both target lists empty.

Assembly Design settings apply to Oblikovati's existing Assemble
ribbon. XML version 7 identifies a file with an Assembly Design
entry. Part-only exports retain their prior version selection.
Other Assembly entries from the inspection case remain unavailable.

The native Apply and restart test showed the saved command on the
Assemble tab while Part Tools stayed empty. The original 13-step
target inventory case passed again.

The build used base `43c6e57c9` with Assembly Design support and existing
local GUI filesystem, document-opening, and document-saving edits.
The run used a writable copy of the published Normal preferences.
The 50 focused GUI customization tests and GUI and Head Clippy passed.

[Cucumber](case-074/cucumber.json) · [JUnit](case-074/junit.xml) ·
[Scenario arguments](case-074/scenario-arguments.json) ·
[Source revision](case-074/source-revision.json)
