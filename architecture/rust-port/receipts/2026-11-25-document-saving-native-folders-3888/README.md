# Native Save As folders, navigation, and overwrite refusal

Ten current Document Saving scenarios passed all 110 steps on Linux X11
with Xvfb, Openbox, and Vulkan Lavapipe. The dialog used both published Parts
and the real subfolder. The runs checked row selection, Back, cancellation from
a subfolder, preview resets, navigation panes, Details and Tiles, and Save As
menu commands. Declining replacement preserved both native files byte for byte.

Each folder scenario started its own test process in a fresh writable copy of
`fixtures/cad/document-saving/SaveAsNativeFolder`. The BDD source and fixture
reference stayed at the absolute isolated source path. The process working
directory supplied the application's initial Save As folder, including on
reopen after cancellation.

The runs used the matching bindings and existing local GUI filesystem,
opening and saving edits. The published [input checksums](current-inputs.sha256)
matched the isolated copies. See the [scenario inventory](scenario-inventory.json)
and individual Cucumber and JUnit reports. Issue #3978 tracks strict build and
runtime provenance.
