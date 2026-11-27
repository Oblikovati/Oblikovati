# Native template catalog creation and selection

9 additional native catalog scenarios passed individually, with 144 steps.
The current catalog basics feature has passed 21 of 21 scenarios.

| Scenario | Steps | Evidence |
| --- | ---: | --- |
| Create a part after navigating to an empty template root | 16 | [Cucumber](retained1-empty-root/cucumber.json) |
| Create the retained template after changing folders | 15 | [Cucumber](retained2-retained-create/cucumber.json) |
| Select templates with the secondary mouse button | 17 | [Cucumber](retained2-secondary/cucumber.json) |
| Click a folder highlighted by keyboard navigation | 14 | [Cucumber](retained2-highlight-click/cucumber.json) |
| Reverse an ascending template size sort in Details view | 14 | [Cucumber](retained2-reverse-sort/cucumber.json) |
| Retain template sort order when reopening the chooser | 15 | [Cucumber](retained3-retain-sort/cucumber.json) |
| Create from a selected template in a collapsed group | 22 | [Cucumber](retained3-collapsed-create/cucumber.json) |
| Press Enter after moving the template folder highlight with Up | 16 | [Cucumber](retained3-highlight-enter/cucumber.json) |
| Enable creation by selecting a template | 15 | [Cucumber](retained3-enable-create/cucumber.json) |

The [earlier three scenarios](../2026-11-24-document-creation-catalog-first-cases-3899/README.md)
and [nine chooser scenarios](../2026-11-24-document-creation-catalog-views-3899/README.md)
also ran on the current scenario and fixture hashes. The test bindings changed
between groups. The catalog commits after `c4597f77f` changed only test
bindings and evidence.
The [combined report](cucumber.json) records all 21 scenarios and 313 steps.

Each case has its JUnit report, step arguments, source record, and final frame.
The [input hashes](current-inputs.sha256) matched the shared source and its
isolated copy after the runs.

The test binaries used base revision `e79d96e2f0769a6db1fb012d867050247dd89106`
with the catalog bindings in this change and existing local GUI filesystem,
opening, and saving edits. The runs used Linux x86-64, Xvfb, Openbox, and
Vulkan Lavapipe. Martensite was pinned to
`bef25f47548431c8f128f08f66a1f719bcee4982`.

The full gate reported incomplete behavior-ledger fields and its later
workspace build was stopped to release the shared Cargo build lock.
Strict build and runtime provenance remains open under #3978.
