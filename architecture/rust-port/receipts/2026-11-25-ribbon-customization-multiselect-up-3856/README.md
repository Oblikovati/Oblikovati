# Keep two selected rows in place when ordering up

Case 44 passed all 14 steps in the native X11 window. After a three-command
transfer, Shift+Down selected the lower two rows. Up preserved their order and
both highlights. Cancel discarded the draft.

The design migration exposed a font-dependent gap between the Large and Text
headings. Their positions now follow the checkbox columns. The native
checkbox regression in case 45 passed all 14 steps.

The build used base `2bdad5ecb` with the column fix, the new case bindings,
and the existing local GUI file edits. Case 44 passed with both the retained
inputs and the published visual references. The fixture and Gherkin hashes
match across the two runs. The refreshed visual reference and action icon
hashes are in [the published input manifest](published-support/inputs.sha256).

- Retained inputs. [Cucumber](case-044/cucumber.json) · [JUnit](case-044/junit.xml)
- Published inputs. [Cucumber](published-support/cucumber.json) · [JUnit](published-support/junit.xml) · [Final frame](published-support/final.png)
