# Current native ribbon keyboard navigation

All 18 current scenarios passed individually, with 220 steps, using
the published Part with its saved extrusion and sketch. The checks
covered tab and command key tips, unmatched letters, display and panel
menus, temporary and left-docked ribbon groups, and interaction with
view rotation, browser navigation, and face selection.

The panel selector case uses the same full 3D Model setup as the
model-browser navigation case. The Tools selection case picks its face
on the published Part. The two-letter Hole command opened
its properties panel and cancelled without creating a feature.

The builds used base revision `5d8d516b5f37aff62f8a79140a2fe7505f4baee3`, the
current fixture binding where required, and existing local GUI
filesystem, opening, and saving edits. The native test environment
was Linux X11, Xvfb, Openbox, and Vulkan Lavapipe. The feature and
fixture hashes matched the shared source and isolated copy before
and after the runs. Strict build and runtime provenance remains open
under #3978. See the [combined report](cucumber.json),
[scenario inventory](scenario-inventory.json), and [input hashes](current-inputs.sha256).
