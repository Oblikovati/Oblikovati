# Native folder navigation and drawing row activation

Three current Document Opening scenarios passed all 36 steps on Linux X11
with Xvfb, Openbox, and Vulkan Lavapipe. Up, Back, and Forward navigated the
published parent and child folders while preserving the expected filename and
preview. Opening a selected folder kept the dialog open and added no tab.

The double-click case saved base views in both staged drawings, opened the
published assembly, and double-clicked Drawing B. Drawing B became active with
its sheet and saved view, and the assembly tab stayed available.

The runs used the corresponding bindings and existing local GUI filesystem,
opening and saving edits. The published [input checksums](current-inputs.sha256)
matched the isolated copies. See the [scenario inventory](scenario-inventory.json)
and individual Cucumber and JUnit reports. Issue #3978 tracks strict build and
runtime provenance.
