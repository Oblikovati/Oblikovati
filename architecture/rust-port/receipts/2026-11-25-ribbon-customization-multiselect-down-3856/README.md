# Preserve the first two selected rows when ordering down

Case 60 passed all 12 steps on both the retained and published input sets.
The test staged the three authored commands, clicked 2-D Drawing and pressed
Shift+Down to select 2D Sketches as well. Down preserved the three-row order
and both highlights. Cancel dismissed the draft without leaving a custom
panel on Tools.

The native build used base `66370d3b3`, the case bindings, and the existing
local GUI file edits. The Gherkin and fixture hashes match across the runs.
The [published input manifest](published-support/inputs.sha256) records the
updated Design references.

- Retained inputs. [Cucumber](case-060/cucumber.json) · [JUnit](case-060/junit.xml)
- Published inputs. [Cucumber](published-support/cucumber.json) · [JUnit](published-support/junit.xml) · [Final frame](published-support/final.png)
