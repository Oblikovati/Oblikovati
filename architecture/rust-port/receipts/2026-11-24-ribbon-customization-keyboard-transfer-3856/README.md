# Native ribbon keyboard transfer

Issue #3856. These runs use the pinned behavior source
`b576445b98cffd686dcc05aa8c017de2a591c8c8` on Linux X11 with Xvfb,
Openbox, and Vulkan Lavapipe.

The [pinned keyboard transfer scenario](pinned/transfer/cucumber.json) checks
that Enter leaves the selected command in the source list with Apply
disabled. Tab moves focus to Add, and Space stages the selected 2D Sketches
command. Cancel closes the dialog.

The [initial run](before/transfer/cucumber.json) stopped at the unbound
Enter step. With the bindings in place, [the probe](probe/transfer/cucumber.json)
reached the transfer assertion and found 2D Sketches still in the source
list. The dialog now hands focus from the source list to its existing Add
button for plain Tab.

The [target selector keyboard](regression/target-keyboard/cucumber.json),
[staged Ctrl+A](regression/ctrl-a/cucumber.json), and
[pointer transfer](regression/pointer/cucumber.json) regressions exercise
the other input paths. The
[binding inventory](after-bindings.json) has 505 undefined steps across
126 scenarios, down from [511](../2026-11-24-ribbon-customization-presentation-3856/after-bindings.json).
