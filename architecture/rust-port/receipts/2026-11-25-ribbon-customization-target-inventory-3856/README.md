# Inspect Part and Assembly custom panel targets

Current case 21 passed all 13 steps in the native X11 window. It
opened the target selector with the published blank Metric Part active,
verified the required Part and Assembly labels, and confirmed that
Part Tools stayed selected. Escape dismissed the selector and retained
the empty draft. Close returned to the same Part.

Additional targets in this inspection are displayed as unavailable until
they support editing. The six supported Part targets remain editable.
Escape in an open target selector with pending changes still discards
the draft.

The build used base `b402ef7e9` with the target inventory and dismissal
changes plus existing local GUI filesystem, document-opening, and
document-saving edits. The run used a writable copy of the published
Normal preferences. The pending-edit Escape regression passed 16 steps,
the keyboard switching regression passed 18 steps, and GUI and Head
Clippy passed with warnings denied.

[Cucumber](case-021/cucumber.json) · [JUnit](case-021/junit.xml) ·
[Scenario arguments](case-021/scenario-arguments.json) ·
[Source revision](case-021/source-revision.json)
