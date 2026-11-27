# Native ribbon icons from the designer handoff

The test build started from `71d421c7d` and included the ribbon icon refactor
and the existing local file-dialog changes. Native checks ran on Linux under
Xvfb, Openbox, and Vulkan Lavapipe.

## Checks

- The published scenario “Change ribbon command size modes while editing a
  sketch” passed all 18 steps. It exercised Small, Compact, Large, and
  Normal and confirmed that the sketch stayed active and unchanged.
  [Cucumber](cucumber.json) · [JUnit](junit.xml) ·
  [Scenario arguments](scenario-arguments.json) · [Final Sketch frame](Sketch.png)
- The native theme test retained the same modeled Part through all three
  palettes. [Dark](Model-Dark.png) · [Light](Model-Light.png) ·
  [High Contrast](Model-High-Contrast.png)
- The GUI library suite passed all 808 tests. Its rasterization checks
  exercised small and large SVGs at 100%, 150%, and 200% scale in all three
  palettes. Its frame checks verified that the Sketch and Model ribbons
  submitted the expected icon quads in their active layouts.
- Clippy passed for GUI and Head across all targets and features.

## Inputs

The supplied artwork accounts for all 37 new SVGs. The desktop copy of
Precise Input uses vector outlines for the lettering so the rasterizer can
draw it without host fonts.

| Published file | SHA-256 |
| --- | --- |
| `Design/tokens/oblikovati-tokens.css` | `3b89a79f7fc73bafb2791a7d3a5da8f06c1cd684642f764a6734f033d7da6a5e` |
| `features/application/ribbon-appearance/ribbon-appearance.feature` | `8f7a78d393c0fd4acf70c20b2aadb3a2cb7d6b2372fc0d8c0c34e6b3dfdb774b` |
| `fixtures/cad/planar-sketch-empty-xy.opd` | `d936ef6b165b4bcbe14e04ac1b08bb5b16c81fddb597e70acb781e753c07e9a8` |
