# Reorder a staged row after selecting a source command

Case 82 passed all 14 steps on both input snapshots. The test staged two
commands and selected 3D Annotations in the target. After selecting 2-D
Drawing in the source, Up moved 3D Annotations above 2D Sketches and kept
2-D Drawing available. The pending marker stayed visible until Cancel.

The native build used base `7f4807ad4`, the new case bindings, and the
existing local GUI file edits. The Gherkin and fixture hashes match across
the two runs. The [published input manifest](published-support/inputs.sha256)
records the updated design references. GUI and Head Clippy passed.

- Retained inputs. [Cucumber](case-082/cucumber.json) · [JUnit](case-082/junit.xml)
- Published inputs. [Cucumber](published-support/cucumber.json) · [JUnit](published-support/junit.xml) · [Final frame](published-support/final.png)
