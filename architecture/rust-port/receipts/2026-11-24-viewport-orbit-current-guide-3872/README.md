# Current native free and constrained orbit

Two current scenarios passed individually, with 24 steps. Free rotation
changed the view during a drag and Done ended the command. Constrained
Orbit showed its guide, exited with Escape, and remained the marked menu
action when the menu opened again.

| Scenario | Steps | Results |
| --- | ---: | --- |
| Open and exit the free-rotation mode | 13 | [Report](free-ring/cucumber.json), [guide](free-ring/orbit-guide-visible.png) |
| Select and exit constrained orbit | 11 | [Report](constrained-ring/cucumber.json), [guide](constrained-ring/orbit-guide-visible.png) |

The baseline color scan picked up 69 pixels on the rotated solid after
Done. The revised assertion samples four predicted ring positions
using the viewport bounds. The guide captures also show the center
cross and radial marks.

The build used base revision `6357732414ec9d080f12f66d65f64b98ca43fb53`,
the revised guide assertion, and existing local GUI filesystem, opening,
and saving edits. The runs used Linux X11, Xvfb, Openbox, and Vulkan
Lavapipe. The [input hashes](current-inputs.sha256) matched the shared
source and the isolated copy. Strict provenance remains open under #3978.
