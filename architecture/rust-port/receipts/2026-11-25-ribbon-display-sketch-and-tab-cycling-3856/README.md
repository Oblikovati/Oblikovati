# Sketch ribbon and tab cycling

Seven current Ribbon Display scenarios passed all 86 steps on Linux X11
with Xvfb, Openbox, and Vulkan Lavapipe. The native Sketch1 edit stayed active
while switching between tabs-only and full ribbon display. The remaining cases
covered all display modes, the compact Options command list, and switching or
double-clicking tabs while temporary groups were open.

The matching builds included existing local GUI filesystem, opening and saving
edits. The published [input checksums](current-inputs.sha256) matched the isolated
copies. See the [scenario inventory](scenario-inventory.json) and individual
reports. Issue #3978 tracks strict build and runtime provenance.
