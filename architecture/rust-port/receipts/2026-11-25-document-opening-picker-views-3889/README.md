# Open dialog navigation, thumbnail mode, and sorting

Three current Document Opening scenarios passed all 27 steps on Linux X11
with Xvfb, Openbox, and Vulkan Lavapipe. Changing the navigation pane or
switching between Details and Tiles retained the published extrusion part,
its filename, and preview. The picker displayed the three published size-sort
fixtures with their original byte sizes. Reversing the sort restored increasing
size order while the large part stayed selected with its preview.

The runs used the matching bindings in this commit and existing local GUI
filesystem, opening and saving edits. They started from the drawing-preview
fix and staged case-specific bindings. The published
[input checksums](current-inputs.sha256) matched the isolated copies. See the
[scenario inventory](scenario-inventory.json), Cucumber reports, and JUnit
reports. Issue #3978 tracks strict build and runtime provenance.
