# Edit and retain Extrude Taper A

The 8 mm extrusion starts with a 40 mm × 30 mm rectangle and parallel
walls. The +5° edit keeps the start cap and widens the farther cap by
twice the product of the depth and the tangent of 5°. The native test
checks both cap sizes and the shaded preview before accepting the edit.
The committed body retains the same edge keys and an 8 mm depth.

Undo restores the untapered shape. Redo and package reopen restore the
widened cap and the saved 5° value. The sign control previews the narrower
−5° shape. Cancel restores the accepted +5° result and preserves the
document generation and edge keys.

| Scenario | Published Wayland | Published X11 | Pinned Wayland | Pinned X11 |
| --- | --- | --- | --- | --- |
| 07 Accept +5° | [JUnit](wayland/current-07.xml) | [JUnit](x11/current-07.xml) | [Cucumber](wayland/pinned-07.json) | [Cucumber](x11/pinned-07.json) |
| 08 Undo, Redo, and reopen | [JUnit](wayland/current-08.xml) | [JUnit](x11/current-08.xml) | [Cucumber](wayland/pinned-08.json) | [Cucumber](x11/pinned-08.json) |
| 09 Reverse and cancel | [JUnit](wayland/current-09.xml) | [JUnit](x11/current-09.xml) | [Cucumber](wayland/pinned-09.json) | [Cucumber](x11/pinned-09.json) |

The certified taper test also checks ±5° in both normal extrusion
directions, the analytic volume, and the remapped locations and identities.
The typed `features.add` test verifies creation of a widened solid through
the public command.
