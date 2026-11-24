# Native template group endpoints

The single current document creation scenario passed all 18 steps through
the native chooser. From an interior Part tile, Home selected Sheet Metal
(DIN) and End selected Standard (mm). In the Assembly group, End selected
Weldment (JIS) and Home selected Mold Design (DIN). Escape returned to the
original Part.

The [report](first/cucumber.json) and [final frame](first/final.png) came from
the test binary built at base revision `f0f946a907aab1c87aaa2deba6fb184526061dbd`
with the endpoint bindings and existing local GUI filesystem, opening, and
saving edits. The run used Linux x86-64, Xvfb, Openbox, and Vulkan Lavapipe.
The [input hashes](current-inputs.sha256) matched the shared source and its
isolated copy after the run. Strict build and runtime provenance remains
open under #3978.
