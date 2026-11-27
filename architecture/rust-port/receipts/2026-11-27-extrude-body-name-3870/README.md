# Name a New Solid from Extrude

Scenario 14 opens the published base-circle Part with one existing body.
The 8 mm New Solid preview retains the base and draws the separate boss.
The test enters `b` in Body Name, confirms the draft, and expands Solid
Bodies to check the `Solid1` and `b` rows. It verifies the original body
key, the six-face base, the three-face cylindrical solid, and the boss
cap dimensions. One Undo restores the original body and visible source
sketch. One Redo restores both bodies and the name `b`.

| Scenario source | Wayland | X11 |
| --- | --- | --- |
| Published fixture and feature | [JUnit](wayland/current-14.xml) | [JUnit](x11/current-14.xml) |
| Pinned feature at `b576445b9` | [Cucumber](wayland/pinned-14.json) | [Cucumber](x11/pinned-14.json) |

Both executions use the same native package as
[Extrude Join](../2026-11-27-extrude-join-3870/README.md).
