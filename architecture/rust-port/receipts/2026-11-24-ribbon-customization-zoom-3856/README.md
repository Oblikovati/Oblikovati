# Native Ribbon Customization Zoom commands

Issue #3856. The pinned behavior source is
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The desktop runs use Linux X11
with Xvfb, Openbox, and Vulkan Lavapipe.

[Transfer a selected command after reversing the source list](pinned/cucumber.json)
passed all 12 steps. The separator stayed above Zoom Window, Zoom Selected,
Zoom All, and Zoom. The selected 2D Sketches command transferred after the
sort, and Cancel left the Tools ribbon clear. The
[previous run](../2026-11-24-ribbon-customization-lists-3856/probe/sort-transfer/cucumber.json)
failed at the descending order assertion.

The four commands can be staged from All Commands or View Tab. Applying them
gives the custom panel the same Zoom handler as the navigation bar. Their
settings IDs are stable, and a Zoom Window setting is checked through XML
export and import. The native ribbon test invoked all four applied buttons.
Zoom, Zoom Window, and Zoom Selected started their respective prompts and
returned to Ready on Escape. Zoom All enlarged the distant block and kept
its outline inside the viewport. All seven pinned
[Viewport Zoom scenarios](viewport/cucumber.json) passed, covering 74 steps
through the original navigation controls.

The Go reference is `app/commands_standard.go` and
`app/zoom_window_test.go` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The ascending order scenario still expects 2-D Drawing and an unlabeled
entry. The pinned feature describes their placement in the list, and the
Go reference gives no matching command mapping for either entry.
