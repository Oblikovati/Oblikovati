# Ribbon display across Home and appearance changes

Eight current Ribbon Display scenarios passed all 94 steps on Linux X11 with
Xvfb, Openbox, and Vulkan Lavapipe. The cases verified tabs-only and temporary
Home and Part commands, dismissal of a panel menu during minimization, and
appearance changes across the published Part and the native Sketch1 edit.
The saved sketch stayed active and unchanged while toggling its command labels.

The matching builds included existing local GUI filesystem, opening and saving
edits. The published [input checksums](current-inputs.sha256) matched the isolated
copies. See the [scenario inventory](scenario-inventory.json) and individual
reports. Issue #3978 tracks strict build and runtime provenance.
