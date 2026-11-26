# Viewport normal debugging

The API 0.166.0 `viewport.setNormalDebug` method controls the native
viewport. It uses Martensite's `SideColorsV2` policy for unlit
double-sided triangles. Rasterized front faces are green and back faces
are red. Disabling the mode restores the original shaded presentation.
Camera state, model recipes, and sketch and datum overlays stay intact.

The native Wayland framebuffer regression sent the typed request over
the window dispatcher. It verified green and red on two triangles with
opposite winding, a new scene generation, the same camera generation,
and an exact return to the shaded framebuffer after disabling the mode.

The MCP normals driver captured a 50 mm cube on native Wayland with the
installed Bridge. The shaded and restored captures had no debug-green
pixels. The normal capture had 392,140 debug-green pixels, consistent
with the closed cube's visible front faces. The driver's later
`viewport.setMeshColors` request still reached an unsupported method.
