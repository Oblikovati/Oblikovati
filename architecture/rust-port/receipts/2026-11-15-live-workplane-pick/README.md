# Live viewport work-plane selection

On 2026-11-15, the standalone desktop built from `1cc18f787fa44f91796b3a1f473c343077111236`
ran on Linux with Xvfb and Vulkan Lavapipe. The executable SHA-256 was
`89d965f3d90a247a51453139373d6dfb9f2d20866bd63ecacaf022c8922444d0`.
The MCPBridge library SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.
XTest sent OS mouse input to a 1900 by 1100 window. MCPBridge supplied the
screenshots and read-only queries. The window and Xvfb were closed afterward.

The input was a copy of `../2026-11-15-live-parameter-cube/cube.opd`.

1. Create Work Plane1 with a 10 mm offset from XY.
2. Click its visible viewport fill, away from the cube. The selected screenshot
   shows its themed border and browser row.
3. Click Plane and accept the selected base. MCPBridge reports Work Plane1 at
   `[0, 0, 1]` cm and Work Plane2 at `[0, 0, 2]` cm.
4. Click the nearer plane in the viewport and start a 2D sketch. The sketch
   screenshot shows the Sketch ribbon; `get_sketch` reports Sketch2 editing.

The first MCP connection attempt occurred before the bridge was ready and was
retried after its startup message. The scenario uses part work planes only;
assembly picking and other selection categories remain open. This is a live
diagnostic, not a verified BDD receipt (#3978).
