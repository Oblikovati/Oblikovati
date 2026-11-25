# Ribbon changes across sketches, tools, and Home

Seven current Ribbon Display scenarios passed all 75 steps on Linux X11
with Xvfb, Openbox, and Vulkan Lavapipe. The native extrusion's Sketch1 was
opened from its real browser branch and finished from Tools. The separately
published empty Sketch1 was finished from a temporary tabs-only Sketch ribbon.
The remaining cases verified Options dismissal, Measure staying open across a
tab switch, and Home and Part displaying their own Tools commands.

The matching builds included existing local GUI filesystem, opening and saving
edits. The published [input checksums](current-inputs.sha256) matched the
isolated copies. See the [scenario inventory](scenario-inventory.json) and
individual reports. Issue #3978 tracks strict build and runtime provenance.
