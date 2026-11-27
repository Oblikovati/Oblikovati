# Native Workspace Layout browser tools

Nine published cases passed 86 steps on Linux X11 with Xvfb, Openbox and
Vulkan Lavapipe. The Part cases verified browser menus, branch expansion,
search, no results, and searching from an adjacent dock tab. The Drawing
cases verified floating and redocking the Model Browser and switching to
the additional Log tab.

The test build at `2f7418cf6` included existing local GUI filesystem,
opening and saving edits. The [input hashes](current-inputs.sha256)
matched the isolated source and fixtures. The
[scenario inventory](scenario-inventory.json) identifies each report.
Issue #3978 tracks strict build and runtime provenance.
