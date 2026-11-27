# View controls at 640 × 480

The old navigation bar extended 31 pixels below the viewport at minimum
window size. The bar now moves beside the ViewCube when there is too
little height to stack them. The ViewCube stays in its chosen corner.
With room to stack, the bar still centers on the cube.
[Before](before.png) · [After](after.png)

The native X11 test resized to 640 × 480, verified containment, opened
the ViewCube menu, and started Orbit from the navigation bar. The GUI
suite passed 15 orientation tests, including short and tall layouts.

Four published scenarios passed all 50 steps.
[Upper corners](case-0.json) · [Lower corners](case-1.json)
· [Control size](case-2.json) · [Pan](case-4.json)

The two initial window-frame runs reached 640 × 480. Their temporary tint2
configuration placed a taskbar across the title bar, where it intercepted
the Maximize click. The scenario passed all six steps with the repository’s
private-desktop launcher, which positions the taskbar at the bottom.
[Initial updated layout](window-frame.json) · [Initial baseline](window-frame-baseline.json)
· [Rerun](../2026-11-25-orientation-options-4063/README.md)

Verification used Linux X11, Xvfb and Vulkan Lavapipe. The build used
`239996a37` with the control placement changes and pending local
filesystem and Ribbon Customization edits.
[Published input hashes](current-inputs.sha256)
