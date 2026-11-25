# Toggle Explore from the expanded Modify title

All 17 steps in current case 48 passed in the native X11 window. With
the published solid Part open, the test used the lower Modify title
to open Show Panels. It checked the Explore check mark, hid Explore,
and measured Work Features moving left. It reopened the menu, checked
the cleared mark, restored Explore and the original Work Features
position, and verified that both selections closed the unpinned area.

Explore is a 3D Model panel with the existing Measure command. The
scenario checks panel visibility and reflow, and makes no geometry edit.
The title-menu selection uses the existing unpinned expansion dismissal.

The build used base `1c721000b` with the Explore changes and the existing
local GUI filesystem, document-opening, and document-saving edits.
The native appearance regression passed all 18 steps, and the pinned
expansion check also passed. GUI and Head Clippy passed with warnings
denied.

[Cucumber](case-48/cucumber.json) · [JUnit](case-48/junit.xml) ·
[Scenario arguments](case-48/scenario-arguments.json) ·
[Source revision](case-48/source-revision.json)
