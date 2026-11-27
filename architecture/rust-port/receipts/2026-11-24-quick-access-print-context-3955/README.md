# Quick Access Print setting across document contexts

The native scenario passed all 14 steps with the supplied PartOne and
Assembly files. Both customization menus began with Print unchecked.
Selecting Print for the Part kept the Assembly setting unchecked.
Returning to the Part restored the checked setting and its toolbar control.

The [Cucumber report](print-context-native/cucumber.json) and
[final frame](print-context-native/final.png) came from base revision
`f642f090d4646b2677e88e7c8fca4561a274f5b1` with the named-fixture bindings
and existing local GUI filesystem, opening, and saving edits. The
environment was Linux x86-64, Xvfb, Openbox, and Vulkan Lavapipe.
The [input hashes](current-inputs.sha256) matched the shared source and
its isolated copy after the run. Strict build and runtime provenance
remains open under #3978.
