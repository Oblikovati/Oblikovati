# Quick Access shortcut across Part documents

The scenario passed all 11 steps with the supplied PartOne, PartTwo, and
Assembly files. The native toolbar hid the Extrude shortcut while Assembly
was active and showed it again on both Part tabs. PartTwo remained active.

The [Cucumber report](part-assembly-first/cucumber.json) and
[final frame](part-assembly-first/final.png) came from base revision
`30bc170a20c42ed899aa058073ad31dc1ed22efb` with the named-fixture bindings
and existing local GUI filesystem, opening, and saving edits. The
environment was Linux x86-64, Xvfb, Openbox, and Vulkan Lavapipe.
The [input hashes](current-inputs.sha256) matched the shared source and
its isolated copy after the run. Strict build and runtime provenance
remains open under #3978.
