# Native Ribbon Panel Visibility menu and docking checks

Twenty-one current cases passed 247 steps. They covered independent panel
choices, changing dock positions, dismissing or switching menus, and
reflowing open additional commands. They also verified the Tools panel
inventory and its visibility settings.

The Linux X11 test build used `626b53a8b` with the native fixture aliases
documented in the [fixture receipt](../2026-11-25-ribbon-panel-visibility-fixture-3856/README.md)
and existing local GUI filesystem, opening and saving edits. The runs
used Xvfb, Openbox and Vulkan Lavapipe. The
[input hashes](current-inputs.sha256) matched the isolated source and
fixtures. Issue #3978 tracks strict build and runtime provenance.
