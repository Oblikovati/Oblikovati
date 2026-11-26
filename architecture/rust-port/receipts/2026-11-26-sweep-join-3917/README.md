# Edit a separate Sweep solid to Join

The published `sweep-13` scenario opens the block with a circular top
profile and an upward path. New Solid shows Solid2 and creates a second
body. Edit Feature switches the same Sweep to Join and accepts a
certified preview. The original body key is retained and the joined body
has eight faces, with its raised cap at the fixture's upper path end.

| Backend | Scenario evidence |
| --- | --- |
| Wayland on COSMIC | [9 steps](wayland/sweep-13/junit.xml) |
| X11 under Xvfb and Lavapipe | [9 steps](x11/sweep-13/junit.xml) |

The [input hashes](current-inputs.sha256) identify the scenario source
and native fixture.
