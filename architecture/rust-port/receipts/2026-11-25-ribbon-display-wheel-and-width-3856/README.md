# Ribbon wheel navigation and width compaction

Five current Ribbon Display scenarios passed all 55 steps on Linux X11
with Xvfb, Openbox, and Vulkan Lavapipe. The published Part remained active
through wheel navigation at either end of the tabs, tabs-only navigation,
and changes to temporary groups. The wide Manage ribbon displayed the Insert
commands directly. The narrowed ribbon grouped them in a compact button while
the Part remained visible.

The matching builds included existing local GUI filesystem, opening and saving
edits. The published [input checksums](current-inputs.sha256) matched the isolated
copies. See the [scenario inventory](scenario-inventory.json) and individual
reports. Issue #3978 tracks strict build and runtime provenance.
