# Orientation Options and the application frame

At 640 × 480 with 200% text scaling, Orientation Options clipped
selector labels and scrolled OK and Cancel with the settings. The
settings now scroll inside the dialog, with room for the full selector
labels and a fixed action row below. The native test scrolled to Angle
of North, checked that OK stayed in place, and clicked OK to close.
[Before](before.png) · [After](after.png)

The minimum-size Main Window Frame scenario passed all six steps with
the repository's private X11 desktop launcher. The earlier failure
came from a temporary tint2 configuration that covered the title bar
and intercepted the Maximize click. Openbox with the repository's
bottom taskbar also restored the application from minimization. All 20
published Main Window Frame cases passed, covering 169 steps.
[Main Window Frame report](main-window-frame.json)

The native scenario checks count Quick Access command icons separately
from the Oblikovati brand mark. They read the active document in the
redesigned title, recognize uppercase group captions, and use the rendered
control bounds for pointer input.

All 23 published Viewport Orientation scenarios passed, covering 253
steps. All 18 native orientation tests and the Quick Access row check
passed. On a separate 3840 × 2160 Xvfb display, the native DPI test
verified live scaling of the main window and a detached drawing.
[Viewport Orientation report](viewport-orientation.json) · [Published input hashes](current-inputs.sha256)

Verification used X11, Xvfb and Vulkan Lavapipe. The build used
`4617d26df` with these dialog and scenario changes and existing local
filesystem and Ribbon Customization edits.

The workspace gate reported 2,183 incomplete records in the existing
BDD evidence ledger. Clippy passed on retry after simplifying the
Sketch width condition. The retry stopped during the all-target build
to limit disk use, before it reached workspace coverage.
