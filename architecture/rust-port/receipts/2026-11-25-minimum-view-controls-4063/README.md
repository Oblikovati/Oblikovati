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

The window-frame scenario reached 640 × 480 under Openbox and tint2.
It failed to maximize from the title bar on both the updated and
baseline viewport layouts.
[Updated layout](window-frame.json) · [Baseline](window-frame-baseline.json)

Verification used Linux X11, Xvfb and Vulkan Lavapipe. The build used
`239996a37` with the control placement changes and pending local
filesystem and Ribbon Customization edits.
[Published input hashes](current-inputs.sha256)
