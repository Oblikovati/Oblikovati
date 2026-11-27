# Native Workspace Layout frame and Model Browser

Eleven published cases passed 89 steps on Linux X11 with Xvfb, Openbox and
Vulkan Lavapipe. They covered the native Part frame, ribbon rows, View
checklist, browser visibility, float and redock, side docks, and resizing.

The test build at `2f7418cf6` included existing local GUI filesystem,
opening and saving edits. The [input hashes](current-inputs.sha256)
matched the isolated source and fixtures. The
[scenario inventory](scenario-inventory.json) identifies each report.
Issue #3978 tracks strict build and runtime provenance.
