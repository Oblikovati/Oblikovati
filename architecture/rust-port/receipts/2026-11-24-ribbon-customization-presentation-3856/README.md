# Native custom ribbon command presentation

Issue #3856. These runs use the pinned behavior source
`b576445b98cffd686dcc05aa8c017de2a591c8c8` on Linux X11 with Xvfb,
Openbox, and Vulkan Lavapipe.

| Native scenario | Evidence |
| --- | --- |
| Change the display of an applied custom command | [Cucumber](pinned/labeled/cucumber.json) |
| Display a large custom command without text | [Cucumber](pinned/icon-only/cucumber.json) |
| Separate two custom ribbon commands | [Cucumber](pinned/divider/cucumber.json) |
| Retain staged command display options when switching targets | [Cucumber](pinned/draft/cucumber.json) |

The [small and large](before/labeled/cucumber.json),
[icon-only](before/icon-only/cucumber.json), and
[separator](before/divider/cucumber.json) cases initially stopped at undefined
steps. With the bindings in place, the [separator probe](probe/divider/cucumber.json)
found only one icon in User Commands. The Normal appearance drew 3D
Annotations as a plain button. User Commands now draws its glyph through
the shared ribbon icon widget at the configured size.

The native assertions read icon rectangles and caption positions in the
rendered panel. The separator case also samples pixels between the icons
to check that the divider is visible. Go `head/ui/chrome_ribbon.go` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a` provides the reference small and
large caption layouts.

The [binding inventory](after-bindings.json) has 511 undefined steps across
126 scenarios, down from [534](../2026-11-24-ribbon-customization-selection-3856/after-bindings.json).
