# Native template catalog views and navigation

9 additional catalog scenarios passed individually, with 127 steps in total.
This brings native coverage of the current catalog basics feature to 12 of 21 scenarios.

| Scenario | Steps | Evidence |
| --- | ---: | --- |
| Display templates as details | 12 | [Cucumber](views2-details/cucumber.json) |
| Toggle template grouping by document type | 15 | [Cucumber](views2-grouping/cucumber.json) |
| Select an assembly template in list view | 16 | [Cucumber](views2-list/cucumber.json) |
| Navigate backward and forward through template folders | 15 | [Cucumber](views2-history/cucumber.json) |
| Reopen the chooser after canceling a selected template | 12 | [Cucumber](views2-reopen/cucumber.json) |
| Navigate home from a template folder | 15 | [Cucumber](views2-home/cucumber.json) |
| Change template icon size | 17 | [Cucumber](views2-icons/cucumber.json) |
| Create a part by double-clicking its template | 12 | [Cucumber](views2-double-click/cucumber.json) |
| Create a selected part template with Enter | 13 | [Cucumber](views2-enter/cucumber.json) |

Each case has its original JUnit report, step arguments, source record, and
final frame. The [input hashes](current-inputs.sha256) matched the shared
scenario source and its isolated copy after the runs.

The test binary used base revision `f0ebed98717d03ba14b1d56ba1c4c9082369a242`
with the catalog bindings in this change and existing local GUI filesystem,
opening, and saving edits. The runs used Linux x86-64, Xvfb, Openbox, and
Vulkan Lavapipe. Martensite was pinned to
`bef25f47548431c8f128f08f66a1f719bcee4982`.

The workspace gate reported 2,183 scenarios missing required behavior-ledger
fields. Strict build and runtime provenance remains open under #3978.
