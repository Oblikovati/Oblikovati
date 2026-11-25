# Ribbon Display in Part and Drawing workspaces

Six current scenarios passed all 71 steps on Linux X11 with Xvfb, Openbox,
and Vulkan Lavapipe. The blank native Drawing stayed active through all four
ribbon modes. Switching between the native Part and Drawing retained their
compact ribbon modes. The remaining cases verified temporary group changes,
Annotate, and selecting the panel titles and buttons policies.

The runs used the published native files and existing local GUI filesystem,
opening and saving edits. The published [input checksums](current-inputs.sha256)
matched the isolated copies. See the [scenario inventory](scenario-inventory.json)
and individual reports. Issue #3978 tracks strict build and runtime provenance.
