# Live extrusion distance edit

On 2026-11-15, the standalone desktop built from
`9c07f056e` ran under Linux Xvfb and Vulkan Lavapipe with the real MCPBridge.
The binary SHA-256 was
`91bcce958e84093be390e2ecdf2a1c961e2313f9b5d37cf72c2f5549c1375b65`;
the bridge SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.
The application and Xvfb server were closed after the run.

The input was a copy of the saved 20 mm parameter-driven cube from the
[live cube journey](../2026-11-15-live-parameter-cube/README.md). XTest sent
operating-system mouse and keyboard input; MCPBridge supplied read-only queries
and PNG captures. The window was 1900 × 1100 at root-screen origin `(0, 0)`.

1. Right-click `Extrusion1` in the browser and choose **Edit Feature**.
2. Change Distance A from `size` to `30 mm` and preview. The bridge still
   reported the committed feature distance as 20 mm. Cancel; the distance and
   the `size`, `d6`, and `d7` parameter values remained 20 mm.
3. Reopen Edit Feature, enter `30 mm`, and accept. The same feature ID reported
   30 mm while `size`, `d6`, and `d7` remained at 20 mm. The body volume was
   approximately 12000 mm³. The six face, 12 edge, and eight vertex key sets
   matched those before the edit.
4. Ctrl+Z restored 20 mm and Ctrl+Y restored 30 mm. Save, close, and reopen
   through the UI. The feature still reported 30 mm, the parameter values
   remained 20 mm, and the topology key sets again matched the original.

The PNGs were captured by MCPBridge and reviewed. `isometric.png` shows the
reopened 20 × 20 × 30 mm solid. The copied scripts, query results, and saved
`cube.opd` retain this manual check. This test replaces the feature's distance
expression with a literal; it does not test taper or general edit support.
The sketch solve precision certificate remains open (#3928).
