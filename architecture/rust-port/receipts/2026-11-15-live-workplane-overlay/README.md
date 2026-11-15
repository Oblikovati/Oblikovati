# Live work-plane overlay

On 2026-11-15, the standalone desktop built from merge commit
`af813ba1c01b28990e1c5394df914656729f7695` ran on Linux, Xvfb and Vulkan
Lavapipe. The executable SHA-256 was
`dc6423ddd44a215b5acce1cd301226a1fbde48ba24a09e08159fc5157b0be483`.
The MCPBridge library SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.
The application window and its Xvfb server were closed after the run.

The input was a copy of `../2026-11-15-live-parameter-cube/cube.opd`. XTest
sent OS mouse and keyboard input to the 1900 by 1100 window. MCPBridge supplied
screenshots and read-only work-plane queries.

1. Click Plane, choose the XY Plane browser row, and accept the 10 mm offset.
2. The created screenshot shows the translucent plane and its border; the
   `list_work_planes` result names Work Plane1 at `[0, 0, 1]` cm.
3. Press Ctrl+Z and Ctrl+Y. The screenshots show the overlay disappear and
   return. The query returns no authored plane after undo and the original
   `datum/v1/2` reference after redo.

An initial query used an incorrect MCP tool name and returned an error. The
subsequent `list_work_planes` queries produced the saved JSON reports. This
run does not verify viewport picking of planes, assembly overlays, visibility
editing, or other platforms. The frame-loop tests cover save and reopen. This
is a live diagnostic, not a verified BDD receipt (#3978).
