# Select the ends of a staged ribbon list

Case 101 passed all 12 steps on both input snapshots. Home selected the
first staged command, 2-D Drawing. End selected the last, 3D Annotations.
The order and six unchecked display options stayed in place. Cancel
dismissed the draft without applying the commands.

The native build used base `5fc7a5f62`, the target keyboard change, the
case bindings, and the existing local GUI file edits. The feature and
fixture hashes match across the runs. The
[published input manifest](published-support/inputs.sha256) records the
updated design references. All 52 GUI customization tests and GUI and
Head Clippy passed.

- Retained inputs. [Cucumber](case-101/cucumber.json) · [JUnit](case-101/junit.xml)
- Published inputs. [Cucumber](published-support/cucumber.json) · [JUnit](published-support/junit.xml) · [Final frame](published-support/final.png)
