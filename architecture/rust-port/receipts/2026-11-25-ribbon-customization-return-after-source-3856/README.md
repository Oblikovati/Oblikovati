# Return and restage a command after selecting another source row

Case 83 passed all 17 steps on both input snapshots. After selecting 2-D
Drawing in the source, Remove returned the staged 2D Sketches row to the
source. Add transferred 2D Sketches back into the empty target. 2-D Drawing
stayed available, the pending marker was visible after removal, and Cancel
closed the draft with the Tools ribbon visible.

The native build used base `783160b68`, the return-step binding, and the
existing local GUI file edits. The Gherkin and fixture hashes match across
the runs. The [published input manifest](published-support/inputs.sha256)
records the updated design references.

- Retained inputs. [Cucumber](case-083/cucumber.json) · [JUnit](case-083/junit.xml)
- Published inputs. [Cucumber](published-support/cucumber.json) · [JUnit](published-support/junit.xml) · [Final frame](published-support/final.png)
