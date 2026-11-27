# Multi-file opening and tab identity

Three current Document Opening scenarios passed all 38 steps on Linux X11
with Xvfb, Openbox, and Vulkan Lavapipe. The desktop opened both selected
staged drawings and displayed their sheets and saved projections. The staged
Part tab remained open. Selecting two already open drawings reused their
existing tabs. Selecting one open and one closed drawing preserved the
existing tab and made the newly opened drawing active.

The runs used the corresponding bindings and existing local GUI filesystem,
opening and saving edits. The published [input checksums](current-inputs.sha256)
matched the isolated copies. See the [scenario inventory](scenario-inventory.json)
and individual Cucumber and JUnit reports. Issue #3978 tracks strict build and
runtime provenance.
