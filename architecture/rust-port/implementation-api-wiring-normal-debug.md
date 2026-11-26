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
with the closed cube's visible front faces. The first run stopped at
the then-unsupported mesh-color request.

## Indexed mesh colors

The native owner now handles API 0.166.0 `viewport.setMeshColors`.
Face mode uses the B-rep face order carried through tessellation,
body composition, and assembly occurrence placement. Triangle mode
uses final scene triangle order. Both use the Go viewport's index
palette under the native shaded lighting. Turning off the mesh palette
restores the previous render mode, including normal debug if enabled.

New assembly display snapshots retain face ordinals. The reader accepts
missing ordinals. When such a cache is admitted by producer validation,
it supports triangle colors and face mode reports that the live model
is needed.

A disposable 50 mm cube completed both mode changes through MCP HTTP,
the Go Bridge, the C ABI, and the Rust desktop with `DISPLAY` unset.
The Wayland protocol log recorded an `xdg_surface`. The 392,140 pixels
identified as front-facing by normal debug showed three flat colors
in face mode and six flat colors in triangle mode. Both restoration
captures matched the shaded image pixel for pixel. The native
framebuffer regression also exercises switching modes while normal
debug is active. The Express scene regression checks that face
ordinals survive a save, reopen, and cached render.
