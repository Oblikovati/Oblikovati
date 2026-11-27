# Document Opening on the native desktop

The first three current Document Opening scenarios passed all 26 steps on
Linux X11 with Xvfb, Openbox, and Vulkan Lavapipe. The Home action opened
`single-occurrence.oad` in the Assembly workspace and loaded its referenced
component. Cancelling the dialog preserved the exact active extrusion part
and tab count. With `PartOne.opd` active, the picker showed the extrusion
part's preview, part options, and file details, including the original byte
size and the available Author and Part Number fields. Cancelling kept
`PartOne.opd` active.

The runs used application base `de01e6ad5`, the bindings in this change,
and existing local GUI filesystem, opening and saving edits. The shared
scenario source is unversioned. The published [input checksums](current-inputs.sha256)
matched the isolated copies used for each run. See the
[scenario inventory](scenario-inventory.json) and individual Cucumber and
JUnit reports. Issue #3978 tracks strict build and runtime provenance.
