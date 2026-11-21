# Live parameter-driven cube

On 2026-11-15, the standalone desktop ran under Linux Xvfb with Vulkan
Lavapipe and the real MCPBridge. The application binary was built from
`6aa2c4909ca96eade3f24bc95e613d00c397714e`; its SHA-256 was
`d3a42e76ca7b160791c384438c686afa3c396504ac538fafa66d4f97062388af`.
The bridge library SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.
The application, bridge listener, and Xvfb server were closed after the run.

XTest sent operating-system mouse and keyboard events. MCPBridge was used for
read-only queries and the PNG captures; it did not create or edit model data.
The window was resized to 1900 × 1100 at root-screen origin `(0, 0)`.

1. Create a metric Part and add the user parameter `size = 10 mm` in Parameters.
2. Start an XY sketch. Draw a rectangle and enter `size` for both dimensions.
   Two near-origin attempts retained two DOF and were undone. Draw a rectangle
   away from the origin, then use Coincident to join a corner to the origin.
   The bridge reported zero DOF, two horizontal, two vertical, and one
   coincident constraint. The two driving dimensions referenced `size`.
3. Finish the sketch, open Extrude, enter `size` for Distance A, preview, and
   accept. The bridge reported one extrusion and one body. The eight vertices
   spanned approximately 1 cm on each axis; all six face areas were about
   100 mm².
4. Change `size` to `20 mm` in Parameters. The sketch remained at zero DOF.
   The vertices spanned approximately 2 cm on each axis, the six original face
   keys measured about 400 mm² each, and all face, edge, and vertex key sets
   matched the 10 mm state.
5. Undo and redo with Ctrl+Z and Ctrl+Y. The six original face keys measured
   about 100 mm² after undo and 400 mm² after redo. The full topology key sets
   matched at both steps.
6. Save as `/tmp/obk-live-cube/cube.opd`, close with Ctrl+W, and reopen with
   Ctrl+O. The bridge reported one body, the same feature, zero DOF, dimensions
   `d6 = size` and `d7 = size`, and `size = 20 mm`. All topology keys matched;
   all original face keys measured about 400 mm². Body volume was approximately
   8000 mm³. The saved recipe links the extrusion displacement to the `size`
   parameter. The reviewed `cube_isometric.png` shows the reopened solid.

`keys10.json`, `keys20.json`, and the check outputs retain the measured values.
`cube.opd` is the saved test part. The copied scripts show the XTest driver,
MCPBridge probe, launcher, and six-face checks. Screenshots were captured by
MCPBridge and reviewed. This is a manual live integration check on one Linux
configuration, not a generated BDD receipt or a general kernel qualification.
The solver's spatial and angular precision certificate remains open (#3928).
