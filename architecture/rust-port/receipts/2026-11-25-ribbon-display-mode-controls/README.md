# Ribbon display modes on published Parts

The first six current Ribbon Display scenarios passed all 79 steps on Linux
X11 with Xvfb, Openbox, and Vulkan Lavapipe. The runs used the published
Workspace Layout Part and blank metric Part. The generic Part setup retains
an already open native Part so it stays active during the checks.

The cases checked compact buttons, title-only and tabs-only modes, temporary
Windows and View commands, restored full groups, and the selected policy while
the full Tools ribbon stayed visible.

The matching builds included existing local GUI filesystem, opening and saving
edits. The published [input checksums](current-inputs.sha256) matched the
isolated copies. See the [scenario inventory](scenario-inventory.json) and
individual reports. Issue #3978 tracks strict build and runtime provenance.
