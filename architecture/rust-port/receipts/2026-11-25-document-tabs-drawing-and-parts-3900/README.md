# Current drawing and Part document tabs

Eight current scenarios passed all 77 steps. They opened disposable
copies of the published drawing and Parts. The drawing retained its
collapsed resource branch across sketch editing. Reordering the
inactive profile Part rendered its rectangle after activation. The
detachment cases checked a separate native window, its sheet and
bottom tab, its context menu, and shared browser and ribbon controls.

When the drawing is the only attached document, detaching it now
shows Home in the main window. The drawing keeps its own sheet and
tab. With attached Parts, the main window still shows a Part.

The native runs used base revision `1e4537814349fa98f8fc3acae405fe002ae1603d`, the saved-fixture
bindings and detachment change available for each run, and existing
local GUI filesystem, opening and saving edits. The desktop used
Linux X11 with Xvfb, Openbox and Vulkan Lavapipe. The shared
[inputs](current-inputs.sha256) matched the isolated copies. Strict
build and runtime provenance remains open under #3978.
