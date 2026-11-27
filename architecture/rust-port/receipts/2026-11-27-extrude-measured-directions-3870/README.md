# Measure Extrude previews in each direction

The published outline opens the rectangular profile on its sketch
plane and checks that the unedited preview has parallel walls. Each
example enters its own distance in the desktop properties.

| Direction | Preview extent relative to the sketch plane |
| --- | --- |
| Default | 0 to +8 mm |
| Flipped | −8 to 0 mm |
| Symmetric | −4 to +4 mm |
| Asymmetric | −4 to +8 mm |

The test measures the submitted triangles, compares both cap bounds,
checks that shaded geometry is visible, and verifies that OK is enabled
with no feature committed. Distance B uses the same test input events
on both window systems.

[Wayland JUnit](wayland/junit.xml) and [X11 JUnit](x11/junit.xml)
record all four examples. The pinned scenario revision predates this
outline.
