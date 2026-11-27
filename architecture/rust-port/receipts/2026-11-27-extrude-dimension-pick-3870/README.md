# Reference a sketch dimension from Extrude Distance A

The part has a rectangle and a separate circle with its diameter
parameter named `d2`. Measure in the Distance A menu prompts for a
dimension. Clicking the 3 mm annotation inserts `d2` and previews just
the rectangle at a depth of 3 mm. A repeat pick inserts a second
reference, producing `d2d2` and disabling OK. Cancel leaves the single
source sketch and creates no feature.

The pinned source describes the loops but predates its native fixture.
The pinned binding opens a bundled copy of the published part. Both
runs use the same annotation and geometry checks.

| Source | Wayland | X11 |
| --- | --- | --- |
| Published | [11 steps](wayland/published.xml) | [11 steps](x11/published.xml) |
| Pinned | [10 steps](wayland/pinned.xml) | [10 steps](x11/pinned.xml) |
