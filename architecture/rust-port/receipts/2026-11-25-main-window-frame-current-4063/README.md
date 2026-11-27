# Current native Main Window Frame

All 11 current declarations passed, with 20 executed cases and 169 steps.
The Home and Part examples checked the application bar and accessible
window controls. The taskbar examples preserved the active Part and its
view from both restored and maximized windows. Eight perimeter examples
checked cursor shapes, resized bounds, and the fixed opposite boundary.
The minimum-size case retained the saved Part and reachable controls.

The two Close cases edited a disposable copy of the published extrusion
Part. Both the application bar and desktop shortcut opened the
save-changes prompt. Cancel kept the edited document open and preserved
the file bytes. See the [custom Close report](case-10/cucumber.json)
and [desktop Close report](case-11/cucumber.json).

The builds used base revision `a30d24618c89387a08b0fb9a5981263e52b80f58`, the
scenario bindings for each run, and existing local GUI filesystem,
opening, and saving edits. The private Linux X11 desktop used Xvfb,
Openbox, tint2, AT-SPI, and Vulkan Lavapipe. The source and fixture hashes
matched the shared source and isolated copies before and after the runs.
Strict build and runtime provenance remains open under #3978. See the
[combined report](cucumber.json), [scenario inventory](scenario-inventory.json),
and [input hashes](current-inputs.sha256).
