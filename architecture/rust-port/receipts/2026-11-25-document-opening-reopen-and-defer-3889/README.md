# Existing tabs and drawing defer options

Four current Document Opening scenarios passed all 54 steps on Linux X11
with Xvfb, Openbox, and Vulkan Lavapipe. Picking the already-open extrusion
activated its existing tab and preserved the drawing tab. Switching Defer Mode
cleared and disabled unresolved-file skipping. Switching between two drawings
reset per-file options without opening either drawing.

The deferred-open case saved base views of the staged Part on both drawings.
The desktop displayed the deferred-update message and then restored the saved
sheet and projection lines from Drawing A. The drawing browser showed its view,
drawing view commands were unavailable, and the original Part tab stayed open.

The runs used the corresponding bindings and existing local GUI filesystem,
opening and saving edits. The published [input checksums](current-inputs.sha256)
matched the isolated copies. See the [scenario inventory](scenario-inventory.json)
and individual Cucumber and JUnit reports. Issue #3978 tracks strict build and
runtime provenance.
