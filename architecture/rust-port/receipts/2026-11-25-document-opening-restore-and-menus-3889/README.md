# Open dialog restoration and menus

Four current Document Opening scenarios passed all 44 steps on Linux X11
with Xvfb, Openbox, and Vulkan Lavapipe. The dialog returned to the original
child folder and drawing preview after cancelling a folder change. Maximize
and Restore kept the selected part and returned the dialog to its original
size and position. Dismissing the file-type and display menus preserved the
native filter, Details view, filename, and preview.

The runs used the corresponding bindings and existing local GUI filesystem,
opening and saving edits. The published [input checksums](current-inputs.sha256)
matched the isolated copies. See the [scenario inventory](scenario-inventory.json)
and individual Cucumber and JUnit reports. Issue #3978 tracks strict build and
runtime provenance.
