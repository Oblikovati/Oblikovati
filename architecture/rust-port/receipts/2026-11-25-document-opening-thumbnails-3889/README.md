# Native thumbnail and context-menu selection

Three current Document Opening scenarios passed all 38 steps on Linux X11
with Xvfb, Openbox, and Vulkan Lavapipe. Selecting the staged Part after a
Drawing updated the thumbnail highlight, filename, preview, and options.
Dismissing the Part context menu cleared the selection count and left the
filename and preview visible.

The selection rectangle started beside the row of drawings. It selected the
two Drawings while excluding the Part in the next row. Choosing one Drawing
again restored its own preview and loading options.

The runs used the corresponding bindings and existing local GUI filesystem,
opening and saving edits. The published [input checksums](current-inputs.sha256)
matched the isolated copies. See the [scenario inventory](scenario-inventory.json)
and individual Cucumber and JUnit reports. Issue #3978 tracks strict build and
runtime provenance.
