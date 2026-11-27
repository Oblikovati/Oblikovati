# Accept or cancel a viewport Extrude edit

The published fixture starts with an 8 mm named extrusion. The test
changes it to 6 mm, checks that the shaded replacement differs from the
current shape, and measures four 6 mm edges after accepting. Cancel on a
fresh copy preserves the visible 8 mm extrusion and its measured edges.
The pinned variants start from a 10 mm extrusion and preview a 20 mm
change. Both fixture variants render the replacement in the viewport.

The published taper cases use the same fixture and test code as the
[Part Create taper scenarios](../2026-11-27-extrude-taper-3870/README.md).
They check +5° acceptance and restoration after canceling a −5° preview.

| Scenario | Wayland | X11 |
| --- | --- | --- |
| Published distance edit | [JUnit](wayland/current-edit-distance.xml) | [JUnit](x11/current-edit-distance.xml) |
| Published distance cancellation | [JUnit](wayland/current-cancel-distance.xml) | [JUnit](x11/current-cancel-distance.xml) |
| Pinned distance edit | [Cucumber](wayland/pinned-edit-distance.json) | [Cucumber](x11/pinned-edit-distance.json) |
| Pinned distance cancellation | [Cucumber](wayland/pinned-cancel-distance.json) | [Cucumber](x11/pinned-cancel-distance.json) |
| Published taper edit | [JUnit](wayland/current-edit-taper.xml) | [JUnit](x11/current-edit-taper.xml) |
| Published taper cancellation | [JUnit](wayland/current-cancel-taper.xml) | [JUnit](x11/current-cancel-taper.xml) |
