# Home Recent checkbox sizing

The list header, list rows, and grid cards use a 14 px checkbox
outline centered in each native checkbox's hit area. The geometry test
checks that the visible footprint stays 14 px when the hit area grows
with the text. Selection keeps the existing unchecked, checked, and
partial marks.

Three published scenarios and 34 steps passed under X11, covering the
header toggle, the partial mark, bulk actions, and selecting and restoring
filtered rows.
[Partial selection capture](home-partial-selection.png) ·
[Cucumber](checks/cucumber.json) · [JUnit](checks/junit.xml)

The List and Grid selection persistence scenario also passed all 11 steps.
[List and Grid scenario](grid/cucumber.json)
