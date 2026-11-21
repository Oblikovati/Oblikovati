# Document tab reordering on a visible strip

The published and pinned runs at `ae0985ad7` both reached the two drag
scenarios and failed during setup. The new default asset dock reduced the
width of the strip. Part1 was outside the visible tabs, so the test could
not locate the intended drop position.

The reorder scenarios now load a saved workspace with the asset panels
closed and check that their neighbors are visible. Native X11 pointer
motion and button events move the tabs in the rendered application.
The narrow-strip test still uses the default layout. All four
[order and overflow tests](overflow.log.gz) passed.

The focused run used the published source at SHA-256
`ee2022e13728cfe8fcda1ef61e60b25f01c308816e6860da91b6dceac3430de9`.
It passed both scenarios and all 17 steps on local changes based on
`5e9cad7d0`, using Xvfb, Openbox and Vulkan Lavapipe. See the
[Cucumber report](cucumber.json).

- The [drawing](Reorder-an-active-drawing-tab.png) is active again in its
  original tab position, with its sheet and browser visible.
- The [part](Reorder-an-inactive-part-tab.png) has moved to the left and is
  active, with the drawing still open.
